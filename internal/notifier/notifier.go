package notifier

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"text/template"
	"time"

	"cert-monitor/internal/checker"
	"cert-monitor/internal/config"
)

// Notifier dispatches alerts when certificates are expiring
type Notifier struct {
	cfg     *config.NotifyConfig
	webhook *template.Template
}

// New creates a new Notifier instance
func New(cfg *config.NotifyConfig) (*Notifier, error) {
	n := &Notifier{cfg: cfg}

	if cfg.Webhook.Enabled && cfg.Webhook.Body != "" {
		tmpl, err := template.New("webhook").Parse(cfg.Webhook.Body)
		if err != nil {
			return nil, fmt.Errorf("invalid webhook template: %w", err)
		}
		n.webhook = tmpl
	}

	return n, nil
}

// Send sends notifications for all results with issues
func (n *Notifier) Send(results []checker.Result) error {
	// Filter to only results that need attention
	var urgent []checker.Result
	for _, r := range results {
		if r.Issue != checker.IssueOK && r.Issue != checker.IssueNone {
			urgent = append(urgent, r)
		}
	}

	if len(urgent) == 0 {
		return nil
	}

	var errs []string

	if n.cfg.Webhook.Enabled {
		if err := n.sendWebhook(urgent); err != nil {
			errs = append(errs, fmt.Sprintf("webhook: %v", err))
		}
	}

	if n.cfg.Email.Enabled {
		if err := n.sendEmail(urgent); err != nil {
			errs = append(errs, fmt.Sprintf("email: %v", err))
		}
	}

	if n.cfg.Slack.Enabled {
		if err := n.sendSlack(urgent); err != nil {
			errs = append(errs, fmt.Sprintf("slack: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// sendWebhook sends an HTTP POST to the configured webhook
func (n *Notifier) sendWebhook(results []checker.Result) error {
	if n.webhook == nil {
		return fmt.Errorf("webhook template not configured")
	}

	var buf bytes.Buffer
	if err := n.webhook.Execute(&buf, results); err != nil {
		return fmt.Errorf("template execution: %w", err)
	}

	method := n.cfg.Webhook.Method
	if method == "" {
		method = "POST"
	}

	req, err := http.NewRequest(method, n.cfg.Webhook.URL, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range n.cfg.Webhook.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}

// sendEmail sends an email alert via SMTP
func (n *Notifier) sendEmail(results []checker.Result) error {
	var body strings.Builder
	body.WriteString("Subject: TLS Certificate Alert\n")
	body.WriteString("MIME-Version: 1.0\n")
	body.WriteString("Content-Type: text/plain; charset=\"utf-8\"\n")
	body.WriteString("\n")

	body.WriteString("TLS Certificate Alert\n")
	body.WriteString(strings.Repeat("=", 40) + "\n\n")

	for _, r := range results {
		emoji := "CRITICAL"
		if r.Issue == checker.IssueWarning {
			emoji = "WARNING"
		} else if r.Issue == checker.IssueInfo {
			emoji = "INFO"
		}

		body.WriteString(fmt.Sprintf("[%s] %s:%d\n", emoji, r.Host, r.Port))
		body.WriteString(fmt.Sprintf("  Days remaining: %d\n", r.DaysRemaining))
		body.WriteString(fmt.Sprintf("  Expires: %s\n", r.NotAfter))
		body.WriteString(fmt.Sprintf("  Issuer: %s\n", r.Issuer))
		body.WriteString(fmt.Sprintf("  CN: %s\n\n", r.CommonName))
	}

	addr := fmt.Sprintf("%s:%d", n.cfg.Email.Host, n.cfg.Email.Port)
	auth := smtp.PlainAuth("", n.cfg.Email.Username, n.cfg.Email.Password, n.cfg.Email.Host)

	err := smtp.SendMail(addr, auth, n.cfg.Email.From, []string{n.cfg.Email.To}, []byte(body.String()))
	if err != nil {
		// Fall back to TLS
		tlsConfig := &tls.Config{ServerName: n.cfg.Email.Host}
		err = sendMailTLS(addr, auth, n.cfg.Email.From, []string{n.cfg.Email.To}, []byte(body.String()), tlsConfig)
	}
	return err
}

// sendMailTLS sends email over a TLS connection
func sendMailTLS(addr string, auth smtp.Auth, from string, to []string, data []byte, tlsConfig *tls.Config) error {
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, addr)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, t := range to {
		if err := client.Rcpt(t); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	if err != nil {
		return err
	}
	w.Close()
	return client.Quit()
}

// sendSlack sends a Slack message via incoming webhook
func (n *Notifier) sendSlack(results []checker.Result) error {
	type slackField struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	type slackBlock struct {
		Type   string       `json:"type"`
		Text   string       `json:"text,omitempty"`
		Fields []slackField `json:"fields,omitempty"`
	}

	blocks := []slackBlock{
		{Type: "header", Text: "TLS Certificate Alert"},
	}

	for _, r := range results {
		blocks = append(blocks, slackBlock{
			Type: "section",
			Fields: []slackField{
				{Type: "mrkdwn", Text: fmt.Sprintf("*Host:* `%s:%d`", r.Host, r.Port)},
				{Type: "mrkdwn", Text: fmt.Sprintf("*Issue:* `%s`", r.Issue)},
				{Type: "mrkdwn", Text: fmt.Sprintf("*Days Left:* %d", r.DaysRemaining)},
				{Type: "mrkdwn", Text: fmt.Sprintf("*Expires:* %s", r.NotAfter)},
			},
		})
	}

	payload := map[string]interface{}{
		"text":   "TLS Certificate Alert",
		"blocks": blocks,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", n.cfg.Slack.Webhook, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}
	return nil
}

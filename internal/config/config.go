package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the application
type Config struct {
	Targets    []string       `yaml:"targets"`
	Check     CheckConfig    `yaml:"check"`
	Notify    NotifyConfig   `yaml:"notify"`
	Server    ServerConfig   `yaml:"server"`
	Logging   LoggingConfig  `yaml:"logging"`
}

// CheckConfig defines certificate check thresholds
type CheckConfig struct {
	WarnDays     int  `yaml:"warn_days"`
	CriticalDays int  `yaml:"critical_days"`
	Timeout      int  `yaml:"timeout"`
	Interval     int  `yaml:"interval"`
}

// NotifyConfig defines notification settings
type NotifyConfig struct {
	Enabled    bool          `yaml:"enabled"`
	Webhook    WebhookConfig `yaml:"webhook"`
	Email      EmailConfig   `yaml:"email"`
	Slack      SlackConfig   `yaml:"slack"`
}

// WebhookConfig holds webhook notification settings
type WebhookConfig struct {
	Enabled  bool   `yaml:"enabled"`
	URL      string `yaml:"url"`
	Method   string `yaml:"method"`
	Headers  map[string]string `yaml:"headers"`
	Body     string `yaml:"body"`
}

// EmailConfig holds email notification settings
type EmailConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
	To       string `yaml:"to"`
}

// SlackConfig holds Slack notification settings
type SlackConfig struct {
	Enabled bool   `yaml:"enabled"`
	Webhook string `yaml:"webhook"`
	Channel string `yaml:"channel"`
}

// ServerConfig holds HTTP server settings for daemon mode
type ServerConfig struct {
	Addr      string `yaml:"addr"`
	Port      int    `yaml:"port"`
	StaticDir string `yaml:"static_dir"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`
	File   string `yaml:"file"`
	Format string `yaml:"format"`
}

// Load reads and parses a YAML configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	// Apply defaults
	if cfg.Check.WarnDays == 0 {
		cfg.Check.WarnDays = 30
	}
	if cfg.Check.CriticalDays == 0 {
		cfg.Check.CriticalDays = 7
	}
	if cfg.Check.Timeout == 0 {
		cfg.Check.Timeout = 10
	}
	if cfg.Check.Interval == 0 {
		cfg.Check.Interval = 3600
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}

	return &cfg, nil
}

// SampleConfig returns a default configuration as a YAML string
func SampleConfig() string {
	return `# cert-monitor configuration
# TLS/SSL Certificate Expiration Monitor

# Targets: list of host:port to check
targets:
  - "google.com:443"
  - "github.com:443"
  - "cloudflare.com:443"
  - "letsencrypt.org:443"

# Check thresholds
check:
  warn_days: 30       # Warn if cert expires within 30 days
  critical_days: 7    # Critical alert if within 7 days
  timeout: 10         # Connection timeout in seconds
  interval: 3600      # Check interval in seconds (daemon mode)

# Notification settings
notify:
  enabled: true

  # Webhook notification (generic HTTP POST)
  webhook:
    enabled: false
    url: "https://your-webhook-endpoint.com/notify"
    method: "POST"
    headers:
      Authorization: "Bearer YOUR_TOKEN"
      Content-Type: "application/json"
    body: |
      {
        "host": "{{.Host}}",
        "port": {{.Port}},
        "days_remaining": {{.DaysRemaining}},
        "issue": "{{.Issue}}",
        "expires": "{{.NotAfter}}"
      }

  # Email notification
  email:
    enabled: false
    host: "smtp.example.com"
    port: 587
    username: "your-username"
    password: "your-password"
    from: "cert-monitor@example.com"
    to: "admin@example.com"

  # Slack notification
  slack:
    enabled: false
    webhook: "https://hooks.slack.com/services/XXX/YYY/ZZZ"
    channel: "#alerts"

# HTTP server settings (daemon mode)
server:
  addr: ":8080"
  port: 8080
  static_dir: "./public"

# Logging settings
logging:
  level: "info"   # debug, info, warn, error
  file: ""         # leave empty for stdout
  format: "text"   # text or json
`
}

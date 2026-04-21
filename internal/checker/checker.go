package checker

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"
)

// Issue severity levels
const (
	IssueNone     = ""
	IssueOK       = "ok"
	IssueInfo     = "info"
	IssueWarning  = "warning"
	IssueCritical = "critical"
	IssueExpired  = "expired"
	IssueError    = "error"
)

// Host represents a target to check
type Host struct {
	Host string // hostname or IP address
	Port int    // TLS port (default 443)
}

// Result holds the result of a certificate check
type Result struct {
	Host            string  `json:"host"`
	Port            int     `json:"port"`
	Issuer          string  `json:"issuer"`
	Subject         string  `json:"subject"`
	CommonName      string  `json:"common_name"`
	NotBefore       string  `json:"not_before"`
	NotAfter        string  `json:"not_after"`
	DaysRemaining   int     `json:"days_remaining"`
	Issue           string  `json:"issue"`
	SerialNumber    string  `json:"serial_number,omitempty"`
	SignatureAlg    string  `json:"signature_algorithm,omitempty"`
	KeyAlgorithm    string  `json:"key_algorithm,omitempty"`
	KeySize         int     `json:"key_size,omitempty"`
	IsWildcard      bool    `json:"is_wildcard"`
	SANs            int     `json:"san_count"`
	DNSNames        []string `json:"dns_names,omitempty"`
	Error           string  `json:"error,omitempty"`
}

// CheckHosts checks multiple hosts concurrently
func CheckHosts(hosts []Host, warnDays, criticalDays int) []Result {
	type resultChan struct {
		host string
		port int
		res  Result
	}

	ch := make(chan resultChan, len(hosts))

	for _, h := range hosts {
		go func(host string, port int) {
			r := checkHost(host, port, warnDays, criticalDays)
			ch <- resultChan{host: host, port: port, res: r}
		}(h.Host, h.Port)
	}

	var results []Result
	for range hosts {
		rc := <-ch
		results = append(results, rc.res)
	}
	return results
}

// checkHost performs the TLS check for a single host
func checkHost(host string, port int, warnDays, criticalDays int) Result {
	res := Result{
		Host: host,
		Port: port,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS10,
	})
	if err != nil {
		res.Issue = IssueError
		res.Error = err.Error()
		return res
	}
	defer conn.Close()

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		res.Issue = IssueError
		res.Error = "no certificates received"
		return res
	}

	cert := state.PeerCertificates[0]
	res.SerialNumber = formatSerial(cert.SerialNumber)
	res.SignatureAlg = cert.SignatureAlgorithm.String()
	res.NotBefore = cert.NotBefore.Format("2006-01-02")
	res.NotAfter = cert.NotAfter.Format("2006-01-02")
	res.Issuer = formatIssuer(cert)
	res.Subject = cert.Subject.String()
	res.CommonName = getCN(cert.Subject)

	// Wildcard detection
	if strings.HasPrefix(res.CommonName, "*") {
		res.IsWildcard = true
	}
	res.SANs = len(cert.DNSNames)
	res.DNSNames = cert.DNSNames

	// Key info
	if cert.PublicKey != nil {
		res.KeyAlgorithm = cert.PublicKeyAlgorithm.String()
		if keySize, ok := getKeySize(cert.PublicKey); ok {
			res.KeySize = keySize
		}
	}

	daysLeft := int(time.Until(cert.NotAfter).Hours() / 24)

	// SHA-1 deprecation warning
	if cert.SignatureAlgorithm == x509.SHA1WithRSA || cert.SignatureAlgorithm == x509.DSAWithSHA1 {
		res.Issue = IssueWarning
	} else if daysLeft < 0 {
		res.Issue = IssueExpired
		res.DaysRemaining = daysLeft
	} else if daysLeft <= criticalDays {
		res.Issue = IssueCritical
		res.DaysRemaining = daysLeft
	} else if daysLeft <= warnDays {
		res.Issue = IssueWarning
		res.DaysRemaining = daysLeft
	} else if daysLeft <= warnDays*2 {
		res.Issue = IssueInfo
		res.DaysRemaining = daysLeft
	} else {
		res.Issue = IssueOK
		res.DaysRemaining = daysLeft
	}

	return res
}

func formatIssuer(cert *x509.Certificate) string {
	for _, o := range cert.Issuer.Organization {
		return o
	}
	return cert.Issuer.String()
}

func getCN(name pkix.Name) string {
	return name.CommonName
}

func formatSerial(n *big.Int) string {
	return strings.ToUpper(n.Text(16))
}

func getKeySize(pub interface{}) (int, bool) {
	switch k := pub.(type) {
	case *rsa.PublicKey:
		return k.N.BitLen(), true
	case *ecdsa.PublicKey:
		return k.Curve.Params().BitSize, true
	}
	return 0, false
}

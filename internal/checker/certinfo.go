package checker

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"strings"
	"time"
)

// parseCertBundle extracts all certificates from a connection's peer certificates
func parseCertBundle(certs []*x509.Certificate) []CertificateInfo {
	var infos []CertificateInfo
	for _, cert := range certs {
		infos = append(infos, extractCertInfo(cert))
	}
	return infos
}

// extractCertInfo converts an x509 certificate to our CertificateInfo struct
func extractCertInfo(cert *x509.Certificate) CertificateInfo {
	cn := ""
	if len(cert.Subject.CommonName) > 0 {
		cn = cert.Subject.CommonName
	} else if len(cert.Subject.Names) > 0 {
		cn = cert.Subject.Names[0].Value.(string)
	}

	// Check if it's a wildcard certificate
	isWildcard := false
	for _, dnsName := range cert.DNSNames {
		if strings.HasPrefix(dnsName, "*.") {
			isWildcard = true
			break
		}
	}

	// Determine key algorithm and size
	keyAlg, keySize := parseKeyInfo(cert)

	// Extract DNS names (SANs)
	sans := make([]string, len(cert.DNSNames))
	copy(sans, cert.DNSNames)

	return CertificateInfo{
		Subject:        cert.Subject.String(),
		Issuer:        cert.Issuer.String(),
		CommonName:     cn,
		NotBefore:      cert.NotBefore,
		NotAfter:       cert.NotAfter,
		DaysRemaining:  daysUntilExpiry(cert.NotAfter),
		SerialNumber:   cert.SerialNumber.String(),
		SignatureAlg:   cert.SignatureAlgorithm.String(),
		KeyAlgorithm:   keyAlg,
		KeySize:        keySize,
		IsWildcard:     isWildcard,
		SANs:           sans,
		IsCA:           cert.IsCA,
	}
}

// parseKeyInfo extracts key algorithm and size from a certificate
func parseKeyInfo(cert *x509.Certificate) (algorithm string, size int) {
	switch pubKey := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		return "RSA", pubKey.N.BitLen()
	case *ecdsa.PublicKey:
		return "ECDSA", pubKey.Params().BitSize
	default:
		return "Unknown", 0
	}
}

// daysUntilExpiry calculates days until certificate expiration
func daysUntilExpiry(notAfter time.Time) int {
	days := time.Until(notAfter).Hours() / 24
	return int(math.Round(days))
}

// lookupHost resolves hostname to IP addresses
func lookupHost(host string) ([]string, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup failed for %s: %w", host, err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no addresses found for %s", host)
	}
	return addrs, nil
}

// encodeCertificatePEM encodes a certificate to PEM format
func encodeCertificatePEM(cert *x509.Certificate) string {
	pemBlock := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return string(pem.EncodeToMemory(&pemBlock))
}
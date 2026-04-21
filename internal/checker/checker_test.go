package checker

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"os"
	"sort"
	"strings"
	"testing"
)

func TestCheckHostLive(t *testing.T) {
	res := checkHost("google.com", 443, 30, 7)

	if res.Issue == IssueError && res.Error != "" {
		if strings.Contains(res.Error, "timeout") ||
			strings.Contains(res.Error, "i/o timeout") ||
			strings.Contains(res.Error, "connection refused") {
			t.Skip("google.com not reachable from this network")
		}
	}

	if res.NotAfter == "" {
		t.Error("expected NotAfter to be set")
	}
	if res.Issuer == "" {
		t.Error("issuer should not be empty for google.com")
	}
	if res.CommonName == "" {
		t.Error("common name should not be empty for google.com")
	}
}

func TestCheckHostSHA1(t *testing.T) {
	res := checkHost("sha1.badssl.com", 443, 30, 7)
	if res.Issue == IssueError {
		t.Skip("sha1.badssl.com not reachable")
	}
	if res.Issue != IssueWarning {
		t.Errorf("expected SHA-1 cert to trigger warning, got %s", res.Issue)
	}
}

func TestCheckHosts(t *testing.T) {
	hosts := []Host{
		{Host: "github.com", Port: 443},
		{Host: "cloudflare.com", Port: 443},
	}

	results := CheckHosts(hosts, 30, 7)

	validCount := 0
	for _, r := range results {
		if r.Issue != IssueError {
			validCount++
		}
	}
	if validCount == 0 {
		t.Skip("no hosts reachable for this test")
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DaysRemaining < results[j].DaysRemaining
	})
}

func TestCheckHostInvalid(t *testing.T) {
	res := checkHost("this-host-does-not-exist-xyz-12345.com", 443, 30, 7)

	if res.Issue != IssueError {
		t.Errorf("expected error for invalid host, got %s", res.Issue)
	}
	if res.Error == "" {
		t.Error("expected error message for invalid host")
	}
}

func TestGetKeySize(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	size, ok := getKeySize(&rsaKey.PublicKey)
	if !ok {
		t.Error("expected to detect RSA key size")
	}
	if size != 2048 {
		t.Errorf("expected RSA key size 2048, got %d", size)
	}

	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ECDSA key: %v", err)
	}
	size, ok = getKeySize(&ecdsaKey.PublicKey)
	if !ok {
		t.Error("expected to detect ECDSA key size")
	}
}

func TestResultSort(t *testing.T) {
	results := []Result{
		{Host: "a.com", Port: 443, DaysRemaining: 5, Issue: IssueWarning},
		{Host: "b.com", Port: 443, DaysRemaining: 90, Issue: IssueOK},
		{Host: "c.com", Port: 443, DaysRemaining: 1, Issue: IssueCritical},
		{Host: "d.com", Port: 443, DaysRemaining: 0, Issue: IssueError},
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DaysRemaining < results[j].DaysRemaining
	})

	if results[0].DaysRemaining != 0 {
		t.Errorf("first result should have 0 days (error), got %d", results[0].DaysRemaining)
	}
	if results[len(results)-1].DaysRemaining != 90 {
		t.Errorf("last result should have 90 days, got %d", results[len(results)-1].DaysRemaining)
	}
}

func TestOutput(t *testing.T) {
	results := []Result{
		{
			Host:          "google.com",
			Port:          443,
			Issue:         IssueOK,
			DaysRemaining: 90,
			NotAfter:      "2026-07-01",
			Issuer:        "GTS",
			CommonName:    "*.google.com",
			IsWildcard:    true,
			SANs:          5,
		},
		{
			Host:          "expired.badssl.com",
			Port:          443,
			Issue:         IssueExpired,
			DaysRemaining: -30,
			NotAfter:      "2015-04-12",
			Issuer:        "DigiCert",
			CommonName:    "*.badssl.com",
			IsWildcard:    true,
			SANs:          2,
		},
	}

	PrintTable(os.Stdout, results, 30, 7)

	var buf bytes.Buffer
	PrintJSON(&buf, results)
	if buf.Len() == 0 {
		t.Error("PrintJSON should write something")
	}
}

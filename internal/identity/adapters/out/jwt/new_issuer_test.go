package jwt

import (
	"strings"
	"testing"
	"time"
)

const testSecret = "test-secret-with-at-least-32-bytes"

func TestNewIssuerValidatesConfiguration(t *testing.T) {
	tests := []struct {
		name, secret, issuer string
		expiry               time.Duration
		wantErr              string
	}{
		{"short secret", strings.Repeat("s", 31), "test-suite", time.Hour, "JWT_SECRET"},
		{"missing issuer", testSecret, "", time.Hour, "JWT_ISSUER"},
		{"zero expiry", testSecret, "test-suite", 0, "expiry"},
		{"negative expiry", testSecret, "test-suite", -time.Minute, "expiry"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issuer, err := NewIssuer(tt.secret, tt.issuer, tt.expiry)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) || issuer != nil {
				t.Fatalf("NewIssuer() = (%v, %v), want error containing %q", issuer, err, tt.wantErr)
			}
		})
	}
}

func newTestIssuer(t *testing.T) *Issuer {
	t.Helper()
	issuer, err := NewIssuer(testSecret, "test-suite", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return issuer
}

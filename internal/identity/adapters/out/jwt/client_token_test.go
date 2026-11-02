package jwt

import (
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

func TestIssueClientScopesTokens(t *testing.T) {
	issuer := newTestIssuer(t)
	tests := []struct {
		name, customerID, vehicleID, wantScope string
	}{
		{"customer scope", "customer-1", "", "customer"},
		{"vehicle scope", "customer-1", "vehicle-9", "vehicle"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, expiresAt, err := issuer.IssueClient(tt.customerID, tt.vehicleID)
			if err != nil || !expiresAt.After(time.Now()) {
				t.Fatalf("IssueClient() = (%q, %v, %v)", token, expiresAt, err)
			}
			customerID, vehicleID, err := issuer.ValidateClient(token)
			if err != nil || customerID != tt.customerID || vehicleID != tt.vehicleID {
				t.Fatalf("ValidateClient() = (%q, %q, %v)", customerID, vehicleID, err)
			}
			claims := &ClientClaims{}
			if _, _, err := jwtlib.NewParser().ParseUnverified(token, claims); err != nil || claims.Scope != tt.wantScope {
				t.Fatalf("scope = %q (err %v), want %q", claims.Scope, err, tt.wantScope)
			}
		})
	}
}

func TestAdminAndClientTokensAreNotInterchangeable(t *testing.T) {
	issuer := newTestIssuer(t)
	adminToken, _, err := issuer.Issue("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := issuer.ValidateClient(adminToken); err == nil {
		t.Error("admin token accepted as client token")
	}
	clientToken, _, err := issuer.IssueClient("customer-1", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuer.Validate(clientToken); err == nil {
		t.Error("client token accepted as admin token")
	}
}

func TestValidateClientRejectsMalformedClaims(t *testing.T) {
	issuer := newTestIssuer(t)
	now := time.Now().UTC()
	registered := func(subject string) jwtlib.RegisteredClaims {
		return jwtlib.RegisteredClaims{Issuer: "test-suite", Subject: subject, Audience: jwtlib.ClaimStrings{"fiap-soat-client"}, IssuedAt: jwtlib.NewNumericDate(now), ExpiresAt: jwtlib.NewNumericDate(now.Add(time.Hour))}
	}
	expired := registered("customer-1")
	expired.ExpiresAt = jwtlib.NewNumericDate(now.Add(-time.Minute))
	tests := []struct {
		name   string
		method jwtlib.SigningMethod
		claims ClientClaims
	}{
		{"missing subject", jwtlib.SigningMethodHS256, ClientClaims{Scope: "customer", RegisteredClaims: registered("")}},
		{"unknown scope", jwtlib.SigningMethodHS256, ClientClaims{Scope: "admin", RegisteredClaims: registered("customer-1")}},
		{"vehicle scope without vehicle", jwtlib.SigningMethodHS256, ClientClaims{Scope: "vehicle", RegisteredClaims: registered("customer-1")}},
		{"expired", jwtlib.SigningMethodHS256, ClientClaims{Scope: "customer", RegisteredClaims: expired}},
		{"other hmac algorithm", jwtlib.SigningMethodHS512, ClientClaims{Scope: "customer", RegisteredClaims: registered("customer-1")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := jwtlib.NewWithClaims(tt.method, tt.claims).SignedString([]byte(testSecret))
			if err != nil {
				t.Fatal(err)
			}
			if customerID, vehicleID, err := issuer.ValidateClient(token); err == nil {
				t.Fatalf("ValidateClient() = (%q, %q), want error", customerID, vehicleID)
			}
		})
	}
}

func TestValidateRejectsTokenWithoutSubject(t *testing.T) {
	issuer := newTestIssuer(t)
	now := time.Now().UTC()
	claims := Claims{RegisteredClaims: jwtlib.RegisteredClaims{Issuer: "test-suite", Audience: jwtlib.ClaimStrings{"fiap-soat-admin"}, IssuedAt: jwtlib.NewNumericDate(now), ExpiresAt: jwtlib.NewNumericDate(now.Add(time.Hour))}}
	token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuer.Validate(token); err == nil {
		t.Fatal("token without subject was accepted")
	}
}

package jwt

import (
	"errors"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

type Claims struct{ jwtlib.RegisteredClaims }
type ClientClaims struct {
	Scope     string `json:"scope"`
	VehicleID string `json:"vehicle_id,omitempty"`
	jwtlib.RegisteredClaims
}
type Issuer struct {
	secret []byte
	issuer string
	expiry time.Duration
}

func NewIssuer(secret, issuer string, expiry time.Duration) (*Issuer, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must contain at least 32 bytes")
	}
	if issuer == "" {
		return nil, errors.New("JWT_ISSUER is required")
	}
	if expiry <= 0 {
		return nil, errors.New("JWT expiry must be positive")
	}
	return &Issuer{secret: []byte(secret), issuer: issuer, expiry: expiry}, nil
}

func (i *Issuer) Issue(userID string) (string, time.Time, error) {
	now := time.Now().UTC()
	expires := now.Add(i.expiry)
	claims := Claims{RegisteredClaims: jwtlib.RegisteredClaims{Issuer: i.issuer, Subject: userID, Audience: jwtlib.ClaimStrings{"fiap-soat-admin"}, IssuedAt: jwtlib.NewNumericDate(now), ExpiresAt: jwtlib.NewNumericDate(expires)}}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	encoded, err := token.SignedString(i.secret)
	return encoded, expires, err
}

func (i *Issuer) Validate(encoded string) (string, error) {
	claims := &Claims{}
	token, err := jwtlib.ParseWithClaims(encoded, claims, func(token *jwtlib.Token) (any, error) {
		if token.Method != jwtlib.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return i.secret, nil
	}, jwtlib.WithValidMethods([]string{jwtlib.SigningMethodHS256.Alg()}), jwtlib.WithIssuer(i.issuer), jwtlib.WithAudience("fiap-soat-admin"), jwtlib.WithExpirationRequired(), jwtlib.WithIssuedAt())
	if err != nil || token == nil || !token.Valid || claims.Subject == "" {
		return "", errors.New("invalid token")
	}
	return claims.Subject, nil
}

func (i *Issuer) IssueClient(customerID, vehicleID string) (string, time.Time, error) {
	now := time.Now().UTC()
	expires := now.Add(i.expiry)
	scope := "customer"
	if vehicleID != "" {
		scope = "vehicle"
	}
	claims := ClientClaims{Scope: scope, VehicleID: vehicleID, RegisteredClaims: jwtlib.RegisteredClaims{Issuer: i.issuer, Subject: customerID, Audience: jwtlib.ClaimStrings{"fiap-soat-client"}, IssuedAt: jwtlib.NewNumericDate(now), ExpiresAt: jwtlib.NewNumericDate(expires)}}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	encoded, err := token.SignedString(i.secret)
	return encoded, expires, err
}

func (i *Issuer) ValidateClient(encoded string) (customerID, vehicleID string, err error) {
	claims := &ClientClaims{}
	token, parseErr := jwtlib.ParseWithClaims(encoded, claims, func(token *jwtlib.Token) (any, error) {
		if token.Method != jwtlib.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return i.secret, nil
	}, jwtlib.WithValidMethods([]string{jwtlib.SigningMethodHS256.Alg()}), jwtlib.WithIssuer(i.issuer), jwtlib.WithAudience("fiap-soat-client"), jwtlib.WithExpirationRequired(), jwtlib.WithIssuedAt())
	if parseErr != nil || token == nil || !token.Valid || claims.Subject == "" || (claims.Scope != "customer" && claims.Scope != "vehicle") || (claims.Scope == "vehicle" && claims.VehicleID == "") {
		return "", "", errors.New("invalid client token")
	}
	return claims.Subject, claims.VehicleID, nil
}

package customerhttp

import (
	"net/http"
	"testing"
)

const (
	loginPath = "/api/v1/customers/client/auth/login"
	platePath = "/api/v1/customers/client/auth/plate"
)

func TestClientLogin(t *testing.T) {
	t.Run("test_it_should_issue_customer_scoped_token_when_credentials_are_valid", func(t *testing.T) {
		env := newClientEnv(t)
		response := env.do(http.MethodPost, loginPath, `{"email":"ANA@example.com","password":"`+clientPassword+`"}`, "")
		assertStatus(t, response, http.StatusOK)
		session := decodeJSON[clientSession](t, response)
		customerID, vehicleID, err := env.issuer.ValidateClient(session.AccessToken)
		if err != nil || customerID != "c-1" || vehicleID != "" {
			t.Errorf("token claims = (%q, %q, %v), want c-1 without vehicle", customerID, vehicleID, err)
		}
		if session.ExpiresAt.IsZero() {
			t.Error("expires_at is zero")
		}
		if len(env.limiter.keys) != 1 || env.limiter.keys[0] != "client:auth:192.0.2.1" {
			t.Errorf("rate limit keys = %v, want client:auth:<ip>", env.limiter.keys)
		}
	})

	cases := []struct {
		name, body string
		status     int
		title      string
	}{
		{"test_it_should_return_unauthorized_when_password_is_wrong", `{"email":"ana@example.com","password":"wrong-password-1"}`, http.StatusUnauthorized, "Invalid credentials"},
		{"test_it_should_return_unauthorized_when_customer_is_inactive", `{"email":"co@example.com","password":"` + clientPassword + `"}`, http.StatusUnauthorized, "Invalid credentials"},
		{"test_it_should_return_bad_request_when_body_has_unknown_field", `{"email":"a","password":"b","extra":1}`, http.StatusBadRequest, "Invalid request"},
		{"test_it_should_return_bad_request_when_body_has_trailing_data", `{"email":"a","password":"b"} {}`, http.StatusBadRequest, "Invalid request"},
		{"test_it_should_return_bad_request_when_body_is_not_json", `not-json`, http.StatusBadRequest, "Invalid request"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertProblem(t, newClientEnv(t).do(http.MethodPost, loginPath, tc.body, ""), tc.status, tc.title)
		})
	}
}

func TestPlateLogin(t *testing.T) {
	t.Run("test_it_should_issue_vehicle_scoped_token_when_plate_and_prefix_match", func(t *testing.T) {
		env := newClientEnv(t)
		env.customers.owner["ABC1D23"], env.customers.plates["ABC1D23"] = "c-1", "v-1"
		response := env.do(http.MethodPost, platePath, `{"plate":"abc-1d23","document_prefix":"529.98"}`, "")
		assertStatus(t, response, http.StatusOK)
		session := decodeJSON[clientSession](t, response)
		customerID, vehicleID, err := env.issuer.ValidateClient(session.AccessToken)
		if err != nil || customerID != "c-1" || vehicleID != "v-1" {
			t.Errorf("token claims = (%q, %q, %v), want (c-1, v-1)", customerID, vehicleID, err)
		}
	})

	t.Run("test_it_should_return_unauthorized_when_plate_is_unknown", func(t *testing.T) {
		response := newClientEnv(t).do(http.MethodPost, platePath, `{"plate":"ZZZ9Z99","document_prefix":"52998"}`, "")
		assertProblem(t, response, http.StatusUnauthorized, "Invalid credentials")
	})

	t.Run("test_it_should_return_bad_request_when_body_is_invalid", func(t *testing.T) {
		response := newClientEnv(t).do(http.MethodPost, platePath, `{"plate":1}`, "")
		assertProblem(t, response, http.StatusBadRequest, "Invalid request")
	})
}

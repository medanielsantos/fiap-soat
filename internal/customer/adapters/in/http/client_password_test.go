package customerhttp

import (
	"net/http"
	"testing"
)

const (
	forgotPath = "/api/v1/customers/client/auth/password/forgot"
	resetPath  = "/api/v1/customers/client/auth/password/reset"
)

func TestClientForgotPassword(t *testing.T) {
	t.Run("test_it_should_accept_and_send_reset_mail_for_active_customer", func(t *testing.T) {
		env := newClientEnv(t)
		assertStatus(t, env.do(http.MethodPost, forgotPath, `{"email":"ana@example.com"}`, ""), http.StatusAccepted)
		if env.customers.resets != 1 || env.mailer.sent != 1 {
			t.Errorf("resets = %d, mails = %d, want 1 and 1", env.customers.resets, env.mailer.sent)
		}
	})

	t.Run("test_it_should_accept_without_revealing_unknown_email", func(t *testing.T) {
		env := newClientEnv(t)
		assertStatus(t, env.do(http.MethodPost, forgotPath, `{"email":"missing@example.com"}`, ""), http.StatusAccepted)
		if env.mailer.sent != 0 {
			t.Errorf("mails = %d, want 0", env.mailer.sent)
		}
	})

	t.Run("test_it_should_still_accept_when_reset_delivery_fails", func(t *testing.T) {
		env := newClientEnv(t)
		env.mailer.err = errBackend
		assertStatus(t, env.do(http.MethodPost, forgotPath, `{"email":"ana@example.com"}`, ""), http.StatusAccepted)
		if env.mailer.sent != 1 {
			t.Errorf("mails attempted = %d, want 1", env.mailer.sent)
		}
	})

	t.Run("test_it_should_return_bad_request_when_body_is_invalid", func(t *testing.T) {
		assertProblem(t, newClientEnv(t).do(http.MethodPost, forgotPath, `{"mail":"x"}`, ""), http.StatusBadRequest, "Invalid request")
	})
}

func TestClientResetPassword(t *testing.T) {
	t.Run("test_it_should_return_no_content_when_reset_succeeds", func(t *testing.T) {
		response := newClientEnv(t).do(http.MethodPost, resetPath, `{"token":"t","password":"`+clientPassword+`"}`, "")
		assertStatus(t, response, http.StatusNoContent)
	})

	t.Run("test_it_should_return_bad_request_when_password_is_too_short", func(t *testing.T) {
		response := newClientEnv(t).do(http.MethodPost, resetPath, `{"token":"t","password":"short"}`, "")
		assertProblem(t, response, http.StatusBadRequest, "Invalid reset request")
	})

	t.Run("test_it_should_return_bad_request_when_token_is_rejected", func(t *testing.T) {
		env := newClientEnv(t)
		env.customers.resetErr = errBackend
		response := env.do(http.MethodPost, resetPath, `{"token":"t","password":"`+clientPassword+`"}`, "")
		assertProblem(t, response, http.StatusBadRequest, "Invalid reset request")
	})

	t.Run("test_it_should_return_bad_request_when_body_is_invalid", func(t *testing.T) {
		assertProblem(t, newClientEnv(t).do(http.MethodPost, resetPath, `[]`, ""), http.StatusBadRequest, "Invalid request")
	})
}

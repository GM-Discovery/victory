package identity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Kernel 76 (K76-H01): open email/password registration is closed. These tests
// pin the closure itself rather than the handler's internals, so reopening it
// by accident is a test failure rather than a silent change in admission
// policy.
func TestPasswordSignupClosedByDefault(t *testing.T) {
	t.Setenv("PASSWORD_SIGNUP_ENABLED", "")

	if PasswordSignupEnabled() {
		t.Fatal("password signup must be closed when PASSWORD_SIGNUP_ENABLED is unset")
	}

	body := strings.NewReader(`{"email":"stranger@example.com","handle":"stranger","password":"hunter2hunter2","display_name":"Stranger"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", body)
	rec := httptest.NewRecorder()

	// A nil pool is safe here precisely because the closure must be decided
	// before the handler touches the database.
	HandleSignup(nil, true).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("closed signup should be 403, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Ok    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Ok || payload.Error != "password_signup_closed" {
		t.Fatalf("unexpected closed-signup payload: %+v", payload)
	}
}

func TestPasswordSignupReopensForLocalDevelopment(t *testing.T) {
	for _, value := range []string{"1", "true", "TRUE", "yes", "on"} {
		t.Setenv("PASSWORD_SIGNUP_ENABLED", value)
		if !PasswordSignupEnabled() {
			t.Fatalf("PASSWORD_SIGNUP_ENABLED=%q should reopen signup", value)
		}
	}

	for _, value := range []string{"", "0", "false", "no", "off", "maybe"} {
		t.Setenv("PASSWORD_SIGNUP_ENABLED", value)
		if PasswordSignupEnabled() {
			t.Fatalf("PASSWORD_SIGNUP_ENABLED=%q must not reopen signup", value)
		}
	}
}

// Kernel 76 (K76-C01): the self-service reset flow is closed because its only
// delivery mechanism was printing the raw token into the backend log. The
// response must say so plainly rather than pretending a mail was sent.
func TestSelfServicePasswordResetIsClosed(t *testing.T) {
	body := strings.NewReader(`{"email":"grant@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/request", body)
	rec := httptest.NewRecorder()

	HandleForgotPassword(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("closed reset request should be 410, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Ok    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Ok || payload.Error != "self_service_password_reset_unavailable" {
		t.Fatalf("unexpected closed-reset payload: %+v", payload)
	}
}

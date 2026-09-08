package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Kernel 76 (K76-H01): open email/password registration is closed. These tests
// pin the closure itself rather than the handler's internals, so reopening it
// by accident is a test failure rather than a silent change in admission
// policy.
//
// Kernel 100 carves one exception into that closure: a brand new install has
// no accounts at all yet, and Discord can't be required to create the very
// first one without contradicting K100's "Discord stays optional at first
// boot" rule (see bootstrapWindowOpen in auth.go). So "closed by default" can
// no longer be proven with a nil pool -- deciding it now requires knowing
// whether a real account already exists. This test proves the ordinary case
// (an install that already has an account stays closed); the bootstrap
// exception itself is proven separately in TestPasswordSignupOpensOnlyForFirstAccount.
func TestPasswordSignupClosedByDefault(t *testing.T) {
	t.Setenv("PASSWORD_SIGNUP_ENABLED", "")

	if PasswordSignupEnabled() {
		t.Fatal("password signup must be closed when PASSWORD_SIGNUP_ENABLED is unset")
	}

	pool := openDiscordTestPool(t)
	ensureBootstrapSchema(t, pool)
	handle := "closure_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
	existingUserID := insertBootstrapUser(t, pool, handle, "Existing Account")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, existingUserID)
	})

	body := strings.NewReader(`{"email":"stranger@example.com","handle":"stranger","password":"hunter2hunter2","display_name":"Stranger"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", body)
	rec := httptest.NewRecorder()

	HandleSignup(pool, true, DiscordOAuthConfig{}).ServeHTTP(rec, req)

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

// TestPasswordSignupOpensOnlyForFirstAccount proves the Kernel 100 bootstrap
// exception is a one-time window, not a standing reopening: signup succeeds
// while the database has no real accounts yet, then the very next attempt
// (now that one exists) is rejected exactly like the always-closed case.
func TestPasswordSignupOpensOnlyForFirstAccount(t *testing.T) {
	t.Setenv("PASSWORD_SIGNUP_ENABLED", "")

	pool := openDiscordTestPool(t)
	ensureBootstrapSchema(t, pool)

	open, err := bootstrapWindowOpen(context.Background(), pool)
	if err != nil {
		t.Fatalf("bootstrapWindowOpen: %v", err)
	}
	if !open {
		t.Skip("test database already has real accounts; bootstrap window is not observable here")
	}

	suffix := strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")

	firstBody := strings.NewReader(`{"email":"first_` + suffix + `@example.com","handle":"first_` + suffix + `","password":"hunter2hunter2","display_name":"First Account"}`)
	firstReq := httptest.NewRequest(http.MethodPost, "/api/auth/signup", firstBody)
	firstRec := httptest.NewRecorder()
	HandleSignup(pool, true, DiscordOAuthConfig{}).ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first-ever signup should succeed while the bootstrap window is open, got %d body=%s", firstRec.Code, firstRec.Body.String())
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE handle = $1`, "first_"+suffix)
	})

	secondBody := strings.NewReader(`{"email":"second_` + suffix + `@example.com","handle":"second_` + suffix + `","password":"hunter2hunter2","display_name":"Second Account"}`)
	secondReq := httptest.NewRequest(http.MethodPost, "/api/auth/signup", secondBody)
	secondRec := httptest.NewRecorder()
	HandleSignup(pool, true, DiscordOAuthConfig{}).ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusForbidden {
		t.Fatalf("signup after the first account exists should be closed again, got %d body=%s", secondRec.Code, secondRec.Body.String())
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
// TestSelfServicePasswordResetIsClosed proves the Kernel 76 closure remains
// the default: an unconfigured ForgotPasswordConfig (Ready: false, the zero
// value) keeps this 410, exactly as it was before Kernel 77 added a real
// delivery path. Kernel 77's open-path behavior (Ready: true, with mail
// actually configured) is covered separately in account_recovery_test.go.
func TestSelfServicePasswordResetIsClosed(t *testing.T) {
	body := strings.NewReader(`{"email":"grant@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/request", body)
	rec := httptest.NewRecorder()

	HandleForgotPassword(nil, ForgotPasswordConfig{}).ServeHTTP(rec, req)

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

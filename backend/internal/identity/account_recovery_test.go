package identity

import (
	"context"
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"victory/backend/internal/ratelimit"
	"victory/backend/internal/sessions"
)

// fakeMailer records what would have been sent instead of touching a real
// SMTP relay, so these tests exercise the full request/store/redeem path
// without any network dependency.
type fakeMailer struct {
	mu          sync.Mutex
	resetLinks  []string
	verifyLinks []string
}

func (m *fakeMailer) SendPasswordReset(ctx context.Context, recipient, resetURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resetLinks = append(m.resetLinks, resetURL)
	return nil
}

func (m *fakeMailer) SendEmailVerification(ctx context.Context, recipient, verifyURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.verifyLinks = append(m.verifyLinks, verifyURL)
	return nil
}

func (m *fakeMailer) lastResetToken(t *testing.T) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		n := len(m.resetLinks)
		var link string
		if n > 0 {
			link = m.resetLinks[n-1]
		}
		m.mu.Unlock()
		if link != "" {
			parts := strings.SplitN(link, "token=", 2)
			if len(parts) == 2 {
				return parts[1]
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("no reset email was sent in time")
	return ""
}

func (m *fakeMailer) lastVerifyToken(t *testing.T) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		n := len(m.verifyLinks)
		var link string
		if n > 0 {
			link = m.verifyLinks[n-1]
		}
		m.mu.Unlock()
		if link != "" {
			parts := strings.SplitN(link, "token=", 2)
			if len(parts) == 2 {
				return parts[1]
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("no verification email was sent in time")
	return ""
}

func readyRecoveryConfig(mailer RecoveryMailer) ForgotPasswordConfig {
	return ForgotPasswordConfig{Ready: true, Mailer: mailer, BaseURL: "https://victory.example"}
}

func TestPasswordResetRequestIsGenericForKnownAndUnknownEmail(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	mailer := &fakeMailer{}

	handle := "recover_known_" + time.Now().UTC().Format("150405.000000")
	userID := insertAccountTestUser(t, pool, handle, "Recover Known")
	email := handle + "@example.com"
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID) })
	if _, err := pool.Exec(ctx, `UPDATE users SET email = $1, email_verified_at = NOW() WHERE id = $2`, email, userID); err != nil {
		t.Fatalf("set verified email: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO auth.password_credentials (user_id, password_hash) VALUES ($1, $2)`, userID, "$argon2id$v=19$m=1,t=1,p=1$c2FsdA$aGFzaA"); err != nil {
		t.Fatalf("insert password credential: %v", err)
	}

	cfg := readyRecoveryConfig(mailer)

	knownReq := httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/request", strings.NewReader(`{"email":"`+email+`"}`))
	knownRec := httptest.NewRecorder()
	HandleForgotPassword(pool, cfg).ServeHTTP(knownRec, knownReq)

	unknownReq := httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/request", strings.NewReader(`{"email":"nobody-`+handle+`@example.com"}`))
	unknownRec := httptest.NewRecorder()
	HandleForgotPassword(pool, cfg).ServeHTTP(unknownRec, unknownReq)

	if knownRec.Code != http.StatusOK || unknownRec.Code != http.StatusOK {
		t.Fatalf("expected both to return 200, got known=%d unknown=%d", knownRec.Code, unknownRec.Code)
	}
	if knownRec.Body.String() != unknownRec.Body.String() {
		t.Fatalf("responses differ: known=%s unknown=%s", knownRec.Body.String(), unknownRec.Body.String())
	}

	rawToken := mailer.lastResetToken(t)

	mailer.mu.Lock()
	sentCount := len(mailer.resetLinks)
	mailer.mu.Unlock()
	if sentCount != 1 {
		t.Fatalf("expected exactly one email sent (only for the known+verified address), got %d", sentCount)
	}

	var storedCount int
	tokenHash := sha256.Sum256([]byte(rawToken))
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM auth.password_reset_tokens WHERE token_hash = $1`, tokenHash[:]).Scan(&storedCount); err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if storedCount != 1 {
		t.Fatalf("expected the raw token to correspond to exactly one hashed row, got %d", storedCount)
	}
}

func TestPasswordResetDoesNotSendForUnverifiedEmail(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	mailer := &fakeMailer{}

	handle := "recover_unverified_" + time.Now().UTC().Format("150405.000000")
	userID := insertAccountTestUser(t, pool, handle, "Recover Unverified")
	email := handle + "@example.com"
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID) })
	if _, err := pool.Exec(ctx, `UPDATE users SET email = $1 WHERE id = $2`, email, userID); err != nil {
		t.Fatalf("set unverified email: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/request", strings.NewReader(`{"email":"`+email+`"}`))
	rec := httptest.NewRecorder()
	HandleForgotPassword(pool, readyRecoveryConfig(mailer)).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected generic 200, got %d", rec.Code)
	}

	time.Sleep(150 * time.Millisecond)
	mailer.mu.Lock()
	sent := len(mailer.resetLinks)
	mailer.mu.Unlock()
	if sent != 0 {
		t.Fatalf("expected no email for an unverified address, got %d sent", sent)
	}
}

func TestPasswordResetConfirmEndToEndAndReplayFails(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	mailer := &fakeMailer{}

	handle := "recover_e2e_" + time.Now().UTC().Format("150405.000000")
	userID := insertAccountTestUser(t, pool, handle, "Recover E2E")
	email := handle + "@example.com"
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID) })
	if _, err := pool.Exec(ctx, `UPDATE users SET email = $1, email_verified_at = NOW() WHERE id = $2`, email, userID); err != nil {
		t.Fatalf("set verified email: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO auth.password_credentials (user_id, password_hash) VALUES ($1, $2)`, userID, "$argon2id$v=19$m=1,t=1,p=1$c2FsdA$aGFzaA"); err != nil {
		t.Fatalf("insert password credential: %v", err)
	}

	// An active session that must be revoked by a successful reset.
	oldRaw, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create pre-reset session: %v", err)
	}

	reqReq := httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/request", strings.NewReader(`{"email":"`+email+`"}`))
	reqRec := httptest.NewRecorder()
	HandleForgotPassword(pool, readyRecoveryConfig(mailer)).ServeHTTP(reqRec, reqReq)
	rawToken := mailer.lastResetToken(t)

	confirmBody := `{"token":"` + rawToken + `","new_password":"a-brand-new-strong-password-1"}`
	confirmReq := httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/confirm", strings.NewReader(confirmBody))
	confirmRec := httptest.NewRecorder()
	HandleResetPassword(pool, false).ServeHTTP(confirmRec, confirmReq)
	if confirmRec.Code != http.StatusOK {
		t.Fatalf("expected reset confirm to succeed, got %d: %s", confirmRec.Code, confirmRec.Body.String())
	}

	// Replay must fail.
	replayReq := httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/confirm", strings.NewReader(confirmBody))
	replayRec := httptest.NewRecorder()
	HandleResetPassword(pool, false).ServeHTTP(replayRec, replayReq)
	if replayRec.Code != http.StatusBadRequest || !strings.Contains(replayRec.Body.String(), "invalid_or_expired_token") {
		t.Fatalf("expected replay to be rejected, got %d: %s", replayRec.Code, replayRec.Body.String())
	}

	// The pre-reset session must be revoked.
	meReq := httptest.NewRequest(http.MethodGet, "/api/account/me", nil)
	meReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: oldRaw})
	meRec := httptest.NewRecorder()
	HandleAccountMe(pool).ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected the pre-reset session to be revoked, got %d", meRec.Code)
	}
}

func TestPasswordResetRequestClosedWithoutConfig(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/request", strings.NewReader(`{"email":"whoever@example.com"}`))
	rec := httptest.NewRecorder()
	HandleForgotPassword(nil, ForgotPasswordConfig{Ready: false}).ServeHTTP(rec, req)
	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410 when not configured, got %d", rec.Code)
	}
}

func TestEmailVerificationRequestAndConfirm(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	mailer := &fakeMailer{}

	handle := "verify_e2e_" + time.Now().UTC().Format("150405.000000")
	userID := insertAccountTestUser(t, pool, handle, "Verify E2E")
	email := handle + "@example.com"
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID) })
	if _, err := pool.Exec(ctx, `UPDATE users SET email = $1 WHERE id = $2`, email, userID); err != nil {
		t.Fatalf("set email: %v", err)
	}

	raw, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	reqReq := httptest.NewRequest(http.MethodPost, "/api/account/email/verify", nil)
	reqReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	reqRec := httptest.NewRecorder()
	HandleRequestEmailVerification(pool, readyRecoveryConfig(mailer)).ServeHTTP(reqRec, reqReq)
	if reqRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", reqRec.Code, reqRec.Body.String())
	}

	token := mailer.lastVerifyToken(t)

	confirmReq := httptest.NewRequest(http.MethodPost, "/api/account/email/verify/confirm", strings.NewReader(`{"token":"`+token+`"}`))
	confirmRec := httptest.NewRecorder()
	HandleConfirmEmailVerification(pool).ServeHTTP(confirmRec, confirmReq)
	if confirmRec.Code != http.StatusOK {
		t.Fatalf("expected verification confirm to succeed, got %d: %s", confirmRec.Code, confirmRec.Body.String())
	}

	var verifiedAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT email_verified_at FROM users WHERE id = $1`, userID).Scan(&verifiedAt); err != nil {
		t.Fatalf("check verified state: %v", err)
	}
	if verifiedAt == nil {
		t.Fatal("expected email_verified_at to be set")
	}

	// Now this address is verified, so a real reset request should send.
	pwReq := httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/request", strings.NewReader(`{"email":"`+email+`"}`))
	pwRec := httptest.NewRecorder()
	HandleForgotPassword(pool, readyRecoveryConfig(mailer)).ServeHTTP(pwRec, pwReq)
	if pwRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", pwRec.Code)
	}
	mailer.lastResetToken(t) // fails the test if nothing was sent

	// Replay of the verification token must fail.
	replayReq := httptest.NewRequest(http.MethodPost, "/api/account/email/verify/confirm", strings.NewReader(`{"token":"`+token+`"}`))
	replayRec := httptest.NewRecorder()
	HandleConfirmEmailVerification(pool).ServeHTTP(replayRec, replayReq)
	if replayRec.Code != http.StatusBadRequest {
		t.Fatalf("expected replayed verification token to be rejected, got %d", replayRec.Code)
	}
}

func TestPasswordResetRequestRateLimited(t *testing.T) {
	pool := openDiscordTestPool(t)
	mailer := &fakeMailer{}
	cfg := readyRecoveryConfig(mailer)

	limiter := ratelimit.New(10, 10)
	handler := ratelimit.Middleware(limiter, HandleForgotPassword(pool, cfg))

	var lastCode int
	for i := 0; i < 15; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/request", strings.NewReader(`{"email":"rl-test@example.com"}`))
		req.RemoteAddr = "203.0.113.9:12345"
		rec := httptest.NewRecorder()
		handler(rec, req)
		lastCode = rec.Code
	}
	if lastCode != http.StatusTooManyRequests {
		t.Fatalf("expected the credential limiter to eventually return 429, last code was %d", lastCode)
	}
}

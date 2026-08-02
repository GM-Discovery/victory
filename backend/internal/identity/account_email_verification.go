package identity

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// emailVerificationExpiry is deliberately longer than the 1-hour password
// reset token: verifying an address is a lower-stakes, unhurried action, and
// people often don't open a "verify your email" link right away.
const emailVerificationExpiry = 24 * time.Hour

// HandleRequestEmailVerification sends a verification link to the
// authenticated user's current email (Kernel 77 §8.5). The account is
// always resolved from the session.
func HandleRequestEmailVerification(pool *pgxpool.Pool, cfg ForgotPasswordConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		var email string
		var alreadyVerifiedAt *time.Time
		if err := pool.QueryRow(ctx, `SELECT COALESCE(email, ''), email_verified_at FROM users WHERE id = $1`, userID).Scan(&email, &alreadyVerifiedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "lookup_failed"})
			return
		}
		email = strings.ToLower(strings.TrimSpace(email))
		if email == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "no_email_on_file"})
			return
		}
		if alreadyVerifiedAt != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "already verified"})
			return
		}

		if !cfg.Ready {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "email_delivery_not_configured"})
			return
		}

		if _, err := pool.Exec(ctx, `
			UPDATE auth.email_verification_tokens
			SET consumed_at = NOW()
			WHERE user_id = $1 AND consumed_at IS NULL AND expires_at > NOW()
		`, userID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "invalidate_old_tokens_failed"})
			return
		}

		rawToken, tokenHash, err := newResetToken()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "token_generate_failed"})
			return
		}
		expiresAt := time.Now().UTC().Add(emailVerificationExpiry)
		if _, err := pool.Exec(ctx, `
			INSERT INTO auth.email_verification_tokens (user_id, email, token_hash, expires_at, request_ip)
			VALUES ($1, $2, $3, $4, $5)
		`, userID, email, tokenHash, expiresAt, clientIP(r)); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "token_store_failed"})
			return
		}

		verifyURL := strings.TrimRight(cfg.BaseURL, "/") + "/account/verify-email.html?token=" + rawToken

		go func(recipient, url string) {
			sendCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := cfg.Mailer.SendEmailVerification(sendCtx, recipient, url); err != nil {
				log.Printf("email verification delivery failed: %v", err)
			}
		}(email, verifyURL)

		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "verification email sent"})
	}
}

type confirmEmailVerificationRequest struct {
	Token string `json:"token"`
}

// HandleConfirmEmailVerification redeems a verification token. It re-checks
// that the token's email still matches users.email, so a token issued for a
// since-changed address cannot verify the new one.
func HandleConfirmEmailVerification(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		var req confirmEmailVerificationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}
		req.Token = strings.TrimSpace(req.Token)
		if req.Token == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "token_required"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		tokenHash := sha256.Sum256([]byte(req.Token))

		var tokenID, userID, tokenEmail string
		err := pool.QueryRow(ctx, `
			SELECT id::text, user_id::text, email FROM auth.email_verification_tokens
			WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > NOW()
			LIMIT 1
		`, tokenHash[:]).Scan(&tokenID, &userID, &tokenEmail)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_or_expired_token"})
			return
		}

		var currentEmail string
		if err := pool.QueryRow(ctx, `SELECT COALESCE(email, '') FROM users WHERE id = $1`, userID).Scan(&currentEmail); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "lookup_failed"})
			return
		}
		if !strings.EqualFold(strings.TrimSpace(currentEmail), tokenEmail) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "email_changed_since_request"})
			return
		}

		if _, err := pool.Exec(ctx, `UPDATE users SET email_verified_at = NOW() WHERE id = $1`, userID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "verify_failed"})
			return
		}
		if _, err := pool.Exec(ctx, `UPDATE auth.email_verification_tokens SET consumed_at = NOW() WHERE id = $1`, tokenID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "token_consume_failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

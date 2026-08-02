package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"victory/backend/internal/access"
	"victory/backend/internal/characters"
	"victory/backend/internal/sessions"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SignupRequest struct {
	Email       string `json:"email"`
	Handle      string `json:"handle"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type LoginRequest struct {
	Handle   string `json:"handle"`
	Password string `json:"password"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// PasswordSignupEnabled reports whether open email/password registration is
// admitted. Kernel 76 (K76-H01) closes it by default: the route was reachable
// from the public internet, needed no invite, and granted every caller an
// active `audience` membership in the amurray-family Location, which was
// enough to read the Third Place roster of real people. Account establishment
// now goes through Discord.
//
// PASSWORD_SIGNUP_ENABLED=true reopens it for local development. Nothing in
// the deployed configuration sets it.
func PasswordSignupEnabled() bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv("PASSWORD_SIGNUP_ENABLED"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func HandleSignup(pool *pgxpool.Pool, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		if !PasswordSignupEnabled() {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":     false,
				"error":  "password_signup_closed",
				"detail": "Victory accounts are created by signing in with Discord.",
			})
			return
		}

		var req SignupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}

		req.Email = strings.TrimSpace(strings.ToLower(req.Email))
		req.Handle = normalizeHandle(req.Handle)
		req.DisplayName = strings.TrimSpace(req.DisplayName)
		if req.DisplayName == "" {
			req.DisplayName = req.Handle
		}

		if req.Email == "" || req.Handle == "" || req.Password == "" || req.DisplayName == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "email_handle_password_display_name_required"})
			return
		}

		passwordHash, err := HashPassword(req.Password)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		tx, err := pool.Begin(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "tx_begin_failed"})
			return
		}
		defer tx.Rollback(ctx)

		var userID string
		err = tx.QueryRow(ctx, `
			INSERT INTO users (email, handle, display_name)
			VALUES ($1, $2, $3)
			RETURNING id
		`, req.Email, req.Handle, req.DisplayName).Scan(&userID)
		if err != nil {
			log.Printf("signup insert user failed: %v", err)
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "signup_failed"})
			return
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO auth.password_credentials (user_id, password_hash)
			VALUES ($1, $2)
		`, userID, passwordHash)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "password_store_failed"})
			return
		}

		var locationID string
		err = tx.QueryRow(ctx, `
			SELECT id
			FROM locations
			WHERE slug = 'amurray-family'
			LIMIT 1
		`).Scan(&locationID)
		if err == nil {
			_, _ = tx.Exec(ctx, `
				INSERT INTO location_memberships (location_id, user_id, role, granted_by_user_id, active)
				VALUES ($1, $2, 'audience', $2, TRUE)
				ON CONFLICT (location_id, user_id, role) DO NOTHING
			`, locationID, userID)
		}

		if err := tx.Commit(ctx); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "tx_commit_failed"})
			return
		}

		sessionCtx, sessionCancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer sessionCancel()

		rawToken, expiresAt, err := sessions.CreateSession(sessionCtx, pool, userID, 24*time.Hour, r)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "session_create_failed"})
			return
		}

		sessions.SetSessionCookie(w, rawToken, expiresAt, secureCookie)

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": map[string]any{
				"user_id":      userID,
				"handle":       req.Handle,
				"display_name": req.DisplayName,
				"email":        req.Email,
			},
		})
	}
}

func HandleLogin(pool *pgxpool.Pool, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}

		handle := normalizeHandle(req.Handle)

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var userID, passwordHash, displayName, email string
		err := pool.QueryRow(ctx, `
			SELECT u.id, pc.password_hash, u.display_name, COALESCE(u.email, '')
			FROM users u
			JOIN auth.password_credentials pc ON pc.user_id = u.id
			WHERE u.handle = $1
			LIMIT 1
		`, handle).Scan(&userID, &passwordHash, &displayName, &email)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "invalid_credentials"})
			return
		}

		ok, err := VerifyPassword(req.Password, passwordHash)
		if err != nil || !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "invalid_credentials"})
			return
		}

		rawToken, expiresAt, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, r)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "session_create_failed"})
			return
		}

		sessions.SetSessionCookie(w, rawToken, expiresAt, secureCookie)

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": map[string]any{
				"user_id":      userID,
				"handle":       handle,
				"display_name": displayName,
				"email":        email,
			},
		})
	}
}

func HandleLogout(pool *pgxpool.Pool, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		raw, err := sessions.ReadSessionCookie(r)
		if err == nil && raw != "" {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			_ = sessions.RevokeSessionByRawToken(ctx, pool, raw)
		}

		sessions.ClearSessionCookie(w, secureCookie)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

// RecoveryMailer is the minimal interface HandleForgotPassword needs. It
// matches mailer.RecoveryMailer without importing that package directly,
// keeping identity free of a hard SMTP dependency.
type RecoveryMailer interface {
	SendPasswordReset(ctx context.Context, recipient, resetURL string) error
	SendEmailVerification(ctx context.Context, recipient, verifyURL string) error
}

// ForgotPasswordConfig controls whether self-service reset is open. Ready
// must only be true when a real delivery channel is configured -- see
// cmd/victory/main.go, which computes it from RECOVERY_EMAIL_ENABLED plus
// the presence of SMTP configuration (Kernel 77 §8.2: "fail closed... when
// recovery is advertised but mail is not configured" -- if the operator
// flips the flag on without finishing SMTP setup, this stays closed rather
// than accepting requests that would silently never deliver).
type ForgotPasswordConfig struct {
	Ready   bool
	Mailer  RecoveryMailer
	BaseURL string
}

const forgotPasswordGenericMessage = "If that address can receive a Victory recovery message, one has been sent."

// HandleForgotPassword was closed by Kernel 76 (K76-C01): Victory had no
// email delivery, so the only way this flow ever returned a token was by
// printing the raw value into the backend log, an account-takeover
// credential for any address the caller could name. Kernel 77 reopens it
// once a real delivery channel exists (cfg.Ready), behind:
//   - a uniform response whether or not the address exists, is verified, or
//     delivery succeeds (§8.3) -- the caller learns nothing;
//   - only proceeding for an address that has completed email verification
//     (§8.5) -- an unverified users.email never receives a reset link;
//   - a fresh, single-use, hashed, 1-hour token, mirroring victory-recover's
//     own break-glass token shape exactly;
//   - older outstanding tokens for the account invalidated first;
//   - delivery happening off the request goroutine so a slow or failing
//     SMTP relay cannot be used to time-probe account existence, and so a
//     delivery failure never surfaces to the caller.
//
// If cfg.Ready is false, the endpoint stays closed exactly as Kernel 76 left
// it: break-glass recovery via cmd/victory-recover remains the only path.
func HandleForgotPassword(pool *pgxpool.Pool, cfg ForgotPasswordConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		if !cfg.Ready {
			writeJSON(w, http.StatusGone, map[string]any{
				"ok":    false,
				"error": "self_service_password_reset_unavailable",
				"detail": "Password reset is handled by the Victory operator. " +
					"Sign in with Discord, or contact the operator for a recovery link.",
			})
			return
		}

		generic := map[string]any{"ok": true, "message": forgotPasswordGenericMessage}

		var req struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusOK, generic)
			return
		}
		email := strings.ToLower(strings.TrimSpace(req.Email))
		if email == "" {
			writeJSON(w, http.StatusOK, generic)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var userID string
		var emailVerifiedAt *time.Time
		err := pool.QueryRow(ctx, `
			SELECT id, email_verified_at FROM users WHERE lower(email) = $1 LIMIT 1
		`, email).Scan(&userID, &emailVerifiedAt)
		if err != nil || emailVerifiedAt == nil {
			// Unknown address or an unverified one: identical response,
			// nothing sent. See §8.5 -- Victory does not advertise recovery
			// availability for an address it never confirmed.
			writeJSON(w, http.StatusOK, generic)
			return
		}

		if _, err := pool.Exec(ctx, `
			UPDATE auth.password_reset_tokens
			SET consumed_at = NOW()
			WHERE user_id = $1 AND consumed_at IS NULL AND expires_at > NOW()
		`, userID); err != nil {
			writeJSON(w, http.StatusOK, generic)
			return
		}

		rawToken, tokenHash, err := newResetToken()
		if err != nil {
			writeJSON(w, http.StatusOK, generic)
			return
		}
		expiresAt := time.Now().UTC().Add(1 * time.Hour)
		if _, err := pool.Exec(ctx, `
			INSERT INTO auth.password_reset_tokens (user_id, token_hash, expires_at, request_ip)
			VALUES ($1, $2, $3, $4)
		`, userID, tokenHash, expiresAt, clientIP(r)); err != nil {
			writeJSON(w, http.StatusOK, generic)
			return
		}

		// /auth/* is reverse-proxied to the backend by Caddy, so the
		// redemption page lives under the statically served /login/ path --
		// the same reason cmd/victory-recover's link uses this shape.
		resetURL := strings.TrimRight(cfg.BaseURL, "/") + "/login/reset.html?token=" + rawToken

		go func(mailer RecoveryMailer, recipient, url string) {
			sendCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := mailer.SendPasswordReset(sendCtx, recipient, url); err != nil {
				log.Printf("recovery email delivery failed: %v", err)
			}
		}(cfg.Mailer, email, resetURL)

		writeJSON(w, http.StatusOK, generic)
	}
}

func HandleResetPassword(pool *pgxpool.Pool, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		var req ResetPasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}

		req.Token = strings.TrimSpace(req.Token)
		if req.Token == "" || strings.TrimSpace(req.NewPassword) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "token_and_new_password_required"})
			return
		}

		passwordHash, err := HashPassword(req.NewPassword)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		tx, err := pool.Begin(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "tx_begin_failed"})
			return
		}
		defer tx.Rollback(ctx)

		tokenHash := sha256.Sum256([]byte(req.Token))

		var userID string
		var tokenID string
		err = tx.QueryRow(ctx, `
			SELECT id, user_id
			FROM auth.password_reset_tokens
			WHERE token_hash = $1
			  AND consumed_at IS NULL
			  AND expires_at > NOW()
			LIMIT 1
		`, tokenHash[:]).Scan(&tokenID, &userID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_or_expired_token"})
			return
		}

		_, err = tx.Exec(ctx, `
			UPDATE auth.password_credentials
			SET password_hash = $2,
			    updated_at = NOW(),
			    password_changed_at = NOW()
			WHERE user_id = $1
		`, userID, passwordHash)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "password_update_failed"})
			return
		}

		_, err = tx.Exec(ctx, `
			UPDATE auth.password_reset_tokens
			SET consumed_at = NOW()
			WHERE id = $1
		`, tokenID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "token_consume_failed"})
			return
		}

		_, err = tx.Exec(ctx, `
			UPDATE auth.sessions
			SET revoked_at = NOW()
			WHERE user_id = $1
			  AND revoked_at IS NULL
		`, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "session_revoke_failed"})
			return
		}

		if err := tx.Commit(ctx); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "tx_commit_failed"})
			return
		}

		rawToken, expiresAt, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, r)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "session_create_failed"})
			return
		}

		sessions.SetSessionCookie(w, rawToken, expiresAt, secureCookie)

		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func newResetToken() (string, []byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}

	raw := base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	return raw, sum[:], nil
}

func clientIP(r *http.Request) any {
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func currentUserID(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (string, error) {
	raw, err := sessions.ReadSessionCookie(r)
	if err != nil {
		return "", err
	}

	rec, err := sessions.GetSessionByRawToken(ctx, pool, raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("not_authenticated")
		}
		return "", err
	}

	return rec.UserID, nil
}

func HandleMe(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok":    false,
				"error": "method_not_allowed",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":        true,
				"signed_in": false,
			})
			return
		}

		var handle, displayName string
		err = pool.QueryRow(ctx, `
			SELECT handle, display_name
			FROM users
			WHERE id = $1
			LIMIT 1
		`, userID).Scan(&handle, &displayName)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":        true,
				"signed_in": false,
			})
			return
		}

		role, err := access.CurrentLocationRole(ctx, pool, userID)
		if err != nil {
			role = "audience"
		}

		activeCharacter, _ := characters.ActiveCharacterForUser(ctx, pool, userID)

		isOperator, err := access.IsOperatorUser(ctx, pool, userID)
		if err != nil {
			isOperator = false
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":        true,
			"signed_in": true,
			"data": map[string]any{
				"user_id":          userID,
				"handle":           handle,
				"display_name":     displayName,
				"role":             role,
				"is_operator":      isOperator,
				"active_character": activeCharacter,
			},
		})
	}
}

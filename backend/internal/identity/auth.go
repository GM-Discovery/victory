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
	"strings"
	"time"

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

func HandleSignup(pool *pgxpool.Pool, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
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

func HandleForgotPassword(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		var req ForgotPasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}

		email := strings.TrimSpace(strings.ToLower(req.Email))
		if email == "" {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var userID string
		err := pool.QueryRow(ctx, `
			SELECT id
			FROM users
			WHERE lower(email) = $1
			LIMIT 1
		`, email).Scan(&userID)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}

		rawToken, tokenHash, err := newResetToken()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "token_generation_failed"})
			return
		}

		expiresAt := time.Now().UTC().Add(1 * time.Hour)

		_, err = pool.Exec(ctx, `
			INSERT INTO auth.password_reset_tokens (user_id, token_hash, expires_at, request_ip, request_user_agent)
			VALUES ($1, $2, $3, $4, $5)
		`, userID, tokenHash, expiresAt, clientIP(r), r.UserAgent())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "reset_store_failed"})
			return
		}

		log.Printf("PASSWORD RESET TOKEN for %s: %s", email, rawToken)

		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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

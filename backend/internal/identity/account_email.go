package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var emailFormatPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type updateEmailRequest struct {
	CurrentPassword string `json:"current_password"`
	NewEmail        string `json:"new_email"`
}

// HandleUpdateAccountEmail changes the authenticated owner's email
// (Kernel 61A §7). It never accepts a target user ID -- the account is
// always resolved from the session -- and it only ever writes to `users`,
// never to any Player Workbook table, so email can never leak into a
// profile event, History entry, or Face.
//
// Reauthentication is real: the current password is verified with the same
// Argon2id check used at login (VerifyPassword). Accounts with no password
// credential (Discord-only signup) have no safe reauthentication path
// available today, so this endpoint deliberately refuses them with a typed
// error rather than accepting a weaker confirmation -- see Kernel 61A §7's
// "stop and report PARTIAL honestly rather than adding weak confirmation."
func HandleUpdateAccountEmail(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
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

		var input updateEmailRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}

		newEmail := strings.ToLower(strings.TrimSpace(input.NewEmail))
		if !emailFormatPattern.MatchString(newEmail) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_email_format"})
			return
		}

		var storedHash string
		err = pool.QueryRow(ctx, `
			SELECT password_hash FROM auth.password_credentials WHERE user_id = $1
		`, userID).Scan(&storedHash)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "password_reauth_unavailable_for_this_account"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "reauth_check_failed"})
			return
		}

		if strings.TrimSpace(input.CurrentPassword) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "current_password_required"})
			return
		}

		ok, err := VerifyPassword(input.CurrentPassword, storedHash)
		if err != nil || !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "invalid_credentials"})
			return
		}

		var existingOwnerID string
		err = pool.QueryRow(ctx, `
			SELECT id FROM users WHERE lower(email) = $1 AND id != $2
		`, newEmail, userID).Scan(&existingOwnerID)
		if err == nil {
			writeJSON(w, http.StatusConflict, map[string]any{"ok": false, "error": "email_already_in_use"})
			return
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "uniqueness_check_failed"})
			return
		}

		if _, err := pool.Exec(ctx, `
			UPDATE users SET email = $1, email_verified_at = NULL, updated_at = NOW() WHERE id = $2
		`, newEmail, userID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "update_failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": map[string]any{"email": newEmail}})
	}
}

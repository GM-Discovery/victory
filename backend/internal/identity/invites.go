package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"victory/backend/internal/sessions"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateInviteRequest struct {
	TargetRole string `json:"target_role"`
	TargetEmail string `json:"target_email"`
	ProductionID string `json:"production_id"`
	VenueID string `json:"venue_id"`
	ExpiresInHours int `json:"expires_in_hours"`
	MaxUses int `json:"max_uses"`
}

type AcceptInviteRequest struct {
	Token string `json:"token"`
	Email string `json:"email"`
	Handle string `json:"handle"`
	Password string `json:"password"`
	DisplayName string `json:"display_name"`
}

func HandleCreateInvite(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		inviterUserID, err := currentUserID(ctx, pool, r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		var req CreateInviteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}

		req.TargetRole = strings.TrimSpace(strings.ToLower(req.TargetRole))
		req.TargetEmail = strings.TrimSpace(strings.ToLower(req.TargetEmail))

		if req.TargetRole == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "target_role_required"})
			return
		}

		switch req.TargetRole {
		case "director", "cast", "crew", "audience":
		default:
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_target_role"})
			return
		}

		if req.ExpiresInHours <= 0 {
			req.ExpiresInHours = 72
		}
		if req.MaxUses <= 0 {
			req.MaxUses = 1
		}

		var locationID string
		err = pool.QueryRow(ctx, `
			SELECT lm.location_id
			FROM location_memberships lm
			WHERE lm.user_id = $1
			  AND lm.role = 'producer'
			  AND lm.active = TRUE
			ORDER BY lm.created_at ASC
			LIMIT 1
		`, inviterUserID).Scan(&locationID)
		if err != nil {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "producer_membership_required"})
			return
		}

		if req.TargetRole == "director" && strings.TrimSpace(req.ProductionID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "director_requires_production_id"})
			return
		}

		rawToken, tokenHash, err := newInviteToken()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "token_generation_failed"})
			return
		}

		expiresAt := time.Now().UTC().Add(time.Duration(req.ExpiresInHours) * time.Hour)

		var inviteID string
		err = pool.QueryRow(ctx, `
			INSERT INTO invites (
				location_id,
				inviter_user_id,
				target_role,
				target_email,
				production_id,
				venue_id,
				token_hash,
				expires_at,
				max_uses
			)
			VALUES ($1, $2, $3::location_role, NULLIF($4, ''), NULLIF($5, '')::uuid, NULLIF($6, '')::uuid, $7, $8, $9)
			RETURNING id
		`, locationID, inviterUserID, req.TargetRole, req.TargetEmail, strings.TrimSpace(req.ProductionID), strings.TrimSpace(req.VenueID), tokenHash, expiresAt, req.MaxUses).Scan(&inviteID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invite_create_failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": map[string]any{
				"invite_id": inviteID,
				"token": rawToken,
				"expires_at": expiresAt,
				"target_role": req.TargetRole,
			},
		})
	}
}

func HandleAcceptInvite(pool *pgxpool.Pool, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		var req AcceptInviteRequest
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

		var existingUserID string
		rawSession, err := sessions.ReadSessionCookie(r)
		if err == nil && rawSession != "" {
			rec, serr := sessions.GetSessionByRawToken(ctx, pool, rawSession)
			if serr == nil {
				existingUserID = rec.UserID
			}
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "tx_begin_failed"})
			return
		}
		defer tx.Rollback(ctx)

		tokenHash := sha256.Sum256([]byte(req.Token))

		var inviteID, locationID, inviterUserID, targetRole string
		var targetEmail, productionID, venueID *string
		var maxUses, usesCount int
		var expiresAt time.Time
		var revokedAt, acceptedLastAt *time.Time

		err = tx.QueryRow(ctx, `
			SELECT id, location_id, inviter_user_id, target_role::text, target_email, production_id::text, venue_id::text,
			       max_uses, uses_count, expires_at, revoked_at, accepted_last_at
			FROM invites
			WHERE token_hash = $1
			LIMIT 1
		`, tokenHash[:]).Scan(
			&inviteID, &locationID, &inviterUserID, &targetRole, &targetEmail, &productionID, &venueID,
			&maxUses, &usesCount, &expiresAt, &revokedAt, &acceptedLastAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_invite"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "invite_lookup_failed"})
			return
		}

		now := time.Now().UTC()
		if revokedAt != nil || !expiresAt.After(now) || usesCount >= maxUses {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invite_invalid_or_expired"})
			return
		}

		userID := existingUserID

		if userID == "" {
			req.Email = strings.TrimSpace(strings.ToLower(req.Email))
			req.Handle = normalizeHandle(req.Handle)
			req.DisplayName = strings.TrimSpace(req.DisplayName)

			if req.Email == "" || req.Handle == "" || req.Password == "" || req.DisplayName == "" {
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "email_handle_password_display_name_required"})
				return
			}

			if targetEmail != nil && strings.TrimSpace(*targetEmail) != "" && req.Email != strings.ToLower(strings.TrimSpace(*targetEmail)) {
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invite_email_mismatch"})
				return
			}

			passwordHash, err := HashPassword(req.Password)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
				return
			}

			err = tx.QueryRow(ctx, `
				INSERT INTO users (email, handle, display_name)
				VALUES ($1, $2, $3)
				RETURNING id
			`, req.Email, req.Handle, req.DisplayName).Scan(&userID)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "user_create_failed"})
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
		}

		_, err = tx.Exec(ctx, `
			UPDATE invites
			SET uses_count = uses_count + 1,
			    accepted_first_at = COALESCE(accepted_first_at, NOW()),
			    accepted_last_at = NOW()
			WHERE id = $1
			  AND revoked_at IS NULL
			  AND expires_at > NOW()
			  AND uses_count < max_uses
		`, inviteID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "invite_update_failed"})
			return
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO memberships (location_id, user_id, role, production_id, venue_id, granted_by_user_id, active)
			VALUES ($1, $2, $3::location_role, NULLIF($4, '')::uuid, NULLIF($5, '')::uuid, $6, TRUE)
			ON CONFLICT DO NOTHING
		`, locationID, userID, targetRole, derefString(productionID), derefString(venueID), inviterUserID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "membership_create_failed"})
			return
		}

		if err := tx.Commit(ctx); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "tx_commit_failed"})
			return
		}

		if existingUserID == "" {
			rawToken, expiresAt, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, r)
			if err == nil {
				sessions.SetSessionCookie(w, rawToken, expiresAt, secureCookie)
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": map[string]any{
				"user_id": userID,
				"target_role": targetRole,
				"location_id": locationID,
				"production_id": derefString(productionID),
				"venue_id": derefString(venueID),
			},
		})
	}
}

func newInviteToken() (string, []byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}

	raw := base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	return raw, sum[:], nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

package identity

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

type createRequestInput struct {
	VenueSlug     string `json:"venue_slug"`
	RequestedRole string `json:"requested_role"`
	Note          string `json:"note"`
}

func HandleCreatePermissionRequest(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil || userID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var input createRequestInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		input.VenueSlug = strings.ToLower(strings.TrimSpace(input.VenueSlug))
		input.RequestedRole = strings.ToLower(strings.TrimSpace(input.RequestedRole))
		input.Note = strings.TrimSpace(input.Note)

		if input.VenueSlug == "" || input.RequestedRole == "" {
			http.Error(w, "venue_slug and requested_role are required", http.StatusBadRequest)
			return
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			http.Error(w, "internal_error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(ctx)

		autoApprove := input.VenueSlug == "catharsis" && (input.RequestedRole == "cast" || input.RequestedRole == "actor")
		requestedRole := input.RequestedRole
		if requestedRole == "actor" {
			requestedRole = "cast"
		}

		var requestID string
		err = tx.QueryRow(ctx, `
			INSERT INTO permission_requests (user_id, venue_slug, requested_role, note, status, reviewed_at)
			VALUES ($1, $2, $3, $4, $5, CASE WHEN $5 = 'approved' THEN NOW() ELSE NULL END)
			RETURNING id::text
		`, userID, input.VenueSlug, requestedRole, input.Note, map[bool]string{true: "approved", false: "pending"}[autoApprove]).Scan(&requestID)
		if err != nil {
			http.Error(w, "internal_error", http.StatusInternalServerError)
			return
		}

		if autoApprove {
			var venueID, locationID string
			err = tx.QueryRow(ctx, `
				SELECT v.id::text, l.location_id::text
				FROM venues v
				JOIN lots l ON l.id = v.lot_id
				WHERE v.slug = $1
				LIMIT 1
			`, input.VenueSlug).Scan(&venueID, &locationID)
			if err != nil {
				http.Error(w, "internal_error", http.StatusInternalServerError)
				return
			}

			_, err = tx.Exec(ctx, `
				INSERT INTO location_memberships (location_id, user_id, role, granted_by_user_id, active)
				VALUES ($1::uuid, $2::uuid, 'cast', $2::uuid, TRUE)
				ON CONFLICT (location_id, user_id, role) DO UPDATE
				SET active = TRUE,
				    granted_by_user_id = EXCLUDED.granted_by_user_id
			`, locationID, userID)
			if err != nil {
				http.Error(w, "internal_error", http.StatusInternalServerError)
				return
			}

			_, err = tx.Exec(ctx, `
				INSERT INTO access_grants (location_id, user_id, grant_type, venue_id, granted_by_user_id, created_at)
				VALUES ($1::uuid, $2::uuid, 'venue_access', $3::uuid, $2::uuid, NOW())
				ON CONFLICT DO NOTHING
			`, locationID, userID, venueID)
			if err != nil {
				http.Error(w, "internal_error", http.StatusInternalServerError)
				return
			}

			_, err = tx.Exec(ctx, `
				INSERT INTO messages (message_type, to_user_id, subject, body, venue_slug, is_read)
				VALUES (
				  'message',
				  $1::uuid,
				  $2,
				  $3,
				  $4,
				  FALSE
				)
			`, userID,
				"Catharsis actor access approved",
				"Your actor access to Catharsis was auto-approved. The message is waiting here for later reference.",
				input.VenueSlug)
			if err != nil {
				http.Error(w, "internal_error", http.StatusInternalServerError)
				return
			}
		}

		if err := tx.Commit(ctx); err != nil {
			http.Error(w, "internal_error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":            true,
			"request_id":    requestID,
			"auto_approved": autoApprove,
		})
	}
}

type permissionRequestRow struct {
	ID            string `json:"id"`
	VenueSlug     string `json:"venue_slug"`
	RequestedRole string `json:"requested_role"`
	Note          string `json:"note"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

func HandleListMyPermissionRequests(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil || userID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		rows, err := pool.Query(ctx, `
			SELECT id::text, venue_slug, requested_role, COALESCE(note, ''), status, created_at::text
			FROM permission_requests
			WHERE user_id = $1
			ORDER BY created_at DESC
		`, userID)
		if err != nil {
			http.Error(w, "internal_error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var out []permissionRequestRow
		for rows.Next() {
			var row permissionRequestRow
			if err := rows.Scan(
				&row.ID,
				&row.VenueSlug,
				&row.RequestedRole,
				&row.Note,
				&row.Status,
				&row.CreatedAt,
			); err != nil {
				http.Error(w, "internal_error", http.StatusInternalServerError)
				return
			}
			out = append(out, row)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":   true,
			"data": out,
		})
	}
}

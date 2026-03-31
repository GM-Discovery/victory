package identity

import (
	"encoding/json"
	"net/http"

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

		if input.VenueSlug == "" || input.RequestedRole == "" {
			http.Error(w, "venue_slug and requested_role are required", http.StatusBadRequest)
			return
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO permission_requests (user_id, venue_slug, requested_role, note)
			VALUES ($1, $2, $3, $4)
		`, userID, input.VenueSlug, input.RequestedRole, input.Note)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
				http.Error(w, err.Error(), http.StatusInternalServerError)
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
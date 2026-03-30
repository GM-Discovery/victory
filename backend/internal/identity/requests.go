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
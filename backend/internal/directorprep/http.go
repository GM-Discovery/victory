package directorprep

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/announcements"
)

type response struct {
	Ok   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, response{Ok: true, Data: data})
}

func writeError(w http.ResponseWriter, err error) {
	code := err.Error()
	status := http.StatusBadRequest
	switch {
	case code == "not_authenticated":
		status = http.StatusUnauthorized
	case code == "not_authorized":
		status = http.StatusForbidden
	case strings.HasSuffix(code, "_not_found"):
		status = http.StatusNotFound
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
}

func requireAuthenticatedUser(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (string, error) {
	sessionCookie := ""
	if c, err := r.Cookie("victory_session"); err == nil {
		sessionCookie = c.Value
	}
	userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
	if err != nil || strings.TrimSpace(userID) == "" {
		return "", errors.New("not_authenticated")
	}
	return userID, nil
}

// HandleAnnouncementStyles handles GET /api/announcement-styles.
//
// The palette is served rather than duplicated in the client so the two
// can never disagree about what "explosion" looks like. It is readable by
// any authenticated user because it contains no Show state at all -- it is
// the vocabulary, not an announcement -- and the Player-side renderer needs
// the same style record to draw an announcement it receives.
func HandleAnnouncementStyles(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if _, err := requireAuthenticatedUser(ctx, pool, r); err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{
			"styles":          announcements.Palette,
			"max_text_length": announcements.MaxTextLength,
		})
	}
}

// HandleShowPreparations handles GET and POST
// /api/shows/{show_id}/director-preparations.
func HandleShowPreparations(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))
		if err := RequireDirector(ctx, pool, userID, showID); err != nil {
			writeError(w, err)
			return
		}

		switch r.Method {
		case http.MethodGet:
			items, err := List(ctx, pool, showID, r.URL.Query().Get("kind"))
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"preparations": items})

		case http.MethodPost:
			var body struct {
				Kind      string         `json:"kind"`
				Label     string         `json:"label"`
				Payload   map[string]any `json:"payload"`
				SortOrder int            `json:"sort_order"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_body"))
				return
			}
			created, err := Create(ctx, pool, CreateInput{
				ShowID:    showID,
				Kind:      strings.TrimSpace(body.Kind),
				Label:     body.Label,
				Payload:   body.Payload,
				SortOrder: body.SortOrder,
				ActorID:   userID,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"preparation": created})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandlePreparationByID handles PATCH and DELETE
// /api/director-preparations/{preparation_id}.
//
// Authority is re-resolved from the STORED row's own show_id, never from a
// client-supplied one -- the same rule Kernel 71's ticket second punch
// established for authority derived from a record the caller names.
func HandlePreparationByID(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		preparationID := strings.TrimSpace(r.PathValue("preparation_id"))
		existing, err := LoadByID(ctx, pool, preparationID)
		if err != nil {
			writeError(w, err)
			return
		}
		if err := RequireDirector(ctx, pool, userID, existing.ShowID); err != nil {
			writeError(w, err)
			return
		}

		switch r.Method {
		case http.MethodPatch:
			var body struct {
				Label     *string         `json:"label"`
				Payload   *map[string]any `json:"payload"`
				SortOrder *int            `json:"sort_order"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_body"))
				return
			}
			updated, err := Update(ctx, pool, preparationID, UpdateInput{
				Label:     body.Label,
				Payload:   body.Payload,
				SortOrder: body.SortOrder,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"preparation": updated})

		case http.MethodDelete:
			if err := Delete(ctx, pool, preparationID); err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"deleted": true})

		default:
			methodNotAllowed(w)
		}
	}
}

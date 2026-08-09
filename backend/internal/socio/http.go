package socio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

type response struct {
	Ok   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
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

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeOK(w http.ResponseWriter, data any) { writeJSON(w, http.StatusOK, response{Ok: true, Data: data}) }

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
	case code == "character_not_on_show":
		status = http.StatusConflict
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

func methodNotAllowed(w http.ResponseWriter) { writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false}) }

// HandleStatusRegistry handles GET /api/socio/statuses.
func HandleStatusRegistry(pool *pgxpool.Pool) http.HandlerFunc {
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
		defs, err := ListStatusDefinitions(ctx, pool)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"statuses": defs})
	}
}

// HandleGameStatus handles GET
// /api/shows/{show_id}/cohorts/{cohort_id}/game-status ({cohort_id} may be
// the literal "ungrouped").
func HandleGameStatus(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))
		cohortID := strings.TrimSpace(r.PathValue("cohort_id"))
		blocks, err := BuildGameStatusForCohort(ctx, pool, userID, showID, cohortID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"characters": blocks})
	}
}

// HandleCharacterPool handles POST
// /api/shows/{show_id}/characters/{character_card_id}/socio/pools/{pool_key}
// (body: {"current": N, "max": N}).
func HandleCharacterPool(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))
		characterCardID := strings.TrimSpace(r.PathValue("character_card_id"))
		poolKey := strings.TrimSpace(r.PathValue("pool_key"))

		var body struct {
			Current int `json:"current"`
			Max     int `json:"max"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		state, err := SetPool(ctx, pool, userID, showID, characterCardID, PoolKey(poolKey), body.Current, body.Max)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"state": state})
	}
}

// HandleCharacterStatus handles POST
// /api/shows/{show_id}/characters/{character_card_id}/socio/statuses (body:
// {"status_key": "...", "intensity": N}) and DELETE
// /api/shows/{show_id}/characters/{character_card_id}/socio/statuses/{status_key}.
func HandleCharacterStatus(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))
		characterCardID := strings.TrimSpace(r.PathValue("character_card_id"))

		switch r.Method {
		case http.MethodPost:
			var body struct {
				StatusKey string `json:"status_key"`
				Intensity *int   `json:"intensity"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			applied, err := ApplyStatus(ctx, pool, userID, showID, characterCardID, body.StatusKey, body.Intensity)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"status": applied})
		case http.MethodDelete:
			statusKey := strings.TrimSpace(r.PathValue("status_key"))
			if err := ClearStatus(ctx, pool, userID, showID, characterCardID, statusKey); err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"cleared": true})
		default:
			methodNotAllowed(w)
		}
	}
}

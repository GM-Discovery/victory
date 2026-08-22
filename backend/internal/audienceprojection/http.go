package audienceprojection

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

type configRequest struct {
	ShortCode          string `json:"short_code"`
	ShowDiceRolls      bool   `json:"show_dice_rolls"`
	ShowPresence       bool   `json:"show_presence"`
	ShowHealthStatuses bool   `json:"show_health_statuses"`
}

// HandleGet handles GET /api/audience-config?short_code=JG5X -- the
// Director's read of the current Dice/Presence/Health toggle state (spec
// §16).
func HandleGet(pool *pgxpool.Pool) http.HandlerFunc {
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
		shortCode := strings.TrimSpace(r.URL.Query().Get("short_code"))
		c, err := ForShowCode(ctx, pool, userID, shortCode)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"config": c})
	}
}

// HandleUpdate handles PUT /api/audience-config -- the Director's
// Dice/Presence/Health toggle row (spec §16). Three explicit booleans, no
// generic settings map (spec §26: "do not create a generic settings
// engine").
func HandleUpdate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
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
		var body configRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		c, err := UpdateForShowCode(ctx, pool, userID, body.ShortCode, body.ShowDiceRolls, body.ShowPresence, body.ShowHealthStatuses)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"config": c})
	}
}

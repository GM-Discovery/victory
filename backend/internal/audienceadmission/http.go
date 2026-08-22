package audienceadmission

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

// requireAuthenticatedUser mirrors tickets/http.go's helper of the same name
// -- small, deliberate per-package duplication rather than a new
// cross-package dependency, matching this codebase's established
// convention.
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

// HandleIssue handles POST /api/audience-admissions -- the
// Director/Producer/Operator-only "issue or assign a single-Showing
// Audience admission to an account" step (spec §3.2, §34.2). Body carries a
// short_code (Kernel 92 Showtime's own Show selector, spec §25) and a
// handle -- never a raw user_id (spec's client-supplied-subject-can't-
// substitute-another-subject doctrine, mirrored from storyboards/grants.go's
// handle-paste convention -- no user-search endpoint exists in this repo).
func HandleIssue(pool *pgxpool.Pool) http.HandlerFunc {
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
		var body struct {
			ShortCode string `json:"short_code"`
			Handle    string `json:"handle"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		a, err := IssueForShowCode(ctx, pool, userID, body.ShortCode, body.Handle)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"admission": a})
	}
}

// HandleList handles GET /api/audience-admissions?short_code=JG5X -- who
// currently holds Audience admission to this Show's live Showing.
func HandleList(pool *pgxpool.Pool) http.HandlerFunc {
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
		list, err := ListForShowCode(ctx, pool, userID, shortCode)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"admissions": list})
	}
}

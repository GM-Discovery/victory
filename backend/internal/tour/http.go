package tour

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

// requireAuthenticatedUser mirrors playerprofile.requireAuthenticatedUser
// (backend/internal/playerprofile/http.go) -- the session cookie is the
// only identity source. No handler in this package ever reads a user_id
// from the request body (kernel-91 S46: "a user cannot mark another user's
// tour complete").
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
	switch code {
	case "not_authenticated":
		status = http.StatusUnauthorized
	case "forbidden":
		status = http.StatusForbidden
	case "invalid_tour_key", "tour_not_skippable":
		status = http.StatusBadRequest
	}
	writeJSON(w, status, map[string]any{"ok": false, "error": code})
}

type stateResponse struct {
	Mandatory *Definition  `json:"mandatory,omitempty"`
	Eligible  []Definition `json:"eligible"`
}

// HandleState serves GET /api/tours/state?venue={slug}. It always resolves
// the mandatory campus tour first (regardless of venue param) since it must
// be resumable from anywhere signed-in, then layers on venue-scoped
// eligible tours. venue is optional; omitted means campus/map scope.
func HandleState(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		subj := Subject{UserID: userID}
		venueSlug := strings.TrimSpace(r.URL.Query().Get("venue"))

		mandatory, err := ResumeMandatory(ctx, pool, subj)
		if err != nil {
			writeError(w, err)
			return
		}

		eligible, err := EligibleTours(ctx, pool, subj, venueSlug)
		if err != nil {
			writeError(w, err)
			return
		}

		writeOK(w, stateResponse{Mandatory: mandatory, Eligible: eligible})
	}
}

type completeRequest struct {
	StepReached string `json:"step_reached"`
}

// HandleComplete serves POST /api/tours/{tour_key}/complete. The tour_key
// path segment is validated against Definitions; the caller's role (for
// role-gated tours) is re-resolved server-side rather than trusted from the
// request, matching how EligibleTours computed it.
func HandleComplete(pool *pgxpool.Pool) http.HandlerFunc {
	return handleWrite(pool, StatusCompleted)
}

// HandleSkip serves POST /api/tours/{tour_key}/skip. Rejects outright if
// the tour is mandatory (kernel-91 S15: "cannot be skipped to completion")
// -- this is the actual enforcement point; hiding the Skip button
// client-side is defense-in-depth only.
func HandleSkip(pool *pgxpool.Pool) http.HandlerFunc {
	return handleWrite(pool, StatusSkipped)
}

func handleWrite(pool *pgxpool.Pool, status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		subj := Subject{UserID: userID}

		tourKey := strings.TrimSpace(r.PathValue("tour_key"))
		def, ok := Definitions[tourKey]
		if !ok || !IsTourKey(tourKey) {
			writeError(w, errors.New("invalid_tour_key"))
			return
		}
		if status == StatusSkipped && def.Mandatory {
			writeError(w, errors.New("tour_not_skippable"))
			return
		}

		var body completeRequest
		if r.ContentLength != 0 {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}

		role, isOperator, err := resolveRole(ctx, pool, userID, def.VenueSlug)
		if err != nil {
			writeError(w, err)
			return
		}
		roleKey := roleKeyForDefinition(def, role, isOperator)

		if err := RecordCompletion(ctx, pool, subj, tourKey, def.VenueSlug, roleKey, status, body.StepReached); err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"tour_key": tourKey, "status": status})
	}
}

type progressRequest struct {
	StepReached string `json:"step_reached"`
}

// HandleProgress serves POST /api/tours/{tour_key}/progress. This is the
// resumable-cursor write, distinct from HandleComplete/HandleSkip: it never
// marks the tour finished, only records how far the user got, so a
// click-gated step whose target navigates away mid-click (Audition Hall,
// Trailer) can resume on return instead of restarting the tour. Written via
// navigator.sendBeacon from the frontend for exactly that reason -- see
// migration 108's comment.
func HandleProgress(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		subj := Subject{UserID: userID}

		tourKey := strings.TrimSpace(r.PathValue("tour_key"))
		def, ok := Definitions[tourKey]
		if !ok {
			writeError(w, errors.New("invalid_tour_key"))
			return
		}

		var body progressRequest
		if r.ContentLength != 0 {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}

		role, isOperator, err := resolveRole(ctx, pool, userID, def.VenueSlug)
		if err != nil {
			writeError(w, err)
			return
		}
		roleKey := roleKeyForDefinition(def, role, isOperator)

		if err := RecordProgress(ctx, pool, subj, tourKey, def.VenueSlug, roleKey, body.StepReached); err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"tour_key": tourKey, "step_reached": body.StepReached})
	}
}

// HandleReplay serves GET /api/tours/{tour_key}/replay. Unlike HandleState's
// eligible list, this ignores prior completion (that's the point of replay)
// but still enforces the same role/venue check as EligibleTours -- a tour a
// user no longer has the role for cannot be replayed either. It never reads
// or writes tour_completions (kernel-91 S46: "replay does not grant
// authority"); it is a pure content lookup.
func HandleReplay(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}

		tourKey := strings.TrimSpace(r.PathValue("tour_key"))
		def, ok := Definitions[tourKey]
		if !ok {
			writeError(w, errors.New("invalid_tour_key"))
			return
		}

		role, isOperator, err := resolveRole(ctx, pool, userID, def.VenueSlug)
		if err != nil {
			writeError(w, err)
			return
		}
		if !roleEligible(def.RequiredRoles, role, isOperator) {
			writeError(w, errors.New("forbidden"))
			return
		}

		writeOK(w, def)
	}
}

// HandleHistory serves GET /api/tours/history, the account-page replay list.
func HandleHistory(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}

		completions, err := LoadCompletions(ctx, pool, Subject{UserID: userID})
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, completions)
	}
}

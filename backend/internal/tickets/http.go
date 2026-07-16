package tickets

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

// requireAuthenticatedUser mirrors showruns/http.go's helper of the same
// name -- small, deliberate per-package duplication rather than a new
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
	case code == "ticket_already_pending" || code == "ticket_not_pending":
		status = http.StatusConflict
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
}

// HandleRequest handles POST /api/show-runs/{id}/tickets/request -- the
// Player-initiated first punch. The body never carries a user_id; the
// subject is always the authenticated caller (spec §11's "client-supplied
// user ID cannot substitute another subject" is closed by construction
// here, not by a check).
func HandleRequest(pool *pgxpool.Pool) http.HandlerFunc {
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
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		showRunID := strings.TrimSpace(r.PathValue("id"))
		t, err := RequestFromPlayer(ctx, pool, userID, showRunID, body.Message)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"ticket": t})
	}
}

// HandleInvite handles POST /api/show-runs/{id}/tickets/invite -- the
// Director-initiated first punch. The body carries a target_profile_id
// (resolved server-side to a user id, the same pattern
// showruns.AddRosterMember already uses), never a raw user_id.
func HandleInvite(pool *pgxpool.Pool) http.HandlerFunc {
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
			TargetProfileID string `json:"target_profile_id"`
			Message         string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		showRunID := strings.TrimSpace(r.PathValue("id"))
		t, err := InviteFromDirector(ctx, pool, userID, showRunID, body.TargetProfileID, body.Message)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"ticket": t})
	}
}

// HandlePunch handles POST /api/tickets/{ticket_id}/punch -- the second
// punch, whichever side is waiting.
func HandlePunch(pool *pgxpool.Pool) http.HandlerFunc {
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
		t, err := SecondPunch(ctx, pool, userID, strings.TrimSpace(r.PathValue("ticket_id")))
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"ticket": t})
	}
}

// HandleDecline handles POST /api/tickets/{ticket_id}/decline.
func HandleDecline(pool *pgxpool.Pool) http.HandlerFunc {
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
		t, err := Decline(ctx, pool, userID, strings.TrimSpace(r.PathValue("ticket_id")))
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"ticket": t})
	}
}

// HandleWithdraw handles POST /api/tickets/{ticket_id}/withdraw.
func HandleWithdraw(pool *pgxpool.Pool) http.HandlerFunc {
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
		t, err := Withdraw(ctx, pool, userID, strings.TrimSpace(r.PathValue("ticket_id")))
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"ticket": t})
	}
}

// HandleListMine handles GET /api/tickets/mine.
func HandleListMine(pool *pgxpool.Pool) http.HandlerFunc {
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
		list, err := ListMineAsPlayer(ctx, pool, userID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"tickets": list})
	}
}

// HandleListIncoming handles GET /api/show-runs/{id}/tickets/incoming.
// ListIncomingForDirector already restricts its result to Show Runs the
// caller manages -- the path's {id} only narrows an already-authorized set
// down to one run, so a caller can never see a ticket for a run they don't
// manage merely by editing the URL.
func HandleListIncoming(pool *pgxpool.Pool) http.HandlerFunc {
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
		showRunID := strings.TrimSpace(r.PathValue("id"))
		all, err := ListIncomingForDirector(ctx, pool, userID)
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]Ticket, 0, len(all))
		for _, t := range all {
			if t.ShowRunID == showRunID {
				out = append(out, t)
			}
		}
		writeOK(w, map[string]any{"tickets": out})
	}
}

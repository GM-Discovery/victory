package merchant

// HTTP surface for Kernel 74's tutorial flow. Every handler follows the
// shape already established in http.go: authenticate, read path values,
// delegate to a function that resolves eligibility server-side, map typed
// errors through writeError.
//
// Live pushes use hub.BroadcastToSessionUser exclusively -- never
// BroadcastSession. The hub does not filter by role (network/hub.go), so
// role scoping must be decided when choosing recipients. A Directors+ note
// pushed session-wide and merely ignored by Player clients would be a
// leak, not a filter.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/network"
	"victory/backend/internal/projection"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
	"victory/backend/internal/tutorial"
)

// HandleInteractionComplete handles POST
// /api/participant-interactions/{interaction_id}/complete -- "Leave Kessa's
// Stall" (S6.1). Records the milestone that reveals the door hotspot for
// this Player only.
func HandleInteractionComplete(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
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
		interactionID := strings.TrimSpace(r.PathValue("interaction_id"))

		// Resolve the session before mutating so the post-write push can
		// target this Player's other tabs -- and only this Player's.
		eligible, err := ResolveEligibleContext(ctx, pool, userID, interactionID)
		if err != nil {
			writeError(w, err)
			return
		}
		progress, err := CompleteKessaIntro(ctx, pool, userID, interactionID)
		if err != nil {
			writeError(w, err)
			return
		}
		pushSelfStageRefresh(hub, eligible.SessionID, userID, "tutorial_progress")
		writeOK(w, map[string]any{"progress": progress})
	}
}

// pushSelfStageRefresh nudges only the acting Player's own clients to
// refetch their world snapshot. Best-effort: a nil hub (tests) or an absent
// session never blocks the HTTP response.
func pushSelfStageRefresh(hub *network.Hub, sessionID, userID, reason string) {
	if hub == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	msg, _ := json.Marshal(map[string]any{
		"type":   "tutorial/stage_refresh",
		"reason": reason,
	})
	hub.BroadcastToSessionUser(sessionID, userID, msg)
}

// HandleInteractionSubmit handles POST
// /api/participant-interactions/{interaction_id}/submit -- the freeform door
// intention (S7).
func HandleInteractionSubmit(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
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
		interactionID := strings.TrimSpace(r.PathValue("interaction_id"))

		var body struct {
			Text           string `json:"text"`
			IdempotencyKey string `json:"idempotency_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}

		eligible, err := ResolveEligibleContext(ctx, pool, userID, interactionID)
		if err != nil {
			writeError(w, err)
			return
		}
		result, noteRecipients, err := SubmitFreeform(ctx, pool, userID, interactionID, body.Text, body.IdempotencyKey)
		if err != nil {
			writeError(w, err)
			return
		}

		// One targeted push per resolved Directors+ recipient. The Player who
		// submitted is not in this list unless they are separately backstage
		// staff, which the recipient query -- not this loop -- decides.
		if hub != nil && eligible.SessionID != "" {
			msg, _ := json.Marshal(map[string]any{"type": "backstage/note_created"})
			for _, recipient := range noteRecipients {
				hub.BroadcastToSessionUser(eligible.SessionID, recipient, msg)
			}
		}
		writeOK(w, map[string]any{"submission": result})
	}
}

// HandleInteractionDialogue handles the three Ra endpoints under
// /api/participant-interactions/{interaction_id}/dialogue/{action} where
// action is open | topic | leave.
func HandleInteractionDialogue(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
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
		interactionID := strings.TrimSpace(r.PathValue("interaction_id"))
		action := strings.TrimSpace(r.PathValue("action"))

		// No "open" action here: opening any Program -- Equip Mode, the door
		// prompt, or Ra -- goes through the single POST .../open verb in
		// http.go, which dispatches on the interaction's stored type. A
		// second open path would be a second place for the "which Program"
		// decision to be made, and eventually to disagree.
		switch action {
		case "topic":
			var body struct {
				TopicKey string `json:"topic_key"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			state, err := ReadTopic(ctx, pool, userID, interactionID, body.TopicKey)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"dialogue": state})

		case "leave":
			eligible, err := ResolveEligibleContext(ctx, pool, userID, interactionID)
			if err != nil {
				writeError(w, err)
				return
			}
			out, err := LeaveDialogue(ctx, pool, userID, interactionID)
			if err != nil {
				writeError(w, err)
				return
			}
			// Only this Player's own clients are told to re-read the world.
			// No session-wide invalidation: nothing shared changed, and
			// nudging every client would imply otherwise.
			pushSelfStageRefresh(hub, eligible.SessionID, userID, "local_projection_entered")
			writeOK(w, map[string]any{"dialogue": out.State, "projection": out.Projection})

		default:
			writeJSON(w, http.StatusNotFound, response{Ok: false})
		}
	}
}

// --- Directors+ tutorial status list ----------------------------------------

// HandleShowTutorialProgress handles GET
// /api/shows/{show_id}/tutorial-progress -- the optional backstage status
// list (S11.5).
//
// Returns names and statuses only. No denominator, no "N of M ready", no
// group-completion signal (S1.11): each Player progresses independently and
// the system must not depend on a declared or inferred group size.
func HandleShowTutorialProgress(pool *pgxpool.Pool) http.HandlerFunc {
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

		show, err := shows.LoadShowByID(ctx, pool, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		sr, err := showruns.LoadShowRunByID(ctx, pool, show.ShowRunID)
		if err != nil {
			writeError(w, err)
			return
		}
		canView, err := showruns.CanViewBackstage(ctx, pool, userID, sr.LocationID)
		if err != nil {
			writeError(w, err)
			return
		}
		if !canView {
			writeError(w, errors.New("not_authorized"))
			return
		}

		list, err := tutorial.ListShowProgress(ctx, pool, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"progress": list})
	}
}

// HandleClearLocalProjection handles DELETE
// /api/shows/{show_id}/local-projections/{user_id} -- the authorized
// backstage "return this Player to the shared stage" control (S11.3).
func HandleClearLocalProjection(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		actorUserID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))
		targetUserID := strings.TrimSpace(r.PathValue("user_id"))

		show, err := shows.LoadShowByID(ctx, pool, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		sr, err := showruns.LoadShowRunByID(ctx, pool, show.ShowRunID)
		if err != nil {
			writeError(w, err)
			return
		}
		allowed, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
		if err != nil {
			writeError(w, err)
			return
		}
		if !allowed {
			writeError(w, errors.New("not_authorized"))
			return
		}

		if err := projection.ClearForViewer(ctx, pool, targetUserID, showID, "backstage_cleared"); err != nil {
			writeError(w, err)
			return
		}
		network.BroadcastShowStageInvalidation(ctx, hub, pool, showID, "local_projection_cleared")
		writeOK(w, map[string]any{"cleared": true})
	}
}

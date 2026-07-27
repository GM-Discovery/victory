package merchant

// HTTP surface for Kernel 75's tutorial completion (S3.2, S4).
//
// Same shape as http_tutorial.go: authenticate, read path values, delegate
// to a function that resolves eligibility server-side, map typed errors
// through writeError. Every gate is the existing one --
// ResolveEligibleContext -- because a second definition of "may this Player
// act here" is exactly the drift Kernel 74 warned against.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/network"
	"victory/backend/internal/recognition"
	"victory/backend/internal/storysofar"
)

// HandleTutorialContinue handles POST
// /api/participant-interactions/{interaction_id}/tutorial/continue -- the
// Player's Continue press after watching Ra open the gate.
//
// Retry-safe: a double-click produces one completion, one story, one
// recognition grant. See CompleteTutorial for the three database-level
// mechanisms that guarantee it.
func HandleTutorialContinue(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		interactionID := strings.TrimSpace(r.PathValue("interaction_id"))

		eligible, err := ResolveEligibleContext(ctx, pool, userID, interactionID)
		if err != nil {
			writeError(w, err)
			return
		}
		completion, err := CompleteTutorial(ctx, pool, userID, interactionID)
		if err != nil {
			writeError(w, err)
			return
		}
		// Only this Player's own tabs. The hub does no role filtering, so a
		// session-wide push here would tell everyone at the table that this
		// Player finished -- a leak, not a filter.
		pushSelfStageRefresh(hub, eligible.SessionID, userID, "tutorial_completed")
		writeOK(w, map[string]any{"completion": completion})
	}
}

// HandleTutorialCompletion handles GET
// /api/participant-interactions/{interaction_id}/tutorial/completion -- the
// reopen path (S4.3).
//
// A pure read. Reopening the Program a week later shows the same ending and
// re-awards nothing.
func HandleTutorialCompletion(pool *pgxpool.Pool) http.HandlerFunc {
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
		interactionID := strings.TrimSpace(r.PathValue("interaction_id"))
		completion, err := LoadTutorialCompletion(ctx, pool, userID, interactionID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"completion": completion})
	}
}

// HandleCharacterStorySoFar handles GET
// /api/characters/{character_card_id}/story-so-far.
//
// Path note: /api/character-cards/ is registered as a PREFIX handler in
// main.go and would swallow a sibling route under it. /api/characters/ is
// the shape that already coexists with both literal siblings
// (chapter2-rules) and a wildcard sibling
// (/api/characters/{character_card_id}/inventory).
//
// PRIVACY. characters.CanEditCard grants read to any location
// authority-holder, so it alone would let a Director read a Player's private
// reflections -- which S12 forbids. The owner check below is therefore
// separate from and stricter than the edit gate, and non-owners receive only
// entries the Player has explicitly shared with the table. The filtering
// happens inside storysofar.ListForCharacter's SQL rather than here, because
// an omit-from-payload gate applied per caller is the pattern that produced
// Kernel 74's authority hole.
func HandleCharacterStorySoFar(pool *pgxpool.Pool) http.HandlerFunc {
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
		characterCardID := strings.TrimSpace(r.PathValue("character_card_id"))
		if characterCardID == "" {
			writeError(w, errors.New("no_character_selected"))
			return
		}

		var ownerUserID string
		if err := pool.QueryRow(ctx, `
			SELECT owner_user_id::text FROM character_cards
			WHERE id = $1 AND is_deleted = FALSE
		`, characterCardID).Scan(&ownerUserID); err != nil {
			writeError(w, errors.New("character_card_not_found"))
			return
		}
		isOwner := ownerUserID == userID

		if !isOwner {
			// A non-owner must still have some legitimate reason to be
			// looking at this Character at all, on top of only ever seeing
			// shared entries.
			allowed, err := viewerMayReadCharacter(ctx, pool, userID, characterCardID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !allowed {
				writeError(w, errors.New("forbidden"))
				return
			}
		}

		events, err := storysofar.ListForCharacter(ctx, pool, characterCardID, isOwner)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{
			"story_events": events,
			"is_owner":     isOwner,
		})
	}
}

// HandleStoryEventVisibility handles PATCH /api/story-events/{event_id} --
// the Player's own reveal control (S1.8, S5.5).
//
// Owner-only, deliberately stricter than CanEditCard: a Director must not be
// able to publish a Player's private reflection. Ownership is enforced in
// the UPDATE's WHERE clause rather than by a preceding check, so there is no
// window between the check and the write.
//
// Note what this cannot do: change the text. There is no route anywhere that
// edits a generated summary (S5.5).
func HandleStoryEventVisibility(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
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
		eventID := strings.TrimSpace(r.PathValue("event_id"))

		var body struct {
			VisibilityState string `json:"visibility_state"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		event, err := storysofar.SetVisibility(ctx, pool, userID, eventID, body.VisibilityState)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"story_event": event})
	}
}

// HandleMyRecognition handles GET /api/player-recognition/me.
// Scoped to the authenticated user; there is no route that reads another
// Player's recognitions.
func HandleMyRecognition(pool *pgxpool.Pool) http.HandlerFunc {
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
		grants, err := recognition.LoadForUser(ctx, pool, userID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"recognition": grants})
	}
}

// viewerMayReadCharacter reports whether a non-owner has any legitimate
// reason to see this Character's shared story entries: they share an active
// Show Run roster, or they hold authority at the Character's Location.
//
// This is the "may you look at all" gate. It is separate from the "what do
// you see" filter, which always restricts a non-owner to visibility_state =
// 'table'. Passing this check never reveals a private entry.
func viewerMayReadCharacter(ctx context.Context, pool *pgxpool.Pool, viewerUserID, characterCardID string) (bool, error) {
	var allowed bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM character_cards cc
			JOIN location_memberships lm
			  ON lm.location_id = cc.location_id
			 AND lm.user_id = $2
			 AND lm.removed_at IS NULL
			WHERE cc.id = $1 AND cc.is_deleted = FALSE
		)
		OR EXISTS (
			SELECT 1
			FROM show_run_roster_members mine
			JOIN show_run_roster_members theirs
			  ON theirs.show_run_id = mine.show_run_id
			 AND theirs.removed_at IS NULL
			WHERE mine.user_id = $2
			  AND mine.removed_at IS NULL
			  AND theirs.character_card_id = $1
		)
	`, characterCardID, viewerUserID).Scan(&allowed)
	return allowed, err
}

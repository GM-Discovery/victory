package network

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/actions"
	"victory/backend/internal/stageobjects"
)

// applyLegacyRevealToCanonicalState converges the pre-Kernel-90
// act/reveal_element / act/hide_element WS control onto canonical
// stage-object state.
//
// Kernel 90 §19 asked which of the old live-stage actions represent durable
// stage-object state and which are legacy presentation. The answer for this
// pair is "durable": actions/reveal.go resolves its target through
// venue_layout_elements and refuses anything whose surface is not 'stage', so
// every reveal action that has ever existed targeted a real durable stage
// object. (The act/show_overlay / act/hide_overlay pair IS pure Cave-era
// presentation, and is deliberately left alone -- §19 is explicit that
// genuine presentation effects must not be forced into the object model.)
//
// Deliberately best-effort and non-fatal. The action row has already been
// committed and broadcast by the time this runs, and failing the WS message
// after that point would tell the client the operation failed while its
// evidence row says otherwise. A canonical-state failure is logged loudly
// instead; the Director's next hide/reveal retries it.
func applyLegacyRevealToCanonicalState(pool *pgxpool.Pool, hub *Hub, sessionID, actorID string, storedAction *actions.StoredAction, visible bool) {
	if pool == nil || storedAction == nil {
		return
	}
	elementID := ""
	if storedAction.Target != nil {
		elementID, _ = storedAction.Target["element_id"].(string)
	}
	elementID = strings.TrimSpace(elementID)
	if elementID == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	showID := showIDForSession(ctx, pool, sessionID)
	if showID == "" {
		// A pre-Show legacy session. Nothing to write: canonical state is
		// Show-scoped by §26, and §26 forbids reviving Session as the durable
		// owner just because a live Session exists.
		return
	}

	op := stageobjects.OpHide
	if visible {
		op = stageobjects.OpReveal
	}

	if _, err := stageobjects.ApplyMutation(ctx, pool, stageobjects.Mutation{
		ShowID:  showID,
		Ref:     stageobjects.Ref{Kind: stageobjects.KindVenueLayoutElement, ID: elementID},
		Op:      op,
		ActorID: actorID,
	}); err != nil {
		log.Printf("kernel90: legacy reveal did not reach canonical state: session=%s element=%s op=%s err=%v",
			sessionID, elementID, op, err)
		return
	}

	// Tell every viewer of this Show to re-read its own scope-filtered
	// snapshot. The existing action broadcast above is not enough on its own:
	// a viewer for whom the object just became perceivable needs a fresh
	// snapshot to receive the object at all, since a hidden object is omitted
	// from payloads rather than flagged (§36).
	BroadcastShowStageInvalidation(ctx, hub, pool, showID, "stage_object_state_changed")
}

func showIDForSession(ctx context.Context, pool *pgxpool.Pool, sessionID string) string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ""
	}
	var showID *string
	if err := pool.QueryRow(ctx, `
		SELECT show_id::text FROM sessions WHERE id = $1::uuid
	`, sessionID).Scan(&showID); err != nil {
		return ""
	}
	if showID == nil {
		return ""
	}
	return strings.TrimSpace(*showID)
}

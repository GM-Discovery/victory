package world

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/tutorial"
)

// loadTutorialStateForViewer builds Kernel 75's viewer-scoped tutorial
// position (S4.3, S10.1).
//
// Scoped to (viewer, selected Character, Show) exactly like every other
// Kernel 74/75 record. Two consequences worth stating, because both are
// requirements rather than side effects:
//
//   - Switching Character switches the reported state. A Player whose second
//     Character has never met Kessa gets no completion affordance, and never
//     sees the first Character's reflection (S12).
//   - A viewer with no selected Character gets nil, not an empty struct, so
//     the field is simply absent from their snapshot.
//
// Returns nil rather than an error when there is nothing to report: a
// missing tutorial state is the ordinary case for most viewers in most
// venues, not a failure.
func loadTutorialStateForViewer(ctx context.Context, pool *pgxpool.Pool, viewerUserID, characterCardID, showID string) (*TutorialState, error) {
	viewerUserID = strings.TrimSpace(viewerUserID)
	characterCardID = strings.TrimSpace(characterCardID)
	showID = strings.TrimSpace(showID)
	if viewerUserID == "" || characterCardID == "" || showID == "" {
		return nil, nil
	}

	p := tutorial.Participation{
		UserID:          viewerUserID,
		CharacterCardID: characterCardID,
		ShowID:          showID,
	}
	milestones, err := tutorial.LoadProgress(ctx, pool, p)
	if err != nil {
		return nil, err
	}
	if len(milestones) == 0 {
		return nil, nil
	}

	seen := map[string]bool{}
	for _, m := range milestones {
		seen[m] = true
	}

	state := &TutorialState{
		Milestones: milestones,
		// The Program becomes reachable once the handoff is entered -- that
		// is the moment Ra has finished and the gate is open. It stays
		// reachable forever after, because S4.3 requires a persistent way to
		// reopen it rather than a one-shot modal.
		CompletionAvailable: seen[tutorial.MilestoneTutorialHandoffEntered],
		Completed:           seen[tutorial.MilestoneTutorialCompleted],
	}

	if state.CompletionAvailable {
		// Find the guided_dialogue interaction this Player finished, so the
		// client can reopen the Program without rediscovering it. Scoped to
		// the Show's placements; a Show with no Ra interaction simply yields
		// an empty ID and the client falls back to hiding the affordance.
		var interactionID string
		err := pool.QueryRow(ctx, `
			SELECT pi.id::text
			FROM participant_interactions pi
			JOIN show_scene_placements ssp ON ssp.id = pi.show_scene_placement_id
			WHERE ssp.show_id = $1
			  AND pi.interaction_type = 'guided_dialogue'
			  AND pi.enabled
			ORDER BY pi.sort_order ASC, pi.id ASC
			LIMIT 1
		`, showID).Scan(&interactionID)
		if err == nil {
			state.CompletionInteractionID = interactionID
		}
	}

	return state, nil
}

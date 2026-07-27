package world

// Kernel 74's two snapshot-time additions: the milestone reveal gate for
// composition elements, and the participant-local projection lookup.
//
// Both are read-only and viewer-scoped. Neither writes anything, and
// neither can change what any other viewer sees -- which is the whole
// safety argument for adding per-viewer divergence to a function that
// previously produced the same elements for everyone at a given role.

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/projection"
	"victory/backend/internal/tutorial"
)

// milestoneGate answers "may this viewer see an element gated behind
// milestone K". It caches per snapshot because a Scene can carry several
// gated elements behind the same milestone and re-querying per row would
// turn one reveal check into N.
//
// bypass is the backstage authoring path (S6.2): Directors+ see gated
// elements regardless of Player progress so they can position and preview
// the door hotspot without playing the tutorial first.
type milestoneGate struct {
	pool          *pgxpool.Pool
	participation tutorial.Participation
	bypass        bool
	cache         map[string]bool
}

func (g *milestoneGate) allows(ctx context.Context, milestoneKey string) (bool, error) {
	if g.bypass {
		return true, nil
	}
	milestoneKey = strings.TrimSpace(milestoneKey)
	if milestoneKey == "" {
		return true, nil
	}
	// No selected Character means no participation context to check against,
	// so a gated element is not revealed. Fail-closed: an anonymous or
	// Audience viewer never sees the door.
	if strings.TrimSpace(g.participation.CharacterCardID) == "" ||
		strings.TrimSpace(g.participation.UserID) == "" ||
		strings.TrimSpace(g.participation.ShowID) == "" {
		return false, nil
	}
	if g.cache == nil {
		g.cache = map[string]bool{}
	}
	if allowed, ok := g.cache[milestoneKey]; ok {
		return allowed, nil
	}
	allowed, err := tutorial.HasMilestone(ctx, g.pool, g.participation, milestoneKey)
	if err != nil {
		return false, err
	}
	g.cache[milestoneKey] = allowed
	return allowed, nil
}

// loadLocalProjectionForViewer resolves the caller's own active projection
// and enriches it with the destination Scene's authored backdrop, so the
// stage renderer can swap maps through its existing texture lifecycle
// without a second round trip.
//
// Returns nil for every viewer without one, which is the overwhelmingly
// common case and the unchanged shared-stage path.
func loadLocalProjectionForViewer(ctx context.Context, pool *pgxpool.Pool, viewerUserID, showID string) (*LocalProjection, error) {
	active, err := projection.LoadActiveForViewer(ctx, pool, viewerUserID, showID)
	if err != nil || active == nil {
		return nil, err
	}

	out := &LocalProjection{
		SceneID:    active.DestinationSceneID,
		SceneSlug:  active.SceneSlug,
		SceneTitle: active.SceneTitle,
	}

	// The backdrop is read from the destination Scene's own map_backdrop
	// composition element rather than a column on the projection, so a
	// Director re-arting the handoff Scene in Scene Setup changes what
	// Players see with no code or data migration.
	var dataText string
	err = pool.QueryRow(ctx, `
		SELECT data::text FROM scene_stage_elements
		WHERE scene_id = $1 AND kind = 'map_backdrop' AND show_scene_placement_id IS NULL
		ORDER BY sort_order ASC, created_at ASC
		LIMIT 1
	`, active.DestinationSceneID).Scan(&dataText)
	if err != nil {
		// A projection Scene with no authored backdrop is legitimate (the
		// Director may be composing it live). Render the elements; the
		// client keeps whatever backdrop it had.
		return out, nil
	}

	data := decodeJSONMap([]byte(dataText))
	if data == nil {
		return out, nil
	}
	if v, ok := data["content_url"].(string); ok {
		out.BackdropURL = strings.TrimSpace(v)
	}
	if v, ok := data["grid_enabled"].(bool); ok {
		out.GridEnabled = v
	}
	return out, nil
}

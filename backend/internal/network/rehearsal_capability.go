package network

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// VenueSceneRehearsalEnabled checks the venues.config JSONB
// scene_rehearsal_enabled boolean flag, following the same
// venues.config-boolean-flag precedent used elsewhere in this codebase
// (index_cards_enabled, actors_can_reveal). Kernel 72A later converted
// session_control.go's hardcoded venue-slug allowlist to this same pattern
// (session_control_enabled). Seeded true for first-theater and
// catharsis, absent/false for middle-school-stage and other venues
// (Kernel 70 §1.5).
func VenueSceneRehearsalEnabled(ctx context.Context, pool *pgxpool.Pool, venueSlug string) (bool, error) {
	venueSlug = strings.ToLower(strings.TrimSpace(venueSlug))
	if venueSlug == "" {
		return false, nil
	}
	var enabled bool
	err := pool.QueryRow(ctx, `
		SELECT COALESCE((config ->> 'scene_rehearsal_enabled')::boolean, FALSE)
		FROM venues
		WHERE slug = $1
		LIMIT 1
	`, venueSlug).Scan(&enabled)
	if err != nil {
		return false, err
	}
	return enabled, nil
}

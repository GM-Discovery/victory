package merchant

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

// EnsureCourtyardScene idempotently seeds the reusable Courtyard Scene for
// fresh installs. Migration 057 already does this for existing databases,
// but on a from-empty install migrations run BEFORE
// access.EnsureKernel16VenueSurface seeds the catharsis venue row (Kernel
// 72's boot order), so 057's own JOIN against venues finds nothing and
// silently inserts zero rows -- the exact same migration-vs-fresh-install
// ordering trap Kernel 72A already hit once for scene_rehearsal_enabled.
// This must be called from main.go AFTER the venue bootstrap, mirroring
// that fix's two-path pattern (migration for existing rows, Go seed for
// fresh installs).
//
// Kernel 100: seeds onto access.DefaultLocationSlug() (this install's own
// configured location), not a fixed slug -- the query used to hardcode
// "amurray-family", which meant the Courtyard (and everything built on it:
// Kessa's opening, the Socio- quickstart) could only ever be seeded onto
// Grant's own location, never a fresh install's.
func EnsureCourtyardScene(ctx context.Context, pool *pgxpool.Pool) error {
	locationSlug := access.DefaultLocationSlug()
	_, err := pool.Exec(ctx, `
		INSERT INTO scenes (
			location_id, source_production_id, slug, title, short_title,
			default_venue_id, audience_title, audience_summary, player_brief,
			status, created_by_user_id
		)
		SELECT
			l.id, NULL, 'courtyard', 'The Courtyard', 'Courtyard', v.id,
			'The Courtyard', 'A shared gathering space at the heart of the story.',
			'The Courtyard is where the company gathers before a scene begins.',
			'ready', NULL
		FROM locations l
		JOIN venues v ON v.slug = 'catharsis' AND v.lot_id IN (
			SELECT id FROM lots WHERE location_id = l.id
		)
		WHERE l.slug = $1
		  AND NOT EXISTS (
			SELECT 1 FROM scenes s WHERE s.location_id = l.id AND s.slug = 'courtyard'
		  )
	`, locationSlug)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO scene_stage_elements (scene_id, kind, label, data, position, sort_order)
		SELECT s.id, 'map_backdrop', 'Courtyard',
			'{"content_url":"/assets/courtyard.png","display_mode":"theater","fit":"contain","crop_x":0.5,"crop_y":0.5,"scale":1,"grid_enabled":false}'::jsonb,
			'{}'::jsonb, 0
		FROM scenes s
		JOIN locations l ON l.id = s.location_id AND l.slug = $1
		WHERE s.slug = 'courtyard'
		  AND NOT EXISTS (
			SELECT 1 FROM scene_stage_elements e
			WHERE e.scene_id = s.id AND e.kind = 'map_backdrop' AND e.show_scene_placement_id IS NULL
		)
	`, locationSlug)
	return err
}

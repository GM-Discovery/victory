package merchant

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
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
func EnsureCourtyardScene(ctx context.Context, pool *pgxpool.Pool) error {
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
		WHERE l.slug = 'amurray-family'
		  AND NOT EXISTS (
			SELECT 1 FROM scenes s WHERE s.location_id = l.id AND s.slug = 'courtyard'
		  )
	`)
	return err
}

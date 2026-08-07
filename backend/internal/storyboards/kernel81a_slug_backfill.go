package storyboards

// Kernel 81A: backfills storyboard_columns/storyboard_bands/storyboard_rows
// rows created before migration 094 added `slug`, which is nullable at
// the schema level for exactly this reason -- the migration itself cannot
// safely compute collision-resolved slugs in pure SQL, so it leaves
// pre-existing rows NULL and this bootstrap (run at every server boot,
// same as every other Ensure*Surface call in main.go) fills them in.
//
// Reuses the exact same allocateUniqueSlug used by AddColumn/AddBand/
// AddRow, so a backfilled slug is produced by the identical deterministic
// algorithm a freshly-created object would get -- not a separate,
// possibly-diverging implementation. Idempotent: a row with a non-NULL
// slug is never touched again, so running this on every boot is a no-op
// once every existing board has been backfilled once. Nothing about IDs,
// titles/labels, sort order, or any other column is touched.

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureKernel81AStoryboardSlugsSurface backfills any structural row left
// over from before migration 094. Safe to call on every boot.
func EnsureKernel81AStoryboardSlugsSurface(ctx context.Context, pool *pgxpool.Pool) error {
	if err := backfillColumnSlugs(ctx, pool); err != nil {
		return err
	}
	if err := backfillBandSlugs(ctx, pool); err != nil {
		return err
	}
	return backfillRowSlugs(ctx, pool)
}

type slugBackfillCandidate struct {
	id      string
	scopeID string
	label   string
}

func backfillColumnSlugs(ctx context.Context, pool *pgxpool.Pool) error {
	candidates, err := loadSlugBackfillCandidates(ctx, pool, `
		SELECT id::text, storyboard_id::text, title
		FROM storyboard_columns
		WHERE slug IS NULL
		ORDER BY storyboard_id, created_at, id
	`)
	if err != nil {
		return err
	}
	for _, c := range candidates {
		if _, err := allocateUniqueSlug(ctx, pool, columnSlugExistsSQL, c.scopeID, c.label, func(ctx context.Context, slug string) (string, error) {
			_, err := pool.Exec(ctx, `UPDATE storyboard_columns SET slug = $2 WHERE id = $1 AND slug IS NULL`, c.id, slug)
			return c.id, err
		}); err != nil {
			return err
		}
	}
	return nil
}

func backfillBandSlugs(ctx context.Context, pool *pgxpool.Pool) error {
	candidates, err := loadSlugBackfillCandidates(ctx, pool, `
		SELECT id::text, storyboard_id::text, label
		FROM storyboard_bands
		WHERE slug IS NULL
		ORDER BY storyboard_id, created_at, id
	`)
	if err != nil {
		return err
	}
	for _, c := range candidates {
		if _, err := allocateUniqueSlug(ctx, pool, bandSlugExistsSQL, c.scopeID, c.label, func(ctx context.Context, slug string) (string, error) {
			_, err := pool.Exec(ctx, `UPDATE storyboard_bands SET slug = $2 WHERE id = $1 AND slug IS NULL`, c.id, slug)
			return c.id, err
		}); err != nil {
			return err
		}
	}
	return nil
}

// backfillRowSlugs scopes by band_id, not storyboard_id, matching
// AddRow's per-band uniqueness scope.
func backfillRowSlugs(ctx context.Context, pool *pgxpool.Pool) error {
	candidates, err := loadSlugBackfillCandidates(ctx, pool, `
		SELECT id::text, band_id::text, label
		FROM storyboard_rows
		WHERE slug IS NULL
		ORDER BY band_id, created_at, id
	`)
	if err != nil {
		return err
	}
	for _, c := range candidates {
		if _, err := allocateUniqueSlug(ctx, pool, rowSlugExistsSQL, c.scopeID, c.label, func(ctx context.Context, slug string) (string, error) {
			_, err := pool.Exec(ctx, `UPDATE storyboard_rows SET slug = $2 WHERE id = $1 AND slug IS NULL`, c.id, slug)
			return c.id, err
		}); err != nil {
			return err
		}
	}
	return nil
}

func loadSlugBackfillCandidates(ctx context.Context, pool *pgxpool.Pool, query string) ([]slugBackfillCandidate, error) {
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []slugBackfillCandidate
	for rows.Next() {
		var c slugBackfillCandidate
		if err := rows.Scan(&c.id, &c.scopeID, &c.label); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

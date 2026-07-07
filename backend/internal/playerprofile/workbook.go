package playerprofile

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureWorkbook returns the user's Player Workbook, creating it if this is
// their first visit. Exactly one workbook exists per account (Kernel 61
// §4.1, AC-1) -- the unique constraint on user_id in migration 036 backs
// this at the database level too.
func EnsureWorkbook(ctx context.Context, pool *pgxpool.Pool, userID string) (Workbook, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Workbook{}, errors.New("user_id_required")
	}

	cat, err := LoadCatalogue()
	if err != nil {
		return Workbook{}, err
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO player_profile_workbooks (user_id, catalogue_key, catalogue_version)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO NOTHING
	`, userID, cat.CatalogueKey, cat.CatalogueVersion); err != nil {
		return Workbook{}, err
	}

	return loadWorkbook(ctx, pool, userID)
}

func loadWorkbook(ctx context.Context, pool *pgxpool.Pool, userID string) (Workbook, error) {
	var wb Workbook
	err := pool.QueryRow(ctx, `
		SELECT id::text, user_id::text, catalogue_key, catalogue_version, projection_version, created_at, updated_at
		FROM player_profile_workbooks
		WHERE user_id = $1
	`, userID).Scan(&wb.ID, &wb.UserID, &wb.CatalogueKey, &wb.CatalogueVersion, &wb.ProjectionVersion, &wb.CreatedAt, &wb.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Workbook{}, errors.New("workbook_not_found")
		}
		return Workbook{}, err
	}
	return wb, nil
}

// EnsureKernel61PlayerWorkbookSurface backfills a Player Workbook for every
// existing user that doesn't already have one, then imports legacy
// performer_profiles data. It is idempotent and safe to run on every process
// start (Kernel 61 §9.2, §9.3, AC-28) -- it never deletes or recreates a
// `users` row, only inserts workbook/event/fact rows keyed off the existing
// user_id.
func EnsureKernel61PlayerWorkbookSurface(ctx context.Context, pool *pgxpool.Pool) error {
	cat, err := LoadCatalogue()
	if err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO player_profile_workbooks (user_id, catalogue_key, catalogue_version)
		SELECT u.id, $1, $2
		FROM users u
		WHERE NOT EXISTS (
			SELECT 1 FROM player_profile_workbooks w WHERE w.user_id = u.id
		)
	`, cat.CatalogueKey, cat.CatalogueVersion); err != nil {
		return err
	}

	return ImportLegacyProfiles(ctx, pool)
}

// touchWorkbookProjection bumps updated_at so ProjectionVersion (a
// GREATEST()-over-timestamps value, matching the Kernel 59A character
// pattern) reflects this mutation without a separate counter table.
func touchWorkbookProjection(ctx context.Context, pool *pgxpool.Pool, workbookID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE player_profile_workbooks SET updated_at = NOW() WHERE id = $1
	`, workbookID)
	return err
}

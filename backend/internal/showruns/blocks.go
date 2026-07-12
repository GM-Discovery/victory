package showruns

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const audienceBlockColumns = `
	id::text, show_run_id::text, user_id::text, blocked_by_user_id::text,
	COALESCE(reason, ''), created_at, lifted_at
`

func scanAudienceBlock(row pgx.Row) (AudienceBlock, error) {
	var b AudienceBlock
	if err := row.Scan(&b.ID, &b.ShowRunID, &b.UserID, &b.BlockedByUserID, &b.Reason, &b.CreatedAt, &b.LiftedAt); err != nil {
		return AudienceBlock{}, err
	}
	return b, nil
}

// BlockUser prevents targetUserID from self-joining as Audience on this run
// and (unless the actor is Operator) from being manually added as Audience.
// Run-scoped only -- never a site-wide moderation record.
func BlockUser(ctx context.Context, pool *pgxpool.Pool, actorUserID, showRunID, targetUserID, reason string) (AudienceBlock, error) {
	sr, err := LoadShowRunByID(ctx, pool, showRunID)
	if err != nil {
		return AudienceBlock{}, err
	}
	ok, err := canManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return AudienceBlock{}, err
	}
	if !ok {
		return AudienceBlock{}, errors.New("not_authorized")
	}
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" {
		return AudienceBlock{}, errors.New("user_id_required")
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO show_run_audience_blocks (show_run_id, user_id, blocked_by_user_id, reason)
		VALUES ($1, $2, $3, NULLIF($4, ''))
		ON CONFLICT (show_run_id, user_id) WHERE lifted_at IS NULL
		DO UPDATE SET reason = NULLIF($4, '')
		RETURNING `+audienceBlockColumns, showRunID, targetUserID, actorUserID, reason)
	return scanAudienceBlock(row)
}

// UnblockUser lifts an active block, restoring the target's self-join
// eligibility.
func UnblockUser(ctx context.Context, pool *pgxpool.Pool, actorUserID, showRunID, blockID string) error {
	sr, err := LoadShowRunByID(ctx, pool, showRunID)
	if err != nil {
		return err
	}
	ok, err := canManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not_authorized")
	}
	_, err = pool.Exec(ctx, `
		UPDATE show_run_audience_blocks
		SET lifted_at = NOW()
		WHERE id = $1 AND show_run_id = $2 AND lifted_at IS NULL
	`, blockID, showRunID)
	return err
}

// IsUserBlocked reports whether the user has an active block on this run.
func IsUserBlocked(ctx context.Context, pool *pgxpool.Pool, showRunID, userID string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM show_run_audience_blocks
			WHERE show_run_id = $1 AND user_id = $2 AND lifted_at IS NULL
		)
	`, showRunID, userID).Scan(&exists)
	return exists, err
}

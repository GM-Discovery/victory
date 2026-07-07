package playerprofile

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	stageNameMinLength = 1
	stageNameMaxLength = 80
)

// ValidateStageNameCandidate trims and validates a proposed stage name
// (Kernel 61 §3.2, §6.5, AC-4). It rejects blank/whitespace-only names and
// unreasonable lengths, returning the trimmed candidate ready to store.
func ValidateStageNameCandidate(candidate string) (string, error) {
	trimmed := strings.TrimSpace(candidate)
	if len(trimmed) < stageNameMinLength {
		return "", errors.New("stage_name_required")
	}
	if len([]rune(trimmed)) > stageNameMaxLength {
		return "", errors.New("stage_name_too_long")
	}
	return trimmed, nil
}

// NormalizeStageName produces the comparison key used for ledger
// idempotency (Kernel 61 §6.5, AC-6) -- case-insensitive, whitespace
// collapsed.
func NormalizeStageName(name string) string {
	fields := strings.Fields(strings.ToLower(name))
	return strings.Join(fields, " ")
}

// ChangeStageName closes the current open ledger interval (if the
// normalized name actually differs) and opens a new one, all inside one
// transaction (Kernel 61 §6.5, §9.4). Submitting the same normalized name is
// idempotent and creates no duplicate row (AC-6). It never touches any other
// account/access data (AC-7).
func ChangeStageName(ctx context.Context, pool *pgxpool.Pool, actorUserID, targetUserID, candidate string) (StageNameEntry, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	targetUserID = strings.TrimSpace(targetUserID)
	if actorUserID == "" {
		return StageNameEntry{}, errors.New("not_authenticated")
	}
	if targetUserID == "" {
		return StageNameEntry{}, errors.New("user_id_required")
	}
	if actorUserID != targetUserID {
		return StageNameEntry{}, errors.New("forbidden")
	}

	trimmed, err := ValidateStageNameCandidate(candidate)
	if err != nil {
		return StageNameEntry{}, err
	}
	normalized := NormalizeStageName(trimmed)

	tx, err := pool.Begin(ctx)
	if err != nil {
		return StageNameEntry{}, err
	}
	defer tx.Rollback(ctx)

	var current StageNameEntry
	var currentNormalized string
	err = tx.QueryRow(ctx, `
		SELECT id::text, stage_name, normalized_stage_name
		FROM player_stage_name_history
		WHERE user_id = $1 AND ended_at IS NULL
		FOR UPDATE
	`, targetUserID).Scan(&current.ID, &current.StageName, &currentNormalized)
	hasCurrent := true
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			hasCurrent = false
		} else {
			return StageNameEntry{}, err
		}
	}

	if hasCurrent && currentNormalized == normalized {
		if err := tx.Commit(ctx); err != nil {
			return StageNameEntry{}, err
		}
		current.UserID = targetUserID
		current.NormalizedName = currentNormalized
		return current, nil
	}

	if hasCurrent {
		if _, err := tx.Exec(ctx, `
			UPDATE player_stage_name_history
			SET ended_at = NOW()
			WHERE id = $1
		`, current.ID); err != nil {
			return StageNameEntry{}, err
		}
	}

	var out StageNameEntry
	if err := tx.QueryRow(ctx, `
		INSERT INTO player_stage_name_history (user_id, stage_name, normalized_stage_name, changed_by_user_id, source)
		VALUES ($1, $2, $3, $4, 'owner_edit')
		RETURNING id::text, user_id::text, stage_name, normalized_stage_name, started_at, changed_by_user_id::text, source, created_at
	`, targetUserID, trimmed, normalized, actorUserID).Scan(
		&out.ID, &out.UserID, &out.StageName, &out.NormalizedName, &out.StartedAt, &out.ChangedByUserID, &out.Source, &out.CreatedAt,
	); err != nil {
		return StageNameEntry{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return StageNameEntry{}, err
	}

	return out, nil
}

// LoadStageNameHistory returns every ledger interval for a user, oldest
// first. This is read-only, owner-facing history -- ordinary event deletion
// endpoints must never target these rows (Kernel 61 §6.5, AC-8).
func LoadStageNameHistory(ctx context.Context, pool *pgxpool.Pool, userID string) ([]StageNameEntry, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("user_id_required")
	}

	rows, err := pool.Query(ctx, `
		SELECT id::text, user_id::text, stage_name, normalized_stage_name, started_at, ended_at, changed_by_user_id::text, source, created_at
		FROM player_stage_name_history
		WHERE user_id = $1
		ORDER BY started_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StageNameEntry
	for rows.Next() {
		var e StageNameEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.StageName, &e.NormalizedName, &e.StartedAt, &e.EndedAt, &e.ChangedByUserID, &e.Source, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// CurrentStageName returns the open ledger interval's stage name for a user,
// or ok=false if none exists yet (e.g. a brand new workbook that hasn't
// completed identity setup).
func CurrentStageName(ctx context.Context, pool *pgxpool.Pool, userID string) (string, bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", false, errors.New("user_id_required")
	}

	var name string
	err := pool.QueryRow(ctx, `
		SELECT stage_name
		FROM player_stage_name_history
		WHERE user_id = $1 AND ended_at IS NULL
	`, userID).Scan(&name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return name, true, nil
}

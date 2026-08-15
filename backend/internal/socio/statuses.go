package socio

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ListStatusDefinitions returns the full status registry (socio_statuses,
// migration 097's seed set) -- data, not a hardcoded list, so adding a
// status is a seed-row INSERT (kernel-85 S2.3).
func ListStatusDefinitions(ctx context.Context, pool *pgxpool.Pool) ([]StatusDefinition, error) {
	rows, err := pool.Query(ctx, `SELECT key, label, description FROM socio_statuses ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StatusDefinition{}
	for rows.Next() {
		var d StatusDefinition
		if err := rows.Scan(&d.Key, &d.Label, &d.Description); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ListActiveStatuses returns a Character's currently-applied (uncleared)
// statuses, canonical tags and Kernel 88 blank flags alike (LEFT JOIN
// because blank-flag rows have a NULL status_key -- see migration 102). No
// authority check -- same "caller gates read access" pattern as GetState.
func ListActiveStatuses(ctx context.Context, pool *pgxpool.Pool, characterCardID string) ([]ActiveStatus, error) {
	rows, err := pool.Query(ctx, `
		SELECT e.id::text, COALESCE(e.status_key, ''), COALESCE(s.label, ''), COALESCE(e.custom_label, ''), e.intensity, e.applied_by_user_id::text, e.applied_at
		FROM character_socio_status_effects e
		LEFT JOIN socio_statuses s ON s.key = e.status_key
		WHERE e.character_card_id = $1 AND e.cleared_at IS NULL
		ORDER BY e.applied_at ASC
	`, characterCardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ActiveStatus{}
	for rows.Next() {
		var a ActiveStatus
		if err := rows.Scan(&a.ID, &a.StatusKey, &a.Label, &a.CustomLabel, &a.Intensity, &a.AppliedByUserID, &a.AppliedAt); err != nil {
			return nil, err
		}
		if a.StatusKey == "" {
			a.IsBlank = true
			a.Label = a.CustomLabel
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ApplyStatus applies (or re-intensifies) a status on a Character. Applying
// an already-active status updates its intensity rather than creating a
// second row -- enforced by migration 097's partial unique index on
// (character_card_id, status_key) WHERE cleared_at IS NULL, not by a
// read-then-write race here.
func ApplyStatus(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, characterCardID, statusKey string, intensity *int) (ActiveStatus, error) {
	if err := requireShowCharacterAuthority(ctx, pool, actorUserID, showID, characterCardID); err != nil {
		return ActiveStatus{}, err
	}
	statusKey = strings.TrimSpace(statusKey)
	if statusKey == "" {
		return ActiveStatus{}, errors.New("status_key_required")
	}

	var id string
	var appliedAt any
	err := pool.QueryRow(ctx, `
		INSERT INTO character_socio_status_effects (character_card_id, status_key, intensity, applied_by_user_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (character_card_id, status_key) WHERE cleared_at IS NULL
		DO UPDATE SET intensity = EXCLUDED.intensity, applied_by_user_id = EXCLUDED.applied_by_user_id, applied_at = NOW()
		RETURNING id::text, applied_at
	`, characterCardID, statusKey, intensity, actorUserID).Scan(&id, &appliedAt)
	if err != nil {
		if strings.Contains(err.Error(), "character_socio_status_effects_status_key_fkey") {
			return ActiveStatus{}, errors.New("invalid_status_key")
		}
		return ActiveStatus{}, err
	}

	statuses, err := ListActiveStatuses(ctx, pool, characterCardID)
	if err != nil {
		return ActiveStatus{}, err
	}
	for _, s := range statuses {
		if s.ID == id {
			return s, nil
		}
	}
	return ActiveStatus{}, errors.New("status_effect_not_found")
}

// ClearStatus marks a Character's active status effect cleared. Clearing an
// already-cleared or never-applied status is a no-op success, matching
// this codebase's idempotent-mutation convention elsewhere (e.g.
// tutorial.RecordMilestone).
func ClearStatus(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, characterCardID, statusKey string) error {
	if err := requireShowCharacterAuthority(ctx, pool, actorUserID, showID, characterCardID); err != nil {
		return err
	}
	statusKey = strings.TrimSpace(statusKey)
	if statusKey == "" {
		return errors.New("status_key_required")
	}
	_, err := pool.Exec(ctx, `
		UPDATE character_socio_status_effects
		SET cleared_at = NOW(), cleared_by_user_id = $3
		WHERE character_card_id = $1 AND status_key = $2 AND cleared_at IS NULL
	`, characterCardID, statusKey, actorUserID)
	return err
}

// AddFlag creates a Kernel 88 Director-authored blank state label (e.g.
// "Waiting on Kessa") -- distinct from a canonical socio_statuses tag: it
// has no registry entry, is free text capped at 60 characters (migration
// 102's check constraint), and unlike ApplyStatus there is no "re-apply
// updates intensity" collapsing -- each AddFlag call is a new row, since
// blank states aren't a fixed vocabulary to dedupe against. Director+ only.
func AddFlag(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, characterCardID, label string) (ActiveStatus, error) {
	if err := requireShowCharacterAuthority(ctx, pool, actorUserID, showID, characterCardID); err != nil {
		return ActiveStatus{}, err
	}
	label = strings.TrimSpace(label)
	if label == "" {
		return ActiveStatus{}, errors.New("label_required")
	}
	if len(label) > 60 {
		return ActiveStatus{}, errors.New("label_too_long")
	}

	var id string
	var appliedAt time.Time
	err := pool.QueryRow(ctx, `
		INSERT INTO character_socio_status_effects (character_card_id, custom_label, applied_by_user_id)
		VALUES ($1, $2, $3)
		RETURNING id::text, applied_at
	`, characterCardID, label, actorUserID).Scan(&id, &appliedAt)
	if err != nil {
		return ActiveStatus{}, err
	}
	return ActiveStatus{
		ID: id, CustomLabel: label, Label: label, IsBlank: true,
		AppliedByUserID: actorUserID, AppliedAt: appliedAt,
	}, nil
}

// ClearFlag clears one blank-state row by ID -- IDs, not labels, since
// blank states aren't unique per Character the way canonical status_keys
// are. Director+ only. Clearing an already-cleared or unknown ID is a
// no-op success, matching ClearStatus's idempotency.
func ClearFlag(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, characterCardID, flagID string) error {
	if err := requireShowCharacterAuthority(ctx, pool, actorUserID, showID, characterCardID); err != nil {
		return err
	}
	flagID = strings.TrimSpace(flagID)
	if flagID == "" {
		return errors.New("flag_id_required")
	}
	_, err := pool.Exec(ctx, `
		UPDATE character_socio_status_effects
		SET cleared_at = NOW(), cleared_by_user_id = $3
		WHERE id = $2 AND character_card_id = $1 AND status_key IS NULL AND cleared_at IS NULL
	`, characterCardID, flagID, actorUserID)
	return err
}

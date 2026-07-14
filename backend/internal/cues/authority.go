package cues

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
)

// activeRosterRole returns the actor's active show_run_roster_members role
// at the given Show Run, or "" if they have no active row -- scoped to the
// specific Show Run (not location-wide) so a player on one Show Run cannot
// trigger a player-eligible Cue on an unrelated Show Run at the same
// location.
func activeRosterRole(ctx context.Context, pool *pgxpool.Pool, userID, showRunID string) (string, error) {
	var role string
	err := pool.QueryRow(ctx, `
		SELECT role FROM show_run_roster_members
		WHERE show_run_id = $1 AND user_id = $2 AND removed_at IS NULL
		LIMIT 1
	`, showRunID, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return role, nil
}

// CanTriggerCue resolves the Cue's trigger_scope against the actor's
// authority. Director/Producer/Operator, or active Crew at the Cue's
// location, may always trigger, regardless of trigger_scope (Kernel 70
// §6.2). Audience is a hard floor -- never permitted to trigger any Cue,
// even under any_roster_member (Kernel 70 §1.7, §6.3's explicit
// requirement), enforced here independent of what activeRosterRole
// returns for an audience row.
func CanTriggerCue(ctx context.Context, pool *pgxpool.Pool, actorUserID, cueID string) (bool, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return false, nil
	}

	c, err := LoadCueByID(ctx, pool, cueID)
	if err != nil {
		return false, err
	}
	if !c.Enabled {
		return false, nil
	}

	_, showRunID, locationID, err := placementShowShowRunLocation(ctx, pool, c.ShowScenePlacementID)
	if err != nil {
		return false, err
	}

	canBackstage, err := showruns.CanViewBackstage(ctx, pool, actorUserID, locationID)
	if err != nil {
		return false, err
	}
	if canBackstage {
		return true, nil
	}

	role, err := activeRosterRole(ctx, pool, actorUserID, showRunID)
	if err != nil {
		return false, err
	}
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" || role == "audience" {
		return false, nil
	}

	switch c.TriggerScope {
	case TriggerScopePlayersMayTrigger:
		return role == "player", nil
	case TriggerScopeAnyRosterMember:
		return true, nil
	default:
		return false, nil
	}
}

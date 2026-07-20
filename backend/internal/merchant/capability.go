package merchant

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VenueParticipantInteractionsEnabled checks the venues.config
// participant_interactions_enabled boolean flag, following the Kernel 72A
// venue-capability-flag pattern (actions.stageElementsEnabled,
// network.VenueSessionControlEnabled): fail-closed for unknown venues and
// missing flags. Deliberately a separate flag from stage_elements_enabled --
// private Program interactions have different authority (participant-scoped,
// not role-scoped) and projection (targeted-push, never broadcast) rules
// than the general element-action surface that flag gates.
func VenueParticipantInteractionsEnabled(ctx context.Context, pool *pgxpool.Pool, venueSlug string) (bool, error) {
	venueSlug = strings.ToLower(strings.TrimSpace(venueSlug))
	if venueSlug == "" {
		return false, nil
	}
	var enabled bool
	err := pool.QueryRow(ctx, `
		SELECT COALESCE((config ->> 'participant_interactions_enabled')::boolean, FALSE)
		FROM venues
		WHERE slug = $1
		LIMIT 1
	`, venueSlug).Scan(&enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return enabled, nil
}

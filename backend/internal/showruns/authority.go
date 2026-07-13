package showruns

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

// CanManageShowRun mirrors network/director_console.go's
// canAccessDirectorConsole pattern, substituting the location-scoped role
// lookup so a Producer/Director at one location cannot manage a Show Run at
// a different location.
func CanManageShowRun(ctx context.Context, pool *pgxpool.Pool, userID, locationID string) (bool, error) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}

	role, err := access.CurrentLocationRoleForLocation(ctx, pool, userID, locationID)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "producer", "director":
		return true, nil
	default:
		return false, nil
	}
}

// CanViewShowRun is any active membership at the run's location, or
// Operator -- the minimum bar to see the Audience Program at all.
func CanViewShowRun(ctx context.Context, pool *pgxpool.Pool, userID, locationID string) (bool, error) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	return access.HasActiveLocationMembership(ctx, pool, userID, locationID)
}

// CanViewBackstage is CanManageShowRun plus an active Show Run crew roster
// row at this location -- enough to see the Stage Management backstage
// surface (listing/detail), but not to edit it. Crew edit authority itself
// is deferred (Kernel 68 §1.5, §3.8); this is visibility only, matching the
// map tile's own crew exception in access.ResolveVisibleVenues.
func CanViewBackstage(ctx context.Context, pool *pgxpool.Pool, userID, locationID string) (bool, error) {
	if ok, err := CanManageShowRun(ctx, pool, userID, locationID); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}

	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM show_run_roster_members rm
			JOIN show_runs sr ON sr.id = rm.show_run_id
			WHERE sr.location_id = $2
			  AND rm.user_id = $1
			  AND rm.role = 'crew'
			  AND rm.removed_at IS NULL
		)
	`, userID, locationID).Scan(&exists)
	return exists, err
}

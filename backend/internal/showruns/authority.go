package showruns

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

// canManageShowRun mirrors network/director_console.go's
// canAccessDirectorConsole pattern, substituting the location-scoped role
// lookup so a Producer/Director at one location cannot manage a Show Run at
// a different location.
func canManageShowRun(ctx context.Context, pool *pgxpool.Pool, userID, locationID string) (bool, error) {
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

// canViewShowRun is any active membership at the run's location, or
// Operator -- the minimum bar to see the Audience Program at all.
func canViewShowRun(ctx context.Context, pool *pgxpool.Pool, userID, locationID string) (bool, error) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	return access.HasActiveLocationMembership(ctx, pool, userID, locationID)
}

package access

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CurrentLocationRoleForLocation returns the caller's best active
// location_memberships role scoped to one specific location. Unlike
// CurrentLocationRole, which ignores location_id entirely and returns the
// caller's single globally-best active role across every location they
// belong to, this resolves role *for the location actually in play* --
// required wherever a location-scoped feature (e.g. Show Runs, Kernel 66)
// must not let a Producer at Location A silently pass a Producer/Director
// check for Location B. Returns "audience" with a nil error when the user
// has no active membership at this location at all, matching
// CurrentLocationRole's no-membership fallback convention -- callers that
// need to distinguish "has no membership" from "has audience membership"
// must use HasActiveLocationMembership instead.
func CurrentLocationRoleForLocation(ctx context.Context, pool *pgxpool.Pool, userID, locationID string) (string, error) {
	userID = strings.TrimSpace(userID)
	locationID = strings.TrimSpace(locationID)
	if userID == "" || locationID == "" {
		return "audience", nil
	}

	var role string
	err := pool.QueryRow(ctx, `
		SELECT m.role::text
		FROM location_memberships m
		WHERE m.user_id = $1
		  AND m.location_id = $2
		  AND m.active = TRUE
		ORDER BY
		  CASE m.role
			WHEN 'producer' THEN 1
			WHEN 'director' THEN 2
			WHEN 'cast' THEN 3
			WHEN 'crew' THEN 4
			WHEN 'audience' THEN 5
			ELSE 99
		  END,
		  m.created_at ASC
		LIMIT 1
	`, userID, locationID).Scan(&role)
	if err != nil {
		return "audience", nil
	}

	switch role {
	case "producer", "director", "cast", "crew", "audience":
		return role, nil
	default:
		return "audience", nil
	}
}

// HasAnyManageableLocation reports whether the user is Producer or Director
// (active) at *any* location, or is Operator. Used where a feature needs a
// coarse "can this person manage something, somewhere" check without
// disambiguating which specific location -- e.g. Third Place's "Add to Show
// Run" chip (Kernel 66), which offers a picker over the viewer's own
// manageable runs rather than needing the location up front.
func HasAnyManageableLocation(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false, nil
	}
	if ok, err := IsOperatorUser(ctx, pool, userID); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}

	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM location_memberships
			WHERE user_id = $1 AND active = TRUE AND role IN ('producer', 'director')
		)
	`, userID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// HasActiveLocationMembership reports whether the user has any active
// location_memberships row (any role) at the given location. Needed because
// CurrentLocationRoleForLocation's "audience" fallback cannot by itself
// distinguish an actual audience-role member from a user with no
// relationship to the location at all -- Show Run visibility and self-join
// both need that distinction.
func HasActiveLocationMembership(ctx context.Context, pool *pgxpool.Pool, userID, locationID string) (bool, error) {
	userID = strings.TrimSpace(userID)
	locationID = strings.TrimSpace(locationID)
	if userID == "" || locationID == "" {
		return false, nil
	}

	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM location_memberships
			WHERE user_id = $1 AND location_id = $2 AND active = TRUE
		)
	`, userID, locationID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

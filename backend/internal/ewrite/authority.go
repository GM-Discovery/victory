package ewrite

// Authority helpers, modeled on showruns/authority.go: Operator
// short-circuit first, then location-scoped role via
// access.CurrentLocationRoleForLocation (never the global-best
// CurrentLocationRole -- a Crew member at Location A must not author at
// Location B). Client-supplied role or user IDs are never authority; every
// helper takes the server-resolved userID and a locationID the server
// looked up from the row being acted on.
//
// Draft READ authority deliberately equals edit authority
// (CanEditPublication): eWrite drafts are production work product shared
// within the authoring team, not private reflections. This is a recorded
// contrast with storysofar/store.go's owner-only precedent -- see
// Construction/eWrite/ewrite-permissions.md.

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

func isAuthoringRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "producer", "director", "crew":
		return true
	default:
		return false
	}
}

func isManagerRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "producer", "director":
		return true
	default:
		return false
	}
}

// CanAuthorInScope: may this user create eWrite content at this location?
// Crew+ (producer/director/crew) or Operator. Gates collection creation,
// publication creation, and the Writer's Room tree.
func CanAuthorInScope(ctx context.Context, pool *pgxpool.Pool, userID, locationID string) (bool, error) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	role, err := access.CurrentLocationRoleForLocation(ctx, pool, userID, locationID)
	if err != nil {
		return false, err
	}
	return isAuthoringRole(role), nil
}

// hasEditorGrant reports whether the user holds any of the named grant
// kinds on the publication.
func hasEditorGrant(ctx context.Context, pool *pgxpool.Pool, userID, publicationID string, kinds ...string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM ewrite_editors
			WHERE publication_id = $1 AND user_id = $2 AND grant_kind = ANY($3)
		)
	`, publicationID, userID, kinds).Scan(&exists)
	return exists, err
}

// CanEditPublication: Operator, producer/director at the publication's
// location, the creator, or any named grant. Crew edit their own or granted
// work only -- role alone never grants edit across all content (spec 1.4).
// Also the draft/archived READ predicate (see package comment).
func CanEditPublication(ctx context.Context, pool *pgxpool.Pool, userID string, pub *Publication) (bool, error) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	if pub.CreatedBy != "" && pub.CreatedBy == userID {
		// Creator must still hold an authoring role at the location: a
		// demoted-to-audience author keeps nothing.
		role, err := access.CurrentLocationRoleForLocation(ctx, pool, userID, pub.LocationID)
		if err != nil {
			return false, err
		}
		if isAuthoringRole(role) {
			return true, nil
		}
	}
	role, err := access.CurrentLocationRoleForLocation(ctx, pool, userID, pub.LocationID)
	if err != nil {
		return false, err
	}
	if isManagerRole(role) {
		return true, nil
	}
	if !isAuthoringRole(role) {
		return false, nil
	}
	return hasEditorGrant(ctx, pool, userID, pub.ID, "edit", "publish", "manage_editors")
}

// CanPublishPublication: Operator, producer/director, the creator, or a
// named 'publish' grant. An 'edit'-only grant cannot publish (spec 17.2).
func CanPublishPublication(ctx context.Context, pool *pgxpool.Pool, userID string, pub *Publication) (bool, error) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	role, err := access.CurrentLocationRoleForLocation(ctx, pool, userID, pub.LocationID)
	if err != nil {
		return false, err
	}
	if isManagerRole(role) {
		return true, nil
	}
	if !isAuthoringRole(role) {
		return false, nil
	}
	if pub.CreatedBy != "" && pub.CreatedBy == userID {
		return true, nil
	}
	return hasEditorGrant(ctx, pool, userID, pub.ID, "publish")
}

// CanManageEditors: Operator, producer/director, or a named
// 'manage_editors' grant.
func CanManageEditors(ctx context.Context, pool *pgxpool.Pool, userID string, pub *Publication) (bool, error) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	role, err := access.CurrentLocationRoleForLocation(ctx, pool, userID, pub.LocationID)
	if err != nil {
		return false, err
	}
	if isManagerRole(role) {
		return true, nil
	}
	if !isAuthoringRole(role) {
		return false, nil
	}
	return hasEditorGrant(ctx, pool, userID, pub.ID, "manage_editors")
}

// CanDeletePublication: Operator, producer/director, or the creator.
// Deliberately narrower than CanEditPublication -- a named editor may not
// delete someone else's work.
func CanDeletePublication(ctx context.Context, pool *pgxpool.Pool, userID string, pub *Publication) (bool, error) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	role, err := access.CurrentLocationRoleForLocation(ctx, pool, userID, pub.LocationID)
	if err != nil {
		return false, err
	}
	if isManagerRole(role) {
		return true, nil
	}
	return isAuthoringRole(role) && pub.CreatedBy != "" && pub.CreatedBy == userID, nil
}

// CanReadPublication enforces effective visibility (spec 10.4).
// Published:
//   - public: any authenticated user. Kernel 78 deliberately serves
//     'public' to authenticated readers only -- the anonymous route is a
//     recorded deferral, not an oversight.
//   - authenticated: any authenticated user.
//   - production: any ACTIVE membership at the publication's location,
//     any role including audience.
//
// Draft/archived: edit authority only (spec 10.5).
func CanReadPublication(ctx context.Context, pool *pgxpool.Pool, userID string, pub *Publication) (bool, error) {
	if strings.TrimSpace(userID) == "" {
		return false, nil
	}
	if pub.Status != "published" {
		return CanEditPublication(ctx, pool, userID, pub)
	}
	switch pub.Visibility {
	case "public", "authenticated":
		return true, nil
	case "production":
		if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
			return false, err
		} else if ok {
			return true, nil
		}
		return access.HasActiveLocationMembership(ctx, pool, userID, pub.LocationID)
	default:
		return false, nil
	}
}

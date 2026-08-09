package storyboards

// Named per-user access grants (spec 1.8, 4.2): sharing is a deliberate
// act, never implied by a My People relationship. Handle-lookup idiom
// copied from ewrite/editors.go's AddEditor -- there is no generic
// user-search endpoint in this repo, handle-paste is the established
// mechanism for "add a specific person."

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound       = errors.New("user_not_found")
	ErrInvalidGrantedRole = errors.New("invalid_granted_role")
	ErrGrantNotFound      = errors.New("grant_not_found")
)

var validGrantedRoles = map[string]bool{
	TierAudience: true,
	TierCast:     true,
	TierCrew:     true,
	TierDirector: true,
	TierProducer: true,
}

// ListGrants returns a board's grants with handles for display. Requires
// CanManageSharing (owner/Operator only) -- grant visibility is itself a
// sharing-management detail, not exposed to Crew/Director+. export.go's
// BuildBoardExport needs a board's grants too but is gated by
// CanExportBoard instead, so it calls listGrantsRaw directly rather than
// through this authority check.
func ListGrants(ctx context.Context, pool *pgxpool.Pool, userID, boardID string) ([]StoryboardGrant, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanManageSharing(ctx, pool, userID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}
	return listGrantsRaw(ctx, pool, boardID)
}

// listGrantsRaw fetches a board's grants with no authority check -- every
// caller is responsible for having already gated on its own appropriate
// authority (ListGrants uses CanManageSharing; BuildBoardExport uses
// CanExportBoard).
func listGrantsRaw(ctx context.Context, pool *pgxpool.Pool, boardID string) ([]StoryboardGrant, error) {
	rows, err := pool.Query(ctx, `
		SELECT g.id::text, g.storyboard_id::text, g.user_id::text, COALESCE(u.handle, ''),
		       g.granted_role::text, g.granted_by::text, g.created_at, g.updated_at
		FROM storyboard_grants g
		LEFT JOIN users u ON u.id = g.user_id
		WHERE g.storyboard_id = $1
		ORDER BY g.created_at
	`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoryboardGrant{}
	for rows.Next() {
		var g StoryboardGrant
		if err := rows.Scan(&g.ID, &g.StoryboardID, &g.UserID, &g.UserHandle,
			&g.GrantedRole, &g.GrantedBy, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// AddGrant grants grantedRole to the user identified by handle. Requires
// CanManageSharing. Re-sharing to an already-granted user updates their
// role rather than erroring.
func AddGrant(ctx context.Context, pool *pgxpool.Pool, grantorID, boardID, userHandle, grantedRole string) (*StoryboardGrant, error) {
	grantedRole = strings.ToLower(strings.TrimSpace(grantedRole))
	if !validGrantedRoles[grantedRole] {
		return nil, ErrInvalidGrantedRole
	}
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanManageSharing(ctx, pool, grantorID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}

	userHandle = strings.TrimSpace(userHandle)
	var userID string
	err = pool.QueryRow(ctx, `SELECT id::text FROM users WHERE handle = $1`, userHandle).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if userID == board.OwnerUserID {
		// Owner is never a grant row (see migration 090's comment) --
		// silently refuse rather than create a confusing, capability-less
		// duplicate row for the owner's own account.
		return nil, errors.New("cannot_grant_to_owner")
	}

	var g StoryboardGrant
	err = pool.QueryRow(ctx, `
		INSERT INTO storyboard_grants (storyboard_id, user_id, granted_role, granted_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (storyboard_id, user_id) DO UPDATE
		  SET granted_role = EXCLUDED.granted_role, granted_by = EXCLUDED.granted_by, updated_at = NOW()
		RETURNING id::text, created_at, updated_at
	`, boardID, userID, grantedRole, grantorID).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, err
	}
	g.StoryboardID = boardID
	g.UserID = userID
	g.UserHandle = userHandle
	g.GrantedRole = grantedRole
	g.GrantedBy = grantorID
	return &g, nil
}

// resolveGrantSubjectUserID maps an opaque Player Workbook / profile ID to
// the account's stable UUID -- a local copy of the same helper
// playerrelationships.resolveSubjectUserID and tickets.resolveProfileUserID
// duplicate (Kernel 61 contract reuse), matching this codebase's established
// per-package-duplication convention (see either of those two functions'
// doc comments for why: no shared package sits below both without an import
// cycle risk, and the query itself is ten lines).
func resolveGrantSubjectUserID(ctx context.Context, pool *pgxpool.Pool, profileID string) (string, error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return "", errors.New("profile_id_required")
	}
	var userID string
	if err := pool.QueryRow(ctx, `
		SELECT user_id::text FROM player_profile_workbooks WHERE id = $1
	`, profileID).Scan(&userID); err != nil {
		return "", ErrUserNotFound
	}
	return userID, nil
}

// AddGrantByProfile is AddGrant's sibling for Kernel 85 §7.4: instead of a
// hand-typed user_handle (a field the client is never otherwise shown --
// playerprofile/types.go's Face projections deliberately never include
// handle), the caller identifies the subject by the same opaque profile_id
// already used throughout this codebase's People Picker precedents
// (playerrelationships.resolveSubjectUserID, tickets.resolveProfileUserID).
// This is what lets a Storyboard share actually succeed for someone visible
// in My People/Third Place, instead of failing ErrUserNotFound against a
// field (handle) that was never shown to the sharer in the first place.
func AddGrantByProfile(ctx context.Context, pool *pgxpool.Pool, grantorID, boardID, subjectProfileID, grantedRole string) (*StoryboardGrant, error) {
	grantedRole = strings.ToLower(strings.TrimSpace(grantedRole))
	if !validGrantedRoles[grantedRole] {
		return nil, ErrInvalidGrantedRole
	}
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanManageSharing(ctx, pool, grantorID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}

	userID, err := resolveGrantSubjectUserID(ctx, pool, subjectProfileID)
	if err != nil {
		return nil, err
	}
	if userID == board.OwnerUserID {
		// Owner is never a grant row (see migration 090's comment) --
		// silently refuse rather than create a confusing, capability-less
		// duplicate row for the owner's own account.
		return nil, errors.New("cannot_grant_to_owner")
	}

	var g StoryboardGrant
	err = pool.QueryRow(ctx, `
		INSERT INTO storyboard_grants (storyboard_id, user_id, granted_role, granted_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (storyboard_id, user_id) DO UPDATE
		  SET granted_role = EXCLUDED.granted_role, granted_by = EXCLUDED.granted_by, updated_at = NOW()
		RETURNING id::text, created_at, updated_at
	`, boardID, userID, grantedRole, grantorID).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, err
	}
	g.StoryboardID = boardID
	g.UserID = userID
	_ = pool.QueryRow(ctx, `SELECT COALESCE(handle, '') FROM users WHERE id = $1`, userID).Scan(&g.UserHandle)
	g.GrantedRole = grantedRole
	g.GrantedBy = grantorID
	return &g, nil
}

// RemoveGrant revokes one grant by id. Requires CanManageSharing.
func RemoveGrant(ctx context.Context, pool *pgxpool.Pool, userID, boardID, grantID string) error {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return err
	}
	if ok, err := CanManageSharing(ctx, pool, userID, board); err != nil {
		return err
	} else if !ok {
		return ErrNotAuthorized
	}
	tag, err := pool.Exec(ctx, `DELETE FROM storyboard_grants WHERE id = $1 AND storyboard_id = $2`, grantID, boardID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGrantNotFound
	}
	return nil
}

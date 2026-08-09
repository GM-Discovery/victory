package storyboards

// Kernel 85 §7.4: AddGrantByProfile is grants.go's sibling to AddGrant,
// identifying the subject by the same opaque profile_id (player_profile_
// workbooks.id) the People Picker frontend selects with, instead of a
// hand-typed user_handle -- a field My People/Third Place never show the
// sharer in the first place (see grants.go's doc comment on
// resolveGrantSubjectUserID for the full rationale).

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

func ensureStoryboardsTestProfile(t *testing.T, pool *pgxpool.Pool, userID string) string {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO player_profile_workbooks (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING
	`, userID); err != nil {
		t.Fatalf("ensure profile for %s: %v", userID, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM player_profile_workbooks WHERE user_id = $1`, userID)
	})
	var profileID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM player_profile_workbooks WHERE user_id = $1`, userID).Scan(&profileID); err != nil {
		t.Fatalf("load profile id for %s: %v", userID, err)
	}
	return profileID
}

func TestAddGrantByProfileSucceedsWithSelectedIdentity(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb85_owner")
	member := insertTestUser(t, pool, "sb85_member")
	board := mustCreateBoard(t, pool, owner, "Profile Share Board")

	profileID := ensureStoryboardsTestProfile(t, pool, member)

	g, err := AddGrantByProfile(ctx, pool, owner, board.ID, profileID, "crew")
	if err != nil {
		t.Fatalf("AddGrantByProfile: %v", err)
	}
	if g.UserID != member {
		t.Fatalf("expected grant.user_id = %s, got %s", member, g.UserID)
	}
	if g.GrantedRole != "crew" {
		t.Fatalf("expected crew, got %s", g.GrantedRole)
	}

	tier, err := ResolveViewerTier(ctx, pool, member, board)
	if err != nil || tier != TierCrew {
		t.Fatalf("expected crew tier after profile-id grant, got %s err=%v", tier, err)
	}
}

func TestAddGrantByProfileRejectsMalformedProfileID(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb85_owner2")
	board := mustCreateBoard(t, pool, owner, "Malformed Profile Board")

	if _, err := AddGrantByProfile(ctx, pool, owner, board.ID, "not-a-real-profile-id", "crew"); err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound for a malformed/nonexistent profile_id, got %v", err)
	}
	if _, err := AddGrantByProfile(ctx, pool, owner, board.ID, "", "crew"); err == nil {
		t.Fatal("expected an error for an empty profile_id")
	}
}

func TestAddGrantByProfileRequiresSharingAuthority(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb85_owner3")
	notOwner := insertTestUser(t, pool, "sb85_stranger")
	target := insertTestUser(t, pool, "sb85_target")
	board := mustCreateBoard(t, pool, owner, "Authority Board")

	profileID := ensureStoryboardsTestProfile(t, pool, target)

	if _, err := AddGrantByProfile(ctx, pool, notOwner, board.ID, profileID, "crew"); err != ErrNotAuthorized {
		t.Fatalf("expected ErrNotAuthorized for a non-owner granting by profile_id, got %v", err)
	}
}

package storyboards

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
)

func TestMyPeopleRelationshipAloneGrantsNothing(t *testing.T) {
	// Kernel 80 spec 1.8: a My People relationship does not automatically
	// grant Storyboard access -- there is no relationship-driven code path
	// in this package at all, so the only thing to assert is that a user
	// with zero storyboard_grants rows (regardless of any relationship
	// bookkeeping elsewhere) has no access.
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	friend := insertTestUser(t, pool, "sb_friend")
	board := mustCreateBoard(t, pool, owner, "Private Board")

	if ok, err := CanViewBoard(ctx, pool, friend, board); err != nil || ok {
		t.Fatalf("friend with no explicit grant should not have access: ok=%v err=%v", ok, err)
	}
}

func TestGrantByHandleAndRoleChange(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	member := insertTestUser(t, pool, "sb_member")
	board := mustCreateBoard(t, pool, owner, "Shared Board")

	var memberHandle string
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, member).Scan(&memberHandle)

	g, err := AddGrant(ctx, pool, owner, board.ID, memberHandle, "crew")
	if err != nil {
		t.Fatalf("add grant: %v", err)
	}
	if g.GrantedRole != "crew" {
		t.Fatalf("expected crew, got %s", g.GrantedRole)
	}

	tier, err := ResolveViewerTier(ctx, pool, member, board)
	if err != nil || tier != TierCrew {
		t.Fatalf("expected crew tier, got %s err=%v", tier, err)
	}

	// Re-grant changes role instead of erroring.
	g2, err := AddGrant(ctx, pool, owner, board.ID, memberHandle, "director")
	if err != nil {
		t.Fatalf("re-grant: %v", err)
	}
	if g2.GrantedRole != "director" {
		t.Fatalf("expected director after re-grant, got %s", g2.GrantedRole)
	}
	tier, _ = ResolveViewerTier(ctx, pool, member, board)
	if tier != TierDirector {
		t.Fatalf("expected director tier after re-grant, got %s", tier)
	}

	grants, err := ListGrants(ctx, pool, owner, board.ID)
	if err != nil || len(grants) != 1 {
		t.Fatalf("expected exactly one grant row after re-grant, got %d err=%v", len(grants), err)
	}
}

func TestRevokeGrantRemovesAccessImmediately(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	member := insertTestUser(t, pool, "sb_member")
	board := mustCreateBoard(t, pool, owner, "Revoke Board")
	var memberHandle string
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, member).Scan(&memberHandle)

	g, err := AddGrant(ctx, pool, owner, board.ID, memberHandle, "crew")
	if err != nil {
		t.Fatalf("add grant: %v", err)
	}
	if ok, _ := CanViewBoard(ctx, pool, member, board); !ok {
		t.Fatalf("expected access after grant")
	}

	if err := RemoveGrant(ctx, pool, owner, board.ID, g.ID); err != nil {
		t.Fatalf("remove grant: %v", err)
	}
	if ok, err := CanViewBoard(ctx, pool, member, board); err != nil || ok {
		t.Fatalf("expected no access after revoke: ok=%v err=%v", ok, err)
	}
}

func TestNonOwnerCannotManageGrants(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	director := insertTestUser(t, pool, "sb_director")
	target := insertTestUser(t, pool, "sb_target")
	board := mustCreateBoard(t, pool, owner, "Board")
	var directorHandle, targetHandle string
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, director).Scan(&directorHandle)
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, target).Scan(&targetHandle)

	if _, err := AddGrant(ctx, pool, owner, board.ID, directorHandle, "director"); err != nil {
		t.Fatalf("seed director grant: %v", err)
	}

	// A Director+-tier grantee still may not manage sharing -- only the
	// owner (or Operator) may (spec 5.4's "Default: only the owner manages
	// Storyboard access grants").
	if _, err := AddGrant(ctx, pool, director, board.ID, targetHandle, "cast"); err != ErrNotAuthorized {
		t.Fatalf("expected ErrNotAuthorized for director granting access, got %v", err)
	}
}

func TestGrantToNonexistentHandleErrors(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Board")

	if _, err := AddGrant(ctx, pool, owner, board.ID, "no-such-handle-xyz", "crew"); err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestInvalidGrantedRoleRejected(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	member := insertTestUser(t, pool, "sb_member")
	board := mustCreateBoard(t, pool, owner, "Board")
	var memberHandle string
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, member).Scan(&memberHandle)

	if _, err := AddGrant(ctx, pool, owner, board.ID, memberHandle, "owner"); err != ErrInvalidGrantedRole {
		t.Fatalf("expected ErrInvalidGrantedRole for 'owner', got %v", err)
	}
}

package participation_test

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/participation"
	"victory/backend/internal/showruns"
)

// TestRosterRoleDoesNotCrossShowRuns is the Kernel 97 spec §28 cross-Show
// isolation proof for roster-based authority (Cast/Player/Crew), which --
// unlike Producer/Director management authority, correctly location-scoped
// via CanManageShowRun -- is scoped per Show Run via show_run_roster_members.
// A Player rostered on Show Run A must not resolve as a participant on an
// unrelated Show Run B at the same Location.
func TestRosterRoleDoesNotCrossShowRuns(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	producer := bugClosureUser(t, pool, "xshow_producer")
	player := bugClosureUser(t, pool, "xshow_player")
	locationID := bugClosureLocationID(t, pool)

	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'producer', TRUE)
	`, locationID, producer); err != nil {
		t.Fatalf("grant producer location_memberships row: %v", err)
	}

	suffix := bugClosureSuffix(t)
	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, locationID, "Cross Show Production "+suffix, "cross-show-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID) })

	showA, err := showruns.CreateShowRun(ctx, pool, producer, productionID, showruns.CreateShowRunInput{
		Title: "Cross Show A " + suffix, Slug: "cross-show-a-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run A: %v", err)
	}
	showB, err := showruns.CreateShowRun(ctx, pool, producer, productionID, showruns.CreateShowRunInput{
		Title: "Cross Show B " + suffix, Slug: "cross-show-b-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run B: %v", err)
	}

	// Player is rostered on A only -- never touches B at all.
	if _, err := pool.Exec(ctx, `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id)
		VALUES ($1, $2, 'player', $3)
	`, showA.ID, player, producer); err != nil {
		t.Fatalf("insert player roster row on show A: %v", err)
	}

	resultA, err := participation.ResolveParticipationContext(ctx, pool, player, "catharsis", showA.ID)
	if err != nil {
		t.Fatalf("ResolveParticipationContext(A): %v", err)
	}
	if resultA.ViewerMode != participation.ViewerModePlayer {
		t.Fatalf("expected ViewerMode player on their own Show Run A, got %q", resultA.ViewerMode)
	}

	resultB, err := participation.ResolveParticipationContext(ctx, pool, player, "catharsis", showB.ID)
	if err != nil {
		t.Fatalf("ResolveParticipationContext(B): %v", err)
	}
	if resultB.ViewerMode == participation.ViewerModePlayer {
		t.Fatalf("Show A's roster Player must not resolve as player on unrelated Show B -- this is the cross-Show leak spec §28 forbids, got %+v", resultB)
	}
	if resultB.CanParticipate {
		t.Fatalf("expected CanParticipate false on unrelated Show B, got %+v", resultB)
	}
}

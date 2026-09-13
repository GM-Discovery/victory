package participation_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/participation"
	"victory/backend/internal/showruns"
)

func bugClosureSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000000")
}

func bugClosureUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
	t.Helper()
	ctx := context.Background()
	handle := handlePrefix + "_" + bugClosureSuffix(t)

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, handle, handlePrefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_run_roster_members WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_runs WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM access_grants WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func bugClosureLocationID(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var locationID string
	if err := pool.QueryRow(context.Background(), `
		SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1
	`).Scan(&locationID); err != nil {
		t.Fatalf("load amurray-family location: %v", err)
	}
	return locationID
}

// TestProducerWithOnlyLocationMembershipsIsNotNone is the exact Kernel 70A
// bug closure (spec §3.2): a real Producer whose only grant is a
// location_memberships row -- the normal shape for anyone bootstrapped or
// signed up after Kernel 66, since neither path writes to the legacy
// memberships table -- must resolve as producer, never "none", at their own
// venue.
func TestProducerWithOnlyLocationMembershipsIsNotNone(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	userID := bugClosureUser(t, pool, "bugclosure_producer")
	locationID := bugClosureLocationID(t, pool)

	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'producer', TRUE)
	`, locationID, userID); err != nil {
		t.Fatalf("grant location_memberships producer row: %v", err)
	}

	// Confirm the legacy memberships table genuinely has no row for this
	// user -- the exact shape that made the old memberships-only
	// lookupVenueRole return "none".
	var legacyCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM memberships WHERE user_id = $1`, userID).Scan(&legacyCount); err != nil {
		t.Fatalf("count legacy memberships rows: %v", err)
	}
	if legacyCount != 0 {
		t.Fatalf("test setup invalid: expected zero legacy memberships rows, got %d", legacyCount)
	}

	role, err := participation.LegacyLookupVenueRole(ctx, pool, userID, "catharsis")
	if err != nil {
		t.Fatalf("LegacyLookupVenueRole: %v", err)
	}
	if role != "producer" {
		t.Fatalf("expected viewer role %q, got %q (this is the Kernel 70A bug if it says \"none\")", "producer", role)
	}

	result, err := participation.ResolveParticipationContext(ctx, pool, userID, "catharsis", "")
	if err != nil {
		t.Fatalf("ResolveParticipationContext: %v", err)
	}
	if result.ViewerMode != participation.ViewerModeProducer {
		t.Fatalf("expected ViewerMode producer, got %q", result.ViewerMode)
	}
	if !result.CanManage || !result.CanViewBackstage || !result.CanEnterVenue {
		t.Fatalf("expected full producer authority, got %+v", result)
	}
}

// TestValidTicketRosterGrantsEntryWithNoManualAccessGrant is the second
// required bug-closure proof (spec §3.2): a user whose only relationship to
// the system is an active show_run_roster_members player row -- no
// location_memberships, no legacy memberships, no access_grants at all --
// must be able to enter their Show Run's theater and participate with zero
// additional manual access-grant step.
func TestValidTicketRosterGrantsEntryWithNoManualAccessGrant(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	producer := bugClosureUser(t, pool, "bugclosure_run_producer")
	player := bugClosureUser(t, pool, "bugclosure_player")
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
	`, locationID, "Bug Closure Production "+suffix, "bug-closure-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID) })

	sr, err := showruns.CreateShowRun(ctx, pool, producer, productionID, showruns.CreateShowRunInput{
		Title: "Bug Closure Show Run " + suffix, Slug: "bug-closure-show-run-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run: %v", err)
	}

	// The player row a valid ticket's second punch would create -- inserted
	// directly here since Phase B's ticket transaction doesn't exist yet;
	// this test proves the resolver side of the contract independent of
	// how the roster row came to exist.
	if _, err := pool.Exec(ctx, `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id)
		VALUES ($1, $2, 'player', $3)
	`, sr.ID, player, producer); err != nil {
		t.Fatalf("insert player roster row: %v", err)
	}

	for _, table := range []string{"location_memberships", "memberships", "access_grants"} {
		var count int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM `+table+` WHERE user_id = $1`, player).Scan(&count); err != nil {
			t.Fatalf("count %s rows for player: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("test setup invalid: expected zero %s rows for the player, got %d", table, count)
		}
	}

	result, err := participation.ResolveParticipationContext(ctx, pool, player, "catharsis", sr.ID)
	if err != nil {
		t.Fatalf("ResolveParticipationContext: %v", err)
	}
	if result.ViewerMode != participation.ViewerModePlayer {
		t.Fatalf("expected ViewerMode player, got %q", result.ViewerMode)
	}
	if !result.CanEnterVenue {
		t.Fatalf("expected CanEnterVenue true from roster membership alone, got %+v", result)
	}
	if !result.CanParticipate {
		t.Fatalf("expected CanParticipate true from roster membership alone, got %+v", result)
	}
}

// TestLegacyLookupVenueRoleResolvesRosterPlayerViaLiveSession is the Kernel
// 97 fix for the exact carried finding in spec Sec5: LegacyLookupVenueRole
// (the only live caller feeding /api/world/* snapshots for the-cave,
// catharsis, and first-theater) used to always pass showRunID="" to
// ResolveParticipationContext, which skips Step 3 -- the only step that
// resolves a real show_run_roster_members role -- entirely. A roster
// Player with no location_memberships row (the normal shape) fell through
// to the bare "audience" fallback despite genuinely being a Player. Proves
// the fix: LegacyLookupVenueRole now discovers the live session's Show Run
// at the venue itself, with no caller change required.
func TestLegacyLookupVenueRoleResolvesRosterPlayerViaLiveSession(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	producer := bugClosureUser(t, pool, "bugclosure_live_producer")
	player := bugClosureUser(t, pool, "bugclosure_live_player")
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
	`, locationID, "Bug Closure Live Production "+suffix, "bug-closure-live-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID) })

	sr, err := showruns.CreateShowRun(ctx, pool, producer, productionID, showruns.CreateShowRunInput{
		Title: "Bug Closure Live Show Run " + suffix, Slug: "bug-closure-live-show-run-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id)
		VALUES ($1, $2, 'player', $3)
	`, sr.ID, player, producer); err != nil {
		t.Fatalf("insert player roster row: %v", err)
	}

	var showID, sessionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, slug, title, status, created_by_user_id)
		VALUES ($1, $2, $3, 'live', $4)
		RETURNING id::text
	`, sr.ID, "bug-closure-live-show-"+suffix, "Bug Closure Live Show "+suffix, producer).Scan(&showID); err != nil {
		t.Fatalf("insert show: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM shows WHERE id = $1`, showID) })

	if err := pool.QueryRow(ctx, `
		INSERT INTO sessions (venue_id, status, show_id)
		SELECT id, 'live', $1 FROM venues WHERE slug = 'catharsis'
		RETURNING id::text
	`, showID).Scan(&sessionID); err != nil {
		t.Fatalf("insert live session at catharsis: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, sessionID) })

	for _, table := range []string{"location_memberships", "memberships", "access_grants"} {
		var count int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM `+table+` WHERE user_id = $1`, player).Scan(&count); err != nil {
			t.Fatalf("count %s rows for player: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("test setup invalid: expected zero %s rows for the player, got %d", table, count)
		}
	}

	role, err := participation.LegacyLookupVenueRole(ctx, pool, player, "catharsis")
	if err != nil {
		t.Fatalf("LegacyLookupVenueRole: %v", err)
	}
	if role != "player" {
		t.Fatalf("expected viewer role %q, got %q (this is the Kernel 97 live-session-hint bug if it says \"none\" or \"audience\")", "player", role)
	}
}

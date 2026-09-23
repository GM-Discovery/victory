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

// TestCastLocationMembershipResolvesAsCastNotAudience is the Kernel 101
// 101-19/101-20/101-22 real root-cause closure: Step 2 (location_memberships)
// used to only recognize "producer"/"director", so a Cast-approved account
// (the exact Kernel 101-17 authority, granted via Audition Hall) with no
// show_run_roster_members seat in the venue's currently-live Show Run fell
// through every canonical step to Step 4's blanket "any active
// location_memberships row at all -> audience" fallback. This is what made
// /api/world/catharsis's theater_context.kind resolve "audience" for a real
// Cast account even after 101-20 taught resolveTheaterContext to treat the
// string "cast" as backstage-tier -- the string never arrived as "cast" in
// the first place. Confirmed live via Grant's own account (role "cast" from
// /api/session/catharsis/join, but "audience" theater_context from
// /api/world/catharsis) before this fix.
func TestCastLocationMembershipResolvesAsCastNotAudience(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	producer := bugClosureUser(t, pool, "bugclosure_cast_producer")
	cast := bugClosureUser(t, pool, "bugclosure_cast_member")
	locationID := bugClosureLocationID(t, pool)

	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'producer', TRUE)
	`, locationID, producer); err != nil {
		t.Fatalf("grant producer location_memberships row: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'cast', TRUE)
	`, locationID, cast); err != nil {
		t.Fatalf("grant cast location_memberships row: %v", err)
	}

	suffix := bugClosureSuffix(t)
	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, locationID, "Bug Closure Cast Production "+suffix, "bug-closure-cast-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID) })

	sr, err := showruns.CreateShowRun(ctx, pool, producer, productionID, showruns.CreateShowRunInput{
		Title: "Bug Closure Cast Show Run " + suffix, Slug: "bug-closure-cast-show-run-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run: %v", err)
	}

	var showID, sessionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, slug, title, status, created_by_user_id)
		VALUES ($1, $2, $3, 'live', $4)
		RETURNING id::text
	`, sr.ID, "bug-closure-cast-live-show-"+suffix, "Bug Closure Cast Live Show "+suffix, producer).Scan(&showID); err != nil {
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

	// Confirm the exact real-world shape: cast has no roster seat at all in
	// this specific, currently-live Show Run.
	var rosterCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM show_run_roster_members WHERE show_run_id = $1 AND user_id = $2`, sr.ID, cast).Scan(&rosterCount); err != nil {
		t.Fatalf("count roster rows for cast: %v", err)
	}
	if rosterCount != 0 {
		t.Fatalf("test setup invalid: expected zero roster rows for cast in this show run, got %d", rosterCount)
	}

	role, err := participation.LegacyLookupVenueRole(ctx, pool, cast, "catharsis")
	if err != nil {
		t.Fatalf("LegacyLookupVenueRole: %v", err)
	}
	if role != "cast" {
		t.Fatalf("expected viewer role %q, got %q (this is the Kernel 101 bug if it says \"audience\")", "cast", role)
	}

	result, err := participation.ResolveParticipationContext(ctx, pool, cast, "catharsis", sr.ID)
	if err != nil {
		t.Fatalf("ResolveParticipationContext: %v", err)
	}
	if result.ViewerMode != participation.ViewerModeCast {
		t.Fatalf("expected ViewerMode cast, got %q", result.ViewerMode)
	}
	if !result.CanEnterVenue || !result.CanParticipate {
		t.Fatalf("expected CanEnterVenue and CanParticipate true for location Cast, got %+v", result)
	}
	if result.CanViewBackstage {
		t.Fatalf("expected CanViewBackstage false for a bare location Cast row with no roster seat -- hidden-object perception must stay gated on actual Show Run roster membership, got %+v", result)
	}
}

// TestLocationCrewNotDowngradedByShowRunPlayerRole guards the edge case the
// Cast fix above introduces: Step 2 now sets ViewerMode for location-level
// Crew too (previously only producer/director), so Step 3's roster check
// needed an explicit guard added alongside it -- without it, a location
// Crew member who also happens to hold a "player" roster row on one
// specific Show Run (e.g. crew doubling as an extra) would have been
// downgraded from Crew to Player, silently losing CanViewBackstage. Crew
// must always keep full backstage state per the product model
// (Docs/Product/Glossary.md), never gated on a specific show run's roster.
func TestLocationCrewNotDowngradedByShowRunPlayerRole(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	producer := bugClosureUser(t, pool, "bugclosure_crew_producer")
	crew := bugClosureUser(t, pool, "bugclosure_crew_member")
	locationID := bugClosureLocationID(t, pool)

	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'producer', TRUE)
	`, locationID, producer); err != nil {
		t.Fatalf("grant producer location_memberships row: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'crew', TRUE)
	`, locationID, crew); err != nil {
		t.Fatalf("grant crew location_memberships row: %v", err)
	}

	suffix := bugClosureSuffix(t)
	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, locationID, "Bug Closure Crew Production "+suffix, "bug-closure-crew-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID) })

	sr, err := showruns.CreateShowRun(ctx, pool, producer, productionID, showruns.CreateShowRunInput{
		Title: "Bug Closure Crew Show Run " + suffix, Slug: "bug-closure-crew-show-run-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id)
		VALUES ($1, $2, 'player', $3)
	`, sr.ID, crew, producer); err != nil {
		t.Fatalf("insert player roster row for the crew member: %v", err)
	}

	result, err := participation.ResolveParticipationContext(ctx, pool, crew, "catharsis", sr.ID)
	if err != nil {
		t.Fatalf("ResolveParticipationContext: %v", err)
	}
	if result.ViewerMode != participation.ViewerModeCrew {
		t.Fatalf("expected ViewerMode to stay crew despite the player roster row, got %q", result.ViewerMode)
	}
	if !result.CanViewBackstage {
		t.Fatalf("expected CanViewBackstage to remain true for location Crew, got %+v", result)
	}
}

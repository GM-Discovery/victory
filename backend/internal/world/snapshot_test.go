package world

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/showruns"
)

// This is the first test file for the world package. It proves the Kernel
// 70 persistent-Show-stage fold in LoadVenueSnapshot: (1) a session linked
// to a Show sees show_id-scoped actions recorded under a DIFFERENT,
// already-ended session -- proving no synthetic re-insertion is needed for
// a Show's state to resume -- and (2) a session with no linked Show
// produces exactly the same result as if the fold code didn't exist,
// proving backward compatibility for every pre-Kernel-70 session.

func openWorldTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.OpenTestPool(t)
}

func testSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
}

func insertWorldTestUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
	t.Helper()
	ctx := context.Background()
	handle := handlePrefix + "_" + testSuffix(t)

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, handle, handlePrefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM actions WHERE actor_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

// worldFixture builds an isolated location -> lot -> venue chain (a
// dedicated test venue, not one of the shared seeded venues, so concurrent
// or later test runs never see each other's "most recent active session"
// for the same venue slug) plus a library element that can be placed onto
// it, and a full location -> production -> show run -> show chain.
type worldFixture struct {
	locationID string
	venueID    string
	venueSlug  string
	elementID  string
	showID     string
}

func buildWorldFixture(t *testing.T, pool *pgxpool.Pool, actorUserID string) worldFixture {
	t.Helper()
	ctx := context.Background()
	suffix := testSuffix(t)

	var f worldFixture
	if err := pool.QueryRow(ctx, `
		INSERT INTO locations (name, slug) VALUES ($1, $2) RETURNING id::text
	`, "World Test Location "+suffix, "world-test-location-"+suffix).Scan(&f.locationID); err != nil {
		t.Fatalf("insert location: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, f.locationID) })

	var lotID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO lots (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, f.locationID, "World Test Lot "+suffix, "world-test-lot-"+suffix).Scan(&lotID); err != nil {
		t.Fatalf("insert lot: %v", err)
	}

	f.venueSlug = "world-test-venue-" + suffix
	if err := pool.QueryRow(ctx, `
		INSERT INTO venues (lot_id, name, slug, kind) VALUES ($1, $2, $3, 'presentation') RETURNING id::text
	`, lotID, "World Test Venue "+suffix, f.venueSlug).Scan(&f.venueID); err != nil {
		t.Fatalf("insert venue: %v", err)
	}

	var libraryID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO libraries (location_id, name) VALUES ($1, $2) RETURNING id::text
	`, f.locationID, "World Test Library "+suffix).Scan(&libraryID); err != nil {
		t.Fatalf("insert library: %v", err)
	}

	if err := pool.QueryRow(ctx, `
		INSERT INTO elements (library_id, name, slug, element_type, data)
		VALUES ($1, $2, $3, 'prop', '{}'::jsonb)
		RETURNING id::text
	`, libraryID, "World Test Prop "+suffix, "world-test-prop-"+suffix).Scan(&f.elementID); err != nil {
		t.Fatalf("insert element: %v", err)
	}

	grantLocationRole(t, pool, f.locationID, actorUserID, "producer")
	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, f.locationID, "World Test Production "+suffix, "world-test-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	sr, err := showruns.CreateShowRun(ctx, pool, actorUserID, productionID, showruns.CreateShowRunInput{
		Title: "World Test Show Run " + suffix, Slug: "world-test-show-run-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run fixture: %v", err)
	}
	// Inserted directly via SQL rather than calling shows.CreateShow: Kernel
	// 70A's shows.StartShowSession now imports network (to reuse
	// network.StartSessionControl's idempotent session-start core), and
	// network imports world -- so world's own test file importing shows
	// would create a real import cycle (world_test -> shows -> network ->
	// world). This mirrors the same small-duplicated-SQL-query pattern
	// Kernel 70 already used for an analogous cross-package test need.
	if err := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, slug, title, created_by_user_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, sr.ID, "world-test-show-"+suffix, "World Test Show "+suffix, actorUserID).Scan(&f.showID); err != nil {
		t.Fatalf("insert show fixture: %v", err)
	}

	return f
}

func grantLocationRole(t *testing.T, pool *pgxpool.Pool, locationID, userID, role string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, $3::location_role, TRUE)
		ON CONFLICT (location_id, user_id, role) DO UPDATE SET active = TRUE
	`, locationID, userID, role); err != nil {
		t.Fatalf("grant location role %q: %v", role, err)
	}
}

// insertSession creates a session row directly (bypassing
// network/session_control.go's command flow, which this package doesn't
// import) and optionally links it to a Show via sessions.show_id.
func insertSession(t *testing.T, pool *pgxpool.Pool, venueID, showID, status string) string {
	t.Helper()
	var sessionID string
	var showIDArg *string
	if showID != "" {
		showIDArg = &showID
	}
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO sessions (venue_id, status, show_id) VALUES ($1, $2::session_status, $3) RETURNING id::text
	`, venueID, status, showIDArg).Scan(&sessionID); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, sessionID) })
	return sessionID
}

// insertPlaceElementAction inserts an act/place_element action row scoped
// to the given session and (optionally) show, mirroring the shape
// upsertPlacedElementFromAction expects (target.element_id,
// target.venue_slug).
func insertPlaceElementAction(t *testing.T, pool *pgxpool.Pool, sessionID, showID, actorUserID, elementID, venueSlug string) {
	t.Helper()
	ctx := context.Background()

	target, err := json.Marshal(map[string]any{"element_id": elementID, "venue_slug": venueSlug})
	if err != nil {
		t.Fatalf("marshal target: %v", err)
	}
	payload, err := json.Marshal(map[string]any{"x": 10, "y": 20})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	var showIDArg *string
	if showID != "" {
		showIDArg = &showID
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO actions (session_id, show_id, moment_id, actor_id, type, target, payload)
		VALUES (
			$1, $2,
			COALESCE((SELECT MAX(moment_id) FROM actions WHERE session_id = $1), 0) + 1,
			$3, 'act/place_element', $4, $5
		)
	`, sessionID, showIDArg, actorUserID, target, payload); err != nil {
		t.Fatalf("insert act/place_element action: %v", err)
	}
}

// TestLoadVenueSnapshotFoldsPersistentShowActionsAcrossSessions is the
// core Kernel 70 SS4.3 proof: a Show's state recorded under an
// already-ended session must be visible to a brand-new session linked to
// the same Show, with zero synthetic re-insertion into the new session's
// own action log.
func TestLoadVenueSnapshotFoldsPersistentShowActionsAcrossSessions(t *testing.T) {
	pool := openWorldTestPool(t)
	producer := insertWorldTestUser(t, pool, "wd_producer")
	f := buildWorldFixture(t, pool, producer)

	// Session A: linked to the Show, records a show_id-scoped
	// act/place_element action, then "ends" (status closed).
	sessionA := insertSession(t, pool, f.venueID, f.showID, "live")
	insertPlaceElementAction(t, pool, sessionA, f.showID, producer, f.elementID, f.venueSlug)
	if _, err := pool.Exec(context.Background(), `UPDATE sessions SET status = 'closed', ended_at = NOW() WHERE id = $1`, sessionA); err != nil {
		t.Fatalf("close session A: %v", err)
	}

	// Session B: a brand-new session, also linked to the Show. Nothing is
	// ever written into session B's own action log by test code -- the
	// fold must surface session A's persisted show_id-scoped action
	// purely by querying show_id, not by any copy step.
	sessionB := insertSession(t, pool, f.venueID, f.showID, "live")

	var sessionBActionCountBefore int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM actions WHERE session_id = $1`, sessionB).Scan(&sessionBActionCountBefore); err != nil {
		t.Fatalf("count session B actions: %v", err)
	}
	if sessionBActionCountBefore != 0 {
		t.Fatalf("expected session B to start with zero of its own actions, got %d", sessionBActionCountBefore)
	}

	snap, err := LoadVenueSnapshot(context.Background(), pool, "producer", producer, f.venueSlug)
	if err != nil {
		t.Fatalf("load venue snapshot: %v", err)
	}

	found := false
	for _, el := range snap.Elements {
		if el.ElementID == f.elementID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected session B's snapshot to include the element placed via session A's show-scoped action, got elements=%+v", snap.Elements)
	}
	if snap.Session.ShowID != f.showID {
		t.Fatalf("expected snapshot session.show_id = %q, got %q", f.showID, snap.Session.ShowID)
	}

	var sessionBActionCountAfter int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM actions WHERE session_id = $1`, sessionB).Scan(&sessionBActionCountAfter); err != nil {
		t.Fatalf("re-count session B actions: %v", err)
	}
	if sessionBActionCountAfter != 0 {
		t.Fatalf("expected session B to still have zero of its own actions after loading the snapshot (no synthetic re-insertion), got %d", sessionBActionCountAfter)
	}
}

// TestLoadVenueSnapshotUnaffectedForSessionWithNoLinkedShow is the
// explicit backward-compatibility proof: a session with show_id IS NULL
// must never see another Show's (or anyone else's) show_id-scoped
// actions, and behaves exactly as if the Kernel 70 fold code didn't exist.
func TestLoadVenueSnapshotUnaffectedForSessionWithNoLinkedShow(t *testing.T) {
	pool := openWorldTestPool(t)
	producer := insertWorldTestUser(t, pool, "wd_unlinked_producer")
	f := buildWorldFixture(t, pool, producer)

	// An unrelated show_id-scoped action exists (as if some other Show's
	// Cue fired) -- it must never leak into a session that isn't linked
	// to that Show.
	otherSession := insertSession(t, pool, f.venueID, f.showID, "closed")
	insertPlaceElementAction(t, pool, otherSession, f.showID, producer, f.elementID, f.venueSlug)

	// This session is NOT linked to any Show.
	unlinkedSession := insertSession(t, pool, f.venueID, "", "live")
	_ = unlinkedSession

	snap, err := LoadVenueSnapshot(context.Background(), pool, "producer", producer, f.venueSlug)
	if err != nil {
		t.Fatalf("load venue snapshot: %v", err)
	}

	for _, el := range snap.Elements {
		if el.ElementID == f.elementID {
			t.Fatalf("expected a session with no linked Show to never see another Show's show_id-scoped action, but found it: %+v", el)
		}
	}
	if snap.Session.ShowID != "" {
		t.Fatalf("expected an unlinked session's snapshot to have an empty show_id, got %q", snap.Session.ShowID)
	}
}

// TestLoadVenueSnapshotStillFoldsOwnSessionActionsWhenLinkedToShow proves
// the fold is additive, not a replacement: a session linked to a Show
// still sees its own session_id-scoped actions exactly as before, on top
// of any persistent show_id-scoped state.
func TestLoadVenueSnapshotStillFoldsOwnSessionActionsWhenLinkedToShow(t *testing.T) {
	pool := openWorldTestPool(t)
	producer := insertWorldTestUser(t, pool, "wd_own_session_producer")
	f := buildWorldFixture(t, pool, producer)

	session := insertSession(t, pool, f.venueID, f.showID, "live")
	// Session-scoped only (no show_id) -- the pre-Kernel-70 path.
	insertPlaceElementAction(t, pool, session, "", producer, f.elementID, f.venueSlug)

	snap, err := LoadVenueSnapshot(context.Background(), pool, "producer", producer, f.venueSlug)
	if err != nil {
		t.Fatalf("load venue snapshot: %v", err)
	}
	found := false
	for _, el := range snap.Elements {
		if el.ElementID == f.elementID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the session's own session_id-scoped action to still be folded in, got elements=%+v", snap.Elements)
	}
}

// TestLoadVenueSnapshotReturnsGracefulVenueOpenForVenueWithNoActiveSession
// is the Kernel 70A fix proof: a venue with zero active (rehearsal/live)
// sessions must return a normal, empty snapshot with theater_context
// "venue_open" for a non-backstage viewer -- not a Scan-type crash (the
// prior bug: s.id/s.status/s.started_at came from a LEFT JOIN and were
// scanned into non-nullable destinations, so this exact case -- a real,
// valid venue with nothing currently on stage -- errored with a pgx
// type-conversion error) and not a hard error either (this is a normal,
// expected "idle theater" state a viewer should see a friendly empty state
// for, not an error page).
func TestLoadVenueSnapshotReturnsGracefulVenueOpenForVenueWithNoActiveSession(t *testing.T) {
	pool := openWorldTestPool(t)
	producer := insertWorldTestUser(t, pool, "wd_no_session_producer")
	f := buildWorldFixture(t, pool, producer)
	// buildWorldFixture creates the venue but no session at all -- exactly
	// the "idle theater" case.

	audienceViewer := insertWorldTestUser(t, pool, "wd_no_session_audience")

	snap, err := LoadVenueSnapshot(context.Background(), pool, "audience", audienceViewer, f.venueSlug)
	if err != nil {
		t.Fatalf("expected a graceful snapshot for a venue with no active session, got error: %v", err)
	}
	if snap.TheaterContext.Kind != "venue_open" {
		t.Fatalf("expected theater_context.kind = venue_open, got %+v", snap.TheaterContext)
	}
	if snap.TheaterContext.Message != "No Show is currently on stage here." {
		t.Fatalf("expected the exact required empty-state message, got %q", snap.TheaterContext.Message)
	}

	// The same idle venue, viewed by backstage-tier staff, gets "backstage"
	// instead -- same idle state, different audience.
	backstageSnap, err := LoadVenueSnapshot(context.Background(), pool, "producer", producer, f.venueSlug)
	if err != nil {
		t.Fatalf("load venue snapshot for backstage viewer: %v", err)
	}
	if backstageSnap.TheaterContext.Kind != "backstage" {
		t.Fatalf("expected theater_context.kind = backstage for a producer, got %+v", backstageSnap.TheaterContext)
	}
}

// insertRosterMember inserts a show_run_roster_members row directly (not
// via showruns.AddRosterMember, which requires a player_profile_workbooks
// row this fixture has no other need for) with the given role.
func insertRosterMember(t *testing.T, pool *pgxpool.Pool, showRunID, userID, role, addedByUserID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id)
		VALUES ($1, $2, $3, $4)
	`, showRunID, userID, role, addedByUserID); err != nil {
		t.Fatalf("insert roster member role %q: %v", role, err)
	}
}

// selectRosterCharacter inserts a minimal character_cards row and selects
// it on the owner's existing show_run_roster_members row -- Kernel 71's
// Show-Run-scoped replacement for the old Session-scoped
// current_session_personas equip, which resolveTheaterContext no longer
// reads.
func selectRosterCharacter(t *testing.T, pool *pgxpool.Pool, locationID, showRunID, ownerUserID string) {
	t.Helper()
	ctx := context.Background()
	var cardID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO character_cards (owner_user_id, location_id, name)
		VALUES ($1, $2, $3)
		RETURNING id::text
	`, ownerUserID, locationID, "Theater Context Test Character").Scan(&cardID); err != nil {
		t.Fatalf("insert character card: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM character_cards WHERE id = $1`, cardID) })
	if _, err := pool.Exec(ctx, `
		UPDATE show_run_roster_members SET character_card_id = $1
		WHERE show_run_id = $2 AND user_id = $3 AND removed_at IS NULL
	`, cardID, showRunID, ownerUserID); err != nil {
		t.Fatalf("select roster character: %v", err)
	}
}

// TestLoadVenueSnapshotTheaterContextParticipantAndAudienceMessages proves
// the two remaining Kernel 70A theater_context branches and their exact
// required message strings: a registered Show Run player with no equipped
// Character yet, the same player once they've equipped one, and a plain
// Audience viewer -- against the same active session/Show so the only
// variable is the viewer.
func TestLoadVenueSnapshotTheaterContextParticipantAndAudienceMessages(t *testing.T) {
	pool := openWorldTestPool(t)
	producer := insertWorldTestUser(t, pool, "wd_tc_producer")
	f := buildWorldFixture(t, pool, producer)

	var showRunID string
	if err := pool.QueryRow(context.Background(), `SELECT show_run_id::text FROM shows WHERE id = $1`, f.showID).Scan(&showRunID); err != nil {
		t.Fatalf("load show_run_id: %v", err)
	}

	insertSession(t, pool, f.venueID, f.showID, "live")

	player := insertWorldTestUser(t, pool, "wd_tc_player")
	insertRosterMember(t, pool, showRunID, player, "player", producer)

	// Registered player, no Character equipped yet.
	playerSnap, err := LoadVenueSnapshot(context.Background(), pool, "audience", player, f.venueSlug)
	if err != nil {
		t.Fatalf("load snapshot for unequipped player: %v", err)
	}
	if playerSnap.TheaterContext.Kind != "participant" {
		t.Fatalf("expected kind = participant for a registered player, got %+v", playerSnap.TheaterContext)
	}
	if playerSnap.TheaterContext.Message != "You are registered for this Show, but you have not chosen a Character yet." {
		t.Fatalf("expected the exact required no-character message, got %q", playerSnap.TheaterContext.Message)
	}

	// Same player, now with a Character selected for this Show Run.
	selectRosterCharacter(t, pool, f.locationID, showRunID, player)
	equippedSnap, err := LoadVenueSnapshot(context.Background(), pool, "audience", player, f.venueSlug)
	if err != nil {
		t.Fatalf("load snapshot for player with a selected character: %v", err)
	}
	if equippedSnap.TheaterContext.Kind != "participant" {
		t.Fatalf("expected kind = participant once a character is selected, got %+v", equippedSnap.TheaterContext)
	}
	if equippedSnap.TheaterContext.Message != "" {
		t.Fatalf("expected no message once a Character is selected, got %q", equippedSnap.TheaterContext.Message)
	}
	if equippedSnap.TheaterContext.SelectedCharacterID == "" {
		t.Fatalf("expected theater_context.selected_character_id to be set once a character is selected")
	}

	// A plain Audience viewer, not on the roster at all.
	audienceViewer := insertWorldTestUser(t, pool, "wd_tc_audience")
	audienceSnap, err := LoadVenueSnapshot(context.Background(), pool, "audience", audienceViewer, f.venueSlug)
	if err != nil {
		t.Fatalf("load snapshot for audience viewer: %v", err)
	}
	if audienceSnap.TheaterContext.Kind != "audience" {
		t.Fatalf("expected kind = audience for an unregistered viewer, got %+v", audienceSnap.TheaterContext)
	}
	if audienceSnap.TheaterContext.Message != "You are watching this Show. Player controls are not active." {
		t.Fatalf("expected the exact required watching message, got %q", audienceSnap.TheaterContext.Message)
	}
}

// TestLoadVenueSnapshotTheaterContextTreatsArchivedCharacterAsUnselected is
// the Kernel 71 §7.2 fallback proof: a Player whose selected Character has
// since been archived (is_deleted) must be treated exactly like a Player
// who never selected one -- prompted again, not left pointing at a
// vanished Character.
func TestLoadVenueSnapshotTheaterContextTreatsArchivedCharacterAsUnselected(t *testing.T) {
	pool := openWorldTestPool(t)
	producer := insertWorldTestUser(t, pool, "wd_tc_archived_producer")
	f := buildWorldFixture(t, pool, producer)

	var showRunID string
	if err := pool.QueryRow(context.Background(), `SELECT show_run_id::text FROM shows WHERE id = $1`, f.showID).Scan(&showRunID); err != nil {
		t.Fatalf("load show_run_id: %v", err)
	}
	insertSession(t, pool, f.venueID, f.showID, "live")

	player := insertWorldTestUser(t, pool, "wd_tc_archived_player")
	insertRosterMember(t, pool, showRunID, player, "player", producer)

	var cardID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO character_cards (owner_user_id, location_id, name) VALUES ($1, $2, $3) RETURNING id::text
	`, player, f.locationID, "Archived Test Character").Scan(&cardID); err != nil {
		t.Fatalf("insert character card: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM character_cards WHERE id = $1`, cardID) })
	if _, err := pool.Exec(context.Background(), `
		UPDATE show_run_roster_members SET character_card_id = $1
		WHERE show_run_id = $2 AND user_id = $3 AND removed_at IS NULL
	`, cardID, showRunID, player); err != nil {
		t.Fatalf("select roster character: %v", err)
	}

	if _, err := pool.Exec(context.Background(), `UPDATE character_cards SET is_deleted = TRUE WHERE id = $1`, cardID); err != nil {
		t.Fatalf("archive character card: %v", err)
	}

	snap, err := LoadVenueSnapshot(context.Background(), pool, "audience", player, f.venueSlug)
	if err != nil {
		t.Fatalf("load snapshot after character archived: %v", err)
	}
	if snap.TheaterContext.Kind != "participant" {
		t.Fatalf("expected kind = participant, got %+v", snap.TheaterContext)
	}
	if snap.TheaterContext.Message != "You are registered for this Show, but you have not chosen a Character yet." {
		t.Fatalf("expected the archived character to be treated as unselected, got %q", snap.TheaterContext.Message)
	}
	if snap.TheaterContext.SelectedCharacterID != "" {
		t.Fatalf("expected no selected_character_id once the character is archived, got %q", snap.TheaterContext.SelectedCharacterID)
	}
}

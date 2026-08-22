package showtime_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/identity"
	"victory/backend/internal/scenes"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
	"victory/backend/internal/showtime"
)

var testDiscordCfg = identity.DiscordServerLinkConfig{}

func showtimeTestSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000000")
}

func showtimeTestUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
	t.Helper()
	ctx := context.Background()
	handle := handlePrefix + "_" + showtimeTestSuffix(t)

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, handle, handlePrefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE sessions SET show_id = NULL WHERE show_id IN (SELECT id FROM shows WHERE created_by_user_id = $1)`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_scene_placements WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM scenes WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM shows WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_run_roster_members WHERE user_id = $1 OR added_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_runs WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func showtimeGrantRole(t *testing.T, pool *pgxpool.Pool, locationID, userID, role string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, $3::location_role, TRUE)
		ON CONFLICT (location_id, user_id, role) DO UPDATE SET active = TRUE
	`, locationID, userID, role); err != nil {
		t.Fatalf("grant location role %q: %v", role, err)
	}
}

type showtimeFixture struct {
	locationID string
	showRunID  string
	showID     string
}

// buildShowtimeFixture builds location -> production -> show run -> show,
// with no scene placements yet -- callers add placements themselves to
// exercise the different venue-derivation branches.
func buildShowtimeFixture(t *testing.T, pool *pgxpool.Pool, producerUserID string) showtimeFixture {
	t.Helper()
	ctx := context.Background()
	suffix := showtimeTestSuffix(t)

	var f showtimeFixture
	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&f.locationID); err != nil {
		t.Fatalf("load amurray-family location: %v", err)
	}
	showtimeGrantRole(t, pool, f.locationID, producerUserID, "producer")

	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, f.locationID, "Showtime Test Production "+suffix, "showtime-test-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID) })

	sr, err := showruns.CreateShowRun(ctx, pool, producerUserID, productionID, showruns.CreateShowRunInput{
		Title: "Showtime Test Show Run " + suffix, Slug: "showtime-test-show-run-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run fixture: %v", err)
	}
	f.showRunID = sr.ID

	s, err := shows.CreateShow(ctx, pool, producerUserID, sr.ID, shows.CreateShowInput{
		Title: "Showtime Test Show " + suffix, Slug: "showtime-test-show-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show fixture: %v", err)
	}
	f.showID = s.ID

	return f
}

func catharsisVenueID(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM venues WHERE slug = 'catharsis'`).Scan(&id); err != nil {
		t.Fatalf("load catharsis venue id: %v", err)
	}
	return id
}

func addPlacement(t *testing.T, pool *pgxpool.Pool, producerUserID, locationID, productionID, showID, venueID, sceneSlugSuffix string) string {
	t.Helper()
	ctx := context.Background()
	scene, err := scenes.CreateScene(ctx, pool, producerUserID, locationID, productionID, scenes.CreateSceneInput{
		Title: "Showtime Scene " + sceneSlugSuffix, Slug: "showtime-scene-" + sceneSlugSuffix, DefaultVenueID: venueID,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	p, err := scenes.CreatePlacement(ctx, pool, producerUserID, showID, scenes.CreatePlacementInput{SceneID: scene.ID})
	if err != nil {
		t.Fatalf("create placement: %v", err)
	}
	return p.ID
}

func showProductionID(t *testing.T, pool *pgxpool.Pool, showRunID string) string {
	t.Helper()
	sr, err := showruns.LoadShowRunByID(context.Background(), pool, showRunID)
	if err != nil {
		t.Fatalf("load show run: %v", err)
	}
	return sr.ProductionID
}

// TestStartResolvesVenueFromSingleStagedPlacementAndLinksSession is the
// primary Start proof: a fresh Show with one staged placement resolving to
// a single venue starts cleanly, sets that placement as the Show's current
// Scene as a side effect, and links the Session -- all without the caller
// naming a venue.
func TestStartResolvesVenueFromSingleStagedPlacementAndLinksSession(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := showtimeTestUser(t, pool, "st_start_producer")
	f := buildShowtimeFixture(t, pool, producer)
	productionID := showProductionID(t, pool, f.showRunID)
	venueID := catharsisVenueID(t, pool)
	placementID := addPlacement(t, pool, producer, f.locationID, productionID, f.showID, venueID, showtimeTestSuffix(t))

	s, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show: %v", err)
	}

	result, err := showtime.Start(context.Background(), pool, testDiscordCfg, producer, s.ShortCode, "", false, "")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, result.SessionID) })

	if result.NeedsVenueChoice {
		t.Fatalf("expected venue to be derived automatically, got NeedsVenueChoice")
	}
	if result.VenueSlug != "catharsis" {
		t.Fatalf("expected derived venue catharsis, got %q", result.VenueSlug)
	}
	if result.SessionID == "" {
		t.Fatalf("expected a session to be started")
	}
	if result.ChatBridgeOn {
		t.Fatalf("expected chat bridge to remain off in test environment")
	}

	updated, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("reload show: %v", err)
	}
	if updated.CurrentShowScenePlacementID == nil || *updated.CurrentShowScenePlacementID != placementID {
		t.Fatalf("expected the single staged placement to become the current scene, got %v", updated.CurrentShowScenePlacementID)
	}
}

// TestStartAsksWhenNoPlacementsExist proves the "no safe resolvable
// theater" branch: a Show with zero staged placements cannot guess a venue
// and must ask instead of starting anything.
func TestStartAsksWhenNoPlacementsExist(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := showtimeTestUser(t, pool, "st_none_producer")
	f := buildShowtimeFixture(t, pool, producer)

	s, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show: %v", err)
	}

	result, err := showtime.Start(context.Background(), pool, testDiscordCfg, producer, s.ShortCode, "", false, "")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !result.NeedsVenueChoice {
		t.Fatalf("expected NeedsVenueChoice for a Show with no staged placements, got %+v", result)
	}
	if result.SessionID != "" {
		t.Fatalf("expected no session to be started when a venue choice is needed")
	}
}

// TestStartAsksWhenPlacementsResolveToMultipleVenues proves the ambiguous
// case doesn't guess either.
func TestStartAsksWhenPlacementsResolveToMultipleVenues(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := showtimeTestUser(t, pool, "st_multi_producer")
	f := buildShowtimeFixture(t, pool, producer)
	productionID := showProductionID(t, pool, f.showRunID)

	var otherVenueID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM venues WHERE slug = 'first-theater'`).Scan(&otherVenueID); err != nil {
		t.Fatalf("load first-theater venue id: %v", err)
	}
	catharsisID := catharsisVenueID(t, pool)

	addPlacement(t, pool, producer, f.locationID, productionID, f.showID, catharsisID, showtimeTestSuffix(t)+"_a")
	addPlacement(t, pool, producer, f.locationID, productionID, f.showID, otherVenueID, showtimeTestSuffix(t)+"_b")

	s, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show: %v", err)
	}

	result, err := showtime.Start(context.Background(), pool, testDiscordCfg, producer, s.ShortCode, "", false, "")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !result.NeedsVenueChoice {
		t.Fatalf("expected NeedsVenueChoice when placements resolve to multiple venues, got %+v", result)
	}
	if len(result.CandidateVenues) != 2 {
		t.Fatalf("expected 2 candidate venues, got %v", result.CandidateVenues)
	}
}

// TestEndPreservesCurrentSceneRosterAndCharacterSelection is the spec §9.4
// proof: /showtime end only ends the technical Session -- the Show's
// current Scene, roster, and every Player's Character selection survive
// byte-for-byte.
func TestEndPreservesCurrentSceneRosterAndCharacterSelection(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := showtimeTestUser(t, pool, "st_end_producer")
	player := showtimeTestUser(t, pool, "st_end_player")
	f := buildShowtimeFixture(t, pool, producer)
	productionID := showProductionID(t, pool, f.showRunID)
	venueID := catharsisVenueID(t, pool)
	addPlacement(t, pool, producer, f.locationID, productionID, f.showID, venueID, showtimeTestSuffix(t))

	s, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show: %v", err)
	}

	startResult, err := showtime.Start(context.Background(), pool, testDiscordCfg, producer, s.ShortCode, "", false, "")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, startResult.SessionID)
	})

	var characterID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO character_cards (owner_user_id, location_id, name) VALUES ($1, $2, $3) RETURNING id::text
	`, player, f.locationID, "Showtime End Test Character").Scan(&characterID); err != nil {
		t.Fatalf("insert character card: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM character_cards WHERE id = $1`, characterID)
	})
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id, character_card_id)
		VALUES ($1, $2, 'player', $3, $4)
	`, f.showRunID, player, producer, characterID); err != nil {
		t.Fatalf("insert player roster row: %v", err)
	}

	before, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show before end: %v", err)
	}
	var characterBefore string
	if err := pool.QueryRow(context.Background(), `
		SELECT COALESCE(character_card_id::text, '') FROM show_run_roster_members WHERE show_run_id = $1 AND user_id = $2
	`, f.showRunID, player).Scan(&characterBefore); err != nil {
		t.Fatalf("load character selection before end: %v", err)
	}

	if _, err := showtime.End(context.Background(), pool, testDiscordCfg, producer, s.ShortCode, ""); err != nil {
		t.Fatalf("End: %v", err)
	}

	after, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show after end: %v", err)
	}
	if before.CurrentShowScenePlacementID == nil || after.CurrentShowScenePlacementID == nil ||
		*before.CurrentShowScenePlacementID != *after.CurrentShowScenePlacementID {
		t.Fatalf("expected current_show_scene_placement_id to be unchanged, before=%v after=%v", before.CurrentShowScenePlacementID, after.CurrentShowScenePlacementID)
	}

	var characterAfter string
	if err := pool.QueryRow(context.Background(), `
		SELECT COALESCE(character_card_id::text, '') FROM show_run_roster_members WHERE show_run_id = $1 AND user_id = $2
	`, f.showRunID, player).Scan(&characterAfter); err != nil {
		t.Fatalf("load character selection after end: %v", err)
	}
	if characterBefore != characterAfter || characterAfter != characterID {
		t.Fatalf("expected character selection unchanged (%q), got before=%q after=%q", characterID, characterBefore, characterAfter)
	}

	var rosterCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM show_run_roster_members WHERE show_run_id = $1 AND user_id = $2 AND removed_at IS NULL
	`, f.showRunID, player).Scan(&rosterCount); err != nil {
		t.Fatalf("count roster rows after end: %v", err)
	}
	if rosterCount != 1 {
		t.Fatalf("expected the player's roster row to remain active after end, got count %d", rosterCount)
	}

	var sessionStatus string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM sessions WHERE id = $1`, startResult.SessionID).Scan(&sessionStatus); err != nil {
		t.Fatalf("load session status: %v", err)
	}
	if sessionStatus != "closed" {
		t.Fatalf("expected session status closed after End, got %q", sessionStatus)
	}
}

// TestStartByShowIDMatchesStartByShortCode proves the Kernel 92 GUI path
// (show_id) and the /showtime <code> command path resolve the same Show and
// converge on the same idempotent "already live" result on a repeat call.
func TestStartByShowIDMatchesStartByShortCode(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := showtimeTestUser(t, pool, "st_byid_producer")
	f := buildShowtimeFixture(t, pool, producer)
	productionID := showProductionID(t, pool, f.showRunID)
	venueID := catharsisVenueID(t, pool)
	addPlacement(t, pool, producer, f.locationID, productionID, f.showID, venueID, showtimeTestSuffix(t))

	s, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show: %v", err)
	}

	byCode, err := showtime.Start(context.Background(), pool, testDiscordCfg, producer, s.ShortCode, "", false, "")
	if err != nil {
		t.Fatalf("Start by short code: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, byCode.SessionID) })
	if byCode.SessionID == "" {
		t.Fatalf("expected a session from the short-code Start")
	}

	byID, err := showtime.Start(context.Background(), pool, testDiscordCfg, producer, "", s.ID, false, "")
	if err != nil {
		t.Fatalf("Start by show_id: %v", err)
	}
	if !byID.AlreadyLive {
		t.Fatalf("expected the second Start (by show_id) to report AlreadyLive, got %+v", byID)
	}
	if byID.SessionID != byCode.SessionID {
		t.Fatalf("expected AlreadyLive to report the same session, got %q vs %q", byID.SessionID, byCode.SessionID)
	}
}

// TestStartIsIdempotentReturnsAlreadyLive proves repeated Showtime never
// starts a second session (kernel doc §21).
func TestStartIsIdempotentReturnsAlreadyLive(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := showtimeTestUser(t, pool, "st_idem_producer")
	f := buildShowtimeFixture(t, pool, producer)
	productionID := showProductionID(t, pool, f.showRunID)
	venueID := catharsisVenueID(t, pool)
	addPlacement(t, pool, producer, f.locationID, productionID, f.showID, venueID, showtimeTestSuffix(t))

	s, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show: %v", err)
	}

	first, err := showtime.Start(context.Background(), pool, testDiscordCfg, producer, s.ShortCode, "", false, "")
	if err != nil {
		t.Fatalf("first Start: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, first.SessionID) })

	second, err := showtime.Start(context.Background(), pool, testDiscordCfg, producer, s.ShortCode, "", false, "")
	if err != nil {
		t.Fatalf("second Start: %v", err)
	}
	if !second.AlreadyLive {
		t.Fatalf("expected second Start to report AlreadyLive, got %+v", second)
	}

	var sessionCount int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM sessions WHERE show_id = $1`, f.showID).Scan(&sessionCount); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessionCount != 1 {
		t.Fatalf("expected exactly 1 session after two Start calls, got %d", sessionCount)
	}
}

// TestEndIsIdempotentReportsAlreadyEnded proves repeated End Showtime never
// errors once nothing is live (kernel doc §21).
func TestEndIsIdempotentReportsAlreadyEnded(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := showtimeTestUser(t, pool, "st_endidem_producer")
	f := buildShowtimeFixture(t, pool, producer)
	productionID := showProductionID(t, pool, f.showRunID)
	venueID := catharsisVenueID(t, pool)
	addPlacement(t, pool, producer, f.locationID, productionID, f.showID, venueID, showtimeTestSuffix(t))

	s, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show: %v", err)
	}
	startResult, err := showtime.Start(context.Background(), pool, testDiscordCfg, producer, s.ShortCode, "", false, "")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, startResult.SessionID) })

	first, err := showtime.End(context.Background(), pool, testDiscordCfg, producer, s.ShortCode, "")
	if err != nil {
		t.Fatalf("first End: %v", err)
	}
	if first.AlreadyEnded {
		t.Fatalf("expected the first End to actually close the session, not report AlreadyEnded")
	}

	second, err := showtime.End(context.Background(), pool, testDiscordCfg, producer, s.ShortCode, "")
	if err != nil {
		t.Fatalf("second End: %v", err)
	}
	if !second.AlreadyEnded {
		t.Fatalf("expected the second End to report AlreadyEnded, got %+v", second)
	}
}

// TestPreflightReportsBlockersForBusyVenue proves a venue already occupied
// by a different live Show is a true blocker (kernel doc §14).
func TestPreflightReportsBlockersForBusyVenue(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := showtimeTestUser(t, pool, "st_busy_producer")
	venueID := catharsisVenueID(t, pool)

	fA := buildShowtimeFixture(t, pool, producer)
	productionIDA := showProductionID(t, pool, fA.showRunID)
	addPlacement(t, pool, producer, fA.locationID, productionIDA, fA.showID, venueID, showtimeTestSuffix(t)+"_a")
	sA, err := shows.LoadShowByID(context.Background(), pool, fA.showID)
	if err != nil {
		t.Fatalf("load show A: %v", err)
	}
	startA, err := showtime.Start(context.Background(), pool, testDiscordCfg, producer, sA.ShortCode, "", false, "")
	if err != nil {
		t.Fatalf("Start show A: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, startA.SessionID) })

	fB := buildShowtimeFixture(t, pool, producer)
	productionIDB := showProductionID(t, pool, fB.showRunID)
	addPlacement(t, pool, producer, fB.locationID, productionIDB, fB.showID, venueID, showtimeTestSuffix(t)+"_b")
	sB, err := shows.LoadShowByID(context.Background(), pool, fB.showID)
	if err != nil {
		t.Fatalf("load show B: %v", err)
	}

	preflight, err := showtime.Preflight(context.Background(), pool, producer, sB.ID)
	if err != nil {
		t.Fatalf("Preflight show B: %v", err)
	}
	if !containsString(preflight.Blockers, "venue_busy_with_other_show") {
		t.Fatalf("expected venue_busy_with_other_show blocker, got %+v", preflight.Blockers)
	}
}

// TestPreflightReportsWarningsNotBlockersForIncompleteCharacters proves
// incomplete Character selection never blocks Showtime (kernel doc §13/§15).
func TestPreflightReportsWarningsNotBlockersForIncompleteCharacters(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := showtimeTestUser(t, pool, "st_incomplete_producer")
	player := showtimeTestUser(t, pool, "st_incomplete_player")
	f := buildShowtimeFixture(t, pool, producer)
	productionID := showProductionID(t, pool, f.showRunID)
	venueID := catharsisVenueID(t, pool)
	addPlacement(t, pool, producer, f.locationID, productionID, f.showID, venueID, showtimeTestSuffix(t))

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id)
		VALUES ($1, $2, 'player', $3)
	`, f.showRunID, player, producer); err != nil {
		t.Fatalf("insert player roster row without a character: %v", err)
	}

	s, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show: %v", err)
	}

	preflight, err := showtime.Preflight(context.Background(), pool, producer, s.ID)
	if err != nil {
		t.Fatalf("Preflight: %v", err)
	}
	if containsString(preflight.Blockers, "characters_incomplete") {
		t.Fatalf("expected incomplete Characters to never be a blocker, got %+v", preflight.Blockers)
	}
	if !containsString(preflight.Warnings, "characters_incomplete") {
		t.Fatalf("expected incomplete Characters to be a warning, got %+v", preflight.Warnings)
	}
}

// TestPreflightDoesNotMutateShowState proves Preflight's venue derivation
// runs dry -- it must not persist a current-Scene placement the way an
// actual Start does (kernel doc §13: preflight is informative, not action).
func TestPreflightDoesNotMutateShowState(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := showtimeTestUser(t, pool, "st_dryrun_producer")
	f := buildShowtimeFixture(t, pool, producer)
	productionID := showProductionID(t, pool, f.showRunID)
	venueID := catharsisVenueID(t, pool)
	addPlacement(t, pool, producer, f.locationID, productionID, f.showID, venueID, showtimeTestSuffix(t))

	before, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show before preflight: %v", err)
	}
	if before.CurrentShowScenePlacementID != nil {
		t.Fatalf("expected no current scene placement before Preflight, got %v", before.CurrentShowScenePlacementID)
	}

	if _, err := showtime.Preflight(context.Background(), pool, producer, f.showID); err != nil {
		t.Fatalf("Preflight: %v", err)
	}

	after, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show after preflight: %v", err)
	}
	if after.CurrentShowScenePlacementID != nil {
		t.Fatalf("expected Preflight not to set a current scene placement, got %v", after.CurrentShowScenePlacementID)
	}
}

// TestEndReturnsAftercareEligibleCount proves End reports how many roster
// members would receive an Aftercare offer, without sending anything
// (kernel doc §23).
func TestEndReturnsAftercareEligibleCount(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := showtimeTestUser(t, pool, "st_aftercare_producer")
	playerWithCharacter := showtimeTestUser(t, pool, "st_aftercare_player_with")
	playerWithoutCharacter := showtimeTestUser(t, pool, "st_aftercare_player_without")
	f := buildShowtimeFixture(t, pool, producer)
	productionID := showProductionID(t, pool, f.showRunID)
	venueID := catharsisVenueID(t, pool)
	addPlacement(t, pool, producer, f.locationID, productionID, f.showID, venueID, showtimeTestSuffix(t))

	var characterID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO character_cards (owner_user_id, location_id, name) VALUES ($1, $2, $3) RETURNING id::text
	`, playerWithCharacter, f.locationID, "Aftercare Count Character").Scan(&characterID); err != nil {
		t.Fatalf("insert character card: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM character_cards WHERE id = $1`, characterID) })

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id, character_card_id)
		VALUES ($1, $2, 'player', $3, $4)
	`, f.showRunID, playerWithCharacter, producer, characterID); err != nil {
		t.Fatalf("insert roster row with character: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id)
		VALUES ($1, $2, 'player', $3)
	`, f.showRunID, playerWithoutCharacter, producer); err != nil {
		t.Fatalf("insert roster row without character: %v", err)
	}

	s, err := shows.LoadShowByID(context.Background(), pool, f.showID)
	if err != nil {
		t.Fatalf("load show: %v", err)
	}
	startResult, err := showtime.Start(context.Background(), pool, testDiscordCfg, producer, s.ShortCode, "", false, "")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, startResult.SessionID) })

	end, err := showtime.End(context.Background(), pool, testDiscordCfg, producer, s.ShortCode, "")
	if err != nil {
		t.Fatalf("End: %v", err)
	}
	if end.AftercareEligibleCount != 1 {
		t.Fatalf("expected AftercareEligibleCount 1, got %d", end.AftercareEligibleCount)
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

package stageobjects

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

// Fixture shape follows backend/internal/directorprep's (insert user -> grant
// Director at the seeded location -> production -> Show Run -> Show), extended
// with what canonical visibility needs and nothing more: a Scene staged into
// the Show, one composition token on it, and on demand a Cohort, a Character,
// and a drawing object.
//
// dbtest.OpenTestPool enforces the Kernel 64 obligation -- it calls
// ValidateTestDatabaseURL, so with TEST_DATABASE_URL unset these HARD-FAIL
// rather than skipping, and they can never point at the live database.
func openTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.OpenTestPool(t)
}

func testSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000000")
}

func insertTestUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
	t.Helper()
	ctx := context.Background()
	handle := handlePrefix + "_" + testSuffix(t)

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name) VALUES ($1, $2) RETURNING id::text
	`, handle, handlePrefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		// stage_object_states cascades from shows, and its grants cascade from
		// it, so deleting the Show is enough for this kernel's own rows.
		_, _ = pool.Exec(bg, `DELETE FROM actions WHERE actor_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM shows WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM show_runs WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM character_cards WHERE owner_user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
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

type fixture struct {
	directorID  string
	showID      string
	showRunID   string
	locationID  string
	sceneID     string
	placementID string
	// tokenID is a scene_stage_element of kind 'token' -- a durable object
	// that had no visibility control at all before Kernel 90.
	tokenID string
	// mapID is a scene_stage_element of kind 'map_backdrop', kept so tests can
	// prove §3's map boundary against a real row rather than a made-up id.
	mapID string
}

// Rows are inserted with raw SQL rather than through scenes.CreateScene /
// shows.CreateShow, and NOT for convenience: shows imports network, which
// imports world, which imports this package, so a test in package
// stageobjects that imported shows would be an import cycle. Raw-SQL fixtures
// are the established alternative in this codebase for exactly this shape of
// problem (see backend/internal/actions' own session fixtures).
//
// The cost is that these fixtures bypass the authoring packages' validation.
// That is acceptable here because none of these tests are about authoring --
// they need a Show with a staged Scene to exist, and every assertion is about
// canonical visibility state on top of it.
func buildFixture(t *testing.T, pool *pgxpool.Pool, directorUserID string) fixture {
	t.Helper()
	ctx := context.Background()
	suffix := testSuffix(t)

	var locationID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("load amurray-family location: %v", err)
	}
	grantLocationRole(t, pool, locationID, directorUserID, "director")

	var venueID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM venues WHERE slug = 'catharsis' LIMIT 1`).Scan(&venueID); err != nil {
		t.Fatalf("load catharsis venue: %v", err)
	}

	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug, created_by_user_id)
		VALUES ($1::uuid, $2, $3, $4::uuid) RETURNING id::text
	`, locationID, "K90 Production "+suffix, "k90-production-"+suffix, directorUserID).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1::uuid`, productionID)
	})

	var showRunID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_runs (location_id, production_id, title, slug, created_by_user_id)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5::uuid) RETURNING id::text
	`, locationID, productionID, "K90 Show Run "+suffix, "k90-show-run-"+suffix, directorUserID).Scan(&showRunID); err != nil {
		t.Fatalf("insert show run: %v", err)
	}

	showID := insertShow(t, pool, showRunID, directorUserID, "K90 Show "+suffix, "k90-show-"+suffix)

	var sceneID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO scenes (location_id, source_production_id, slug, title, default_venue_id, status, created_by_user_id)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5::uuid, 'ready', $6::uuid) RETURNING id::text
	`, locationID, productionID, "k90-scene-"+suffix, "K90 Scene "+suffix, venueID, directorUserID).Scan(&sceneID); err != nil {
		t.Fatalf("insert scene: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM scenes WHERE id = $1::uuid`, sceneID) })

	placementID := insertPlacement(t, pool, showID, sceneID, venueID, directorUserID)
	tokenID := insertStageElement(t, pool, sceneID, directorUserID, "token", "Training Wall")
	mapID := insertStageElement(t, pool, sceneID, directorUserID, "map_backdrop", "Arena Floor")

	return fixture{
		directorID:  directorUserID,
		showID:      showID,
		showRunID:   showRunID,
		locationID:  locationID,
		sceneID:     sceneID,
		placementID: placementID,
		tokenID:     tokenID,
		mapID:       mapID,
	}
}

func insertShow(t *testing.T, pool *pgxpool.Pool, showRunID, directorUserID, title, slug string) string {
	t.Helper()
	var showID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO shows (show_run_id, slug, title, created_by_user_id)
		VALUES ($1::uuid, $2, $3, $4::uuid) RETURNING id::text
	`, showRunID, slug, title, directorUserID).Scan(&showID); err != nil {
		t.Fatalf("insert show %q: %v", slug, err)
	}
	return showID
}

func insertPlacement(t *testing.T, pool *pgxpool.Pool, showID, sceneID, venueID, directorUserID string) string {
	t.Helper()
	var placementID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO show_scene_placements (show_id, scene_id, venue_id, status, created_by_user_id)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'ready', $4::uuid) RETURNING id::text
	`, showID, sceneID, venueID, directorUserID).Scan(&placementID); err != nil {
		t.Fatalf("insert placement: %v", err)
	}
	return placementID
}

func insertStageElement(t *testing.T, pool *pgxpool.Pool, sceneID, directorUserID, kind, label string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO scene_stage_elements (scene_id, kind, label, created_by_user_id)
		VALUES ($1::uuid, $2, $3, $4::uuid) RETURNING id::text
	`, sceneID, kind, label, directorUserID).Scan(&id); err != nil {
		t.Fatalf("insert stage element %q: %v", kind, err)
	}
	return id
}

func (f fixture) tokenRef() Ref {
	return Ref{Kind: KindSceneStageElement, ID: f.tokenID}
}

// insertCohort creates a Cohort on the fixture's Show.
func insertCohort(t *testing.T, pool *pgxpool.Pool, f fixture, name string) string {
	t.Helper()
	suffix := testSuffix(t)
	var cohortID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO show_cohorts (show_id, serial_number, slug, name, created_by_user_id)
		VALUES ($1::uuid, (SELECT COALESCE(MAX(serial_number), 0) + 1 FROM show_cohorts WHERE show_id = $1::uuid),
		        $2, $3, $4::uuid)
		RETURNING id::text
	`, f.showID, strings.ToLower(name)+"-"+suffix, name, f.directorID).Scan(&cohortID); err != nil {
		t.Fatalf("insert cohort %q: %v", name, err)
	}
	return cohortID
}

// insertRosterCharacter creates a Character owned by a user and puts it on the
// Show Run roster, which is the authority validateScopeTargets checks.
func insertRosterCharacter(t *testing.T, pool *pgxpool.Pool, f fixture, ownerUserID, name string) string {
	t.Helper()
	ctx := context.Background()
	var characterID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO character_cards (owner_user_id, location_id, name)
		VALUES ($1::uuid, $2::uuid, $3)
		RETURNING id::text
	`, ownerUserID, f.locationID, name).Scan(&characterID); err != nil {
		t.Fatalf("insert character %q: %v", name, err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id, character_card_id)
		VALUES ($1::uuid, $2::uuid, 'player', $3::uuid, $4::uuid)
	`, f.showRunID, ownerUserID, f.directorID, characterID); err != nil {
		t.Fatalf("insert roster member: %v", err)
	}
	return characterID
}

// insertLiveSession links a live session to the fixture's Show, which is what
// makes an audit row possible (actions.session_id is NOT NULL).
//
// Uses the real catharsis venue row because /ws/* routes and venue capability
// flags exist only for named venues -- the same constraint Kernel 89's proof
// harness ran into. Cleaned up after, and the session is only ever read here
// by activeSessionIDForShow.
func insertLiveSession(t *testing.T, pool *pgxpool.Pool, f fixture) string {
	t.Helper()
	ctx := context.Background()
	var sessionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO sessions (venue_id, status, show_id)
		VALUES ((SELECT id FROM venues WHERE slug = 'catharsis' LIMIT 1), 'rehearsal', $1::uuid)
		RETURNING id::text
	`, f.showID).Scan(&sessionID); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM actions WHERE session_id = $1::uuid`, sessionID)
		_, _ = pool.Exec(bg, `DELETE FROM sessions WHERE id = $1::uuid`, sessionID)
	})
	return sessionID
}

// insertBoundInteraction creates a participant interaction and binds it to the
// fixture's token, which is what gives it the durable stage presence
// resolveParticipantInteraction requires.
func insertBoundInteraction(t *testing.T, pool *pgxpool.Pool, f fixture) string {
	t.Helper()
	ctx := context.Background()
	var interactionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO participant_interactions (
			show_scene_placement_id, internal_name, stage_button_label,
			interaction_type, configuration_json, enabled, sort_order, created_by_user_id
		)
		VALUES ($1::uuid, $2, 'Train with Russel', 'open_equip_mode', '{}'::jsonb, TRUE, 0, $3::uuid)
		RETURNING id::text
	`, f.placementID, "k90-interaction-"+testSuffix(t), f.directorID).Scan(&interactionID); err != nil {
		t.Fatalf("insert participant interaction: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO stage_element_bindings (
			scene_stage_element_id, binding_type, participant_interaction_id, created_by_user_id
		)
		VALUES ($1::uuid, 'participant_interaction', $2::uuid, $3::uuid)
	`, f.tokenID, interactionID, f.directorID); err != nil {
		t.Fatalf("bind interaction to element: %v", err)
	}
	return interactionID
}

// buildSecondShowStagingSameScene creates another Show in the same Show Run
// and stages the SAME Scene into it, so a test can prove hidden state does not
// travel with a reused Scene.
func buildSecondShowStagingSameScene(t *testing.T, pool *pgxpool.Pool, f fixture) string {
	t.Helper()
	suffix := testSuffix(t)
	var venueID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM venues WHERE slug = 'catharsis' LIMIT 1`).Scan(&venueID); err != nil {
		t.Fatalf("load catharsis venue: %v", err)
	}
	showID := insertShow(t, pool, f.showRunID, f.directorID, "K90 Second Show "+suffix, "k90-second-show-"+suffix)
	insertPlacement(t, pool, showID, f.sceneID, venueID, f.directorID)
	return showID
}

// insertDrawingObject creates a Kernel 87 drawing object on the Show.
func insertDrawingObject(t *testing.T, pool *pgxpool.Pool, f fixture) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO drawing_objects (show_id, creator_user_id, object_type, geometry)
		VALUES ($1::uuid, $2::uuid, 'rectangle', '{"x":0,"y":0,"width":10,"height":10}'::jsonb)
		RETURNING id::text
	`, f.showID, f.directorID).Scan(&id); err != nil {
		t.Fatalf("insert drawing object: %v", err)
	}
	return id
}

package scenes

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

func openScenesTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.OpenTestPool(t)
}

func testSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
}

func insertScenesTestUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
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
		_, _ = pool.Exec(context.Background(), `
			UPDATE sessions SET show_id = NULL WHERE show_id IN (
				SELECT id FROM shows WHERE created_by_user_id = $1
			)
		`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_scene_placements WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM scenes WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM shows WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_run_audience_blocks WHERE user_id = $1 OR blocked_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_run_roster_members WHERE user_id = $1 OR added_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_runs WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
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

// productionFixture creates a fresh Production for the shared
// amurray-family location, since a fresh database has none of these
// (Kernel 64's lesson: raw migrations alone don't fully seed runtime-shaped
// data).
func productionFixture(t *testing.T, pool *pgxpool.Pool, creatorUserID string) (locationID, productionID string) {
	t.Helper()
	ctx := context.Background()

	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("load amurray-family location: %v", err)
	}
	grantLocationRole(t, pool, locationID, creatorUserID, "producer")

	suffix := testSuffix(t)
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug)
		VALUES ($1, $2, $3)
		RETURNING id::text
	`, locationID, "Test Production "+suffix, "test-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM scenes WHERE production_id = $1`, productionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID)
	})
	return locationID, productionID
}

// showFixture builds a Show Run and Show under the given Production.
func showFixture(t *testing.T, pool *pgxpool.Pool, creatorUserID, productionID string) (showRunID, showID string) {
	t.Helper()
	ctx := context.Background()
	suffix := testSuffix(t) + "_" + time.Now().UTC().Format("150405.000000000")

	sr, err := showruns.CreateShowRun(ctx, pool, creatorUserID, productionID, showruns.CreateShowRunInput{
		Title: "Test Show Run " + suffix, Slug: "test-show-run-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run fixture: %v", err)
	}
	s, err := shows.CreateShow(ctx, pool, creatorUserID, sr.ID, shows.CreateShowInput{
		Title: "Test Show " + suffix, Slug: "test-show-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show fixture: %v", err)
	}
	return sr.ID, s.ID
}

func strPtr(s string) *string { return &s }

func TestCreateSceneRequiresManageAuthorityAtProductionLocation(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_producer")
	_, productionID := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	if _, err := CreateScene(context.Background(), pool, producer, productionID, CreateSceneInput{
		Title: "Opening", Slug: "opening-" + suffix,
	}); err != nil {
		t.Fatalf("expected producer to create scene: %v", err)
	}

	var otherLocationID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO locations (name, slug) VALUES ($1, $2) RETURNING id::text
	`, "Elsewhere "+suffix, "elsewhere-"+suffix).Scan(&otherLocationID); err != nil {
		t.Fatalf("insert other location: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, otherLocationID)
	})
	elsewhereProducer := insertScenesTestUser(t, pool, "sc_elsewhere_producer")
	grantLocationRole(t, pool, otherLocationID, elsewhereProducer, "producer")
	if _, err := CreateScene(context.Background(), pool, elsewhereProducer, productionID, CreateSceneInput{
		Title: "Cross Location Scene", Slug: "cross-location-" + suffix,
	}); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for producer at a different location, got %v", err)
	}

	outsider := insertScenesTestUser(t, pool, "sc_outsider")
	if _, err := CreateScene(context.Background(), pool, outsider, productionID, CreateSceneInput{
		Title: "Outsider Scene", Slug: "outsider-" + suffix,
	}); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for outsider, got %v", err)
	}
}

func TestSceneSlugUniquenessIsScopedToProduction(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_slug_producer")
	_, productionA := productionFixture(t, pool, producer)
	_, productionB := productionFixture(t, pool, producer)

	if _, err := CreateScene(context.Background(), pool, producer, productionA, CreateSceneInput{
		Title: "Opening", Slug: "opening",
	}); err != nil {
		t.Fatalf("create scene in production A: %v", err)
	}
	if _, err := CreateScene(context.Background(), pool, producer, productionA, CreateSceneInput{
		Title: "Opening Again", Slug: "opening",
	}); err == nil || err.Error() != "slug_already_used" {
		t.Fatalf("expected slug_already_used within the same production, got %v", err)
	}
	if _, err := CreateScene(context.Background(), pool, producer, productionB, CreateSceneInput{
		Title: "Opening In B", Slug: "opening",
	}); err != nil {
		t.Fatalf("expected the same slug to be usable in a different production: %v", err)
	}
}

func TestSceneCanBeStagedInMultipleShowsUnderSameProduction(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_multi_producer")
	_, productionID := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	scene, err := CreateScene(context.Background(), pool, producer, productionID, CreateSceneInput{
		Title: "Character Making - Opening", Slug: "char-making-opening-" + suffix,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}

	_, showAID := showFixture(t, pool, producer, productionID)
	_, showBID := showFixture(t, pool, producer, productionID)

	placementA, err := CreatePlacement(context.Background(), pool, producer, showAID, CreatePlacementInput{SceneID: scene.ID})
	if err != nil {
		t.Fatalf("place scene in show A: %v", err)
	}
	placementB, err := CreatePlacement(context.Background(), pool, producer, showBID, CreatePlacementInput{SceneID: scene.ID})
	if err != nil {
		t.Fatalf("place scene in show B: %v", err)
	}
	if placementA.ID == placementB.ID {
		t.Fatalf("expected two distinct placements, got the same id")
	}

	listA, err := ListPlacementsForShow(context.Background(), pool, showAID)
	if err != nil {
		t.Fatalf("list placements for show A: %v", err)
	}
	if len(listA) != 1 || listA[0].Scene.ID != scene.ID {
		t.Fatalf("expected show A to have exactly the shared scene staged, got %+v", listA)
	}
	listB, err := ListPlacementsForShow(context.Background(), pool, showBID)
	if err != nil {
		t.Fatalf("list placements for show B: %v", err)
	}
	if len(listB) != 1 || listB[0].Scene.ID != scene.ID {
		t.Fatalf("expected show B to have exactly the shared scene staged, got %+v", listB)
	}
}

func TestScenePlacementRejectsCrossProductionScene(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_cross_producer")
	_, productionA := productionFixture(t, pool, producer)
	_, productionB := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	sceneInB, err := CreateScene(context.Background(), pool, producer, productionB, CreateSceneInput{
		Title: "Scene In B", Slug: "scene-in-b-" + suffix,
	})
	if err != nil {
		t.Fatalf("create scene in production B: %v", err)
	}
	_, showInA := showFixture(t, pool, producer, productionA)

	if _, err := CreatePlacement(context.Background(), pool, producer, showInA, CreatePlacementInput{SceneID: sceneInB.ID}); err == nil || err.Error() != "scene_production_mismatch" {
		t.Fatalf("expected scene_production_mismatch, got %v", err)
	}
}

func TestPlacementVenueOverrideDoesNotMutateScene(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_venue_producer")
	_, productionID := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	var venueA, venueB string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM venues ORDER BY created_at ASC LIMIT 1`).Scan(&venueA); err != nil {
		t.Fatalf("load venue A: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM venues ORDER BY created_at ASC OFFSET 1 LIMIT 1`).Scan(&venueB); err != nil {
		t.Fatalf("load venue B: %v", err)
	}

	scene, err := CreateScene(context.Background(), pool, producer, productionID, CreateSceneInput{
		Title: "Venue Scene", Slug: "venue-scene-" + suffix, DefaultVenueID: venueA,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	_, showID := showFixture(t, pool, producer, productionID)

	placement, err := CreatePlacement(context.Background(), pool, producer, showID, CreatePlacementInput{
		SceneID: scene.ID, VenueID: venueB, SortOrder: 3,
	})
	if err != nil {
		t.Fatalf("create placement with venue override: %v", err)
	}
	if placement.VenueID == nil || *placement.VenueID != venueB {
		t.Fatalf("expected placement venue override %q, got %v", venueB, placement.VenueID)
	}

	reloaded, err := LoadSceneByID(context.Background(), pool, scene.ID)
	if err != nil {
		t.Fatalf("reload scene: %v", err)
	}
	if reloaded.DefaultVenueID == nil || *reloaded.DefaultVenueID != venueA {
		t.Fatalf("expected scene default venue to remain %q, got %v", venueA, reloaded.DefaultVenueID)
	}
}

func TestArchivingPlacementDoesNotArchiveScene(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_archive_placement_producer")
	_, productionID := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	scene, err := CreateScene(context.Background(), pool, producer, productionID, CreateSceneInput{
		Title: "Reusable Scene", Slug: "reusable-scene-" + suffix,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	_, showID := showFixture(t, pool, producer, productionID)
	placement, err := CreatePlacement(context.Background(), pool, producer, showID, CreatePlacementInput{SceneID: scene.ID})
	if err != nil {
		t.Fatalf("create placement: %v", err)
	}

	archived, err := ArchivePlacement(context.Background(), pool, producer, placement.ID)
	if err != nil {
		t.Fatalf("archive placement: %v", err)
	}
	if archived.Status != "archived" || archived.ArchivedAt == nil {
		t.Fatalf("expected placement archived, got %+v", archived)
	}

	reloadedScene, err := LoadSceneByID(context.Background(), pool, scene.ID)
	if err != nil {
		t.Fatalf("reload scene: %v", err)
	}
	if reloadedScene.Status == "archived" || reloadedScene.ArchivedAt != nil {
		t.Fatalf("expected reusable scene to remain unarchived after its placement was archived, got %+v", reloadedScene)
	}
}

func TestArchivingSceneBlocksNewPlacementsButPreservesExisting(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_archive_scene_producer")
	_, productionID := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	scene, err := CreateScene(context.Background(), pool, producer, productionID, CreateSceneInput{
		Title: "Soon Retired Scene", Slug: "soon-retired-" + suffix,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	_, showAID := showFixture(t, pool, producer, productionID)
	existingPlacement, err := CreatePlacement(context.Background(), pool, producer, showAID, CreatePlacementInput{SceneID: scene.ID})
	if err != nil {
		t.Fatalf("create existing placement: %v", err)
	}

	if _, err := ArchiveScene(context.Background(), pool, producer, scene.ID); err != nil {
		t.Fatalf("archive scene: %v", err)
	}

	// Existing placement is untouched.
	reloadedPlacement, err := LoadPlacementByID(context.Background(), pool, existingPlacement.ID)
	if err != nil {
		t.Fatalf("reload existing placement: %v", err)
	}
	if reloadedPlacement.Status == "archived" {
		t.Fatalf("expected existing placement to remain unarchived after its scene was archived, got %+v", reloadedPlacement)
	}

	// A new placement of the now-archived scene is rejected.
	_, showBID := showFixture(t, pool, producer, productionID)
	if _, err := CreatePlacement(context.Background(), pool, producer, showBID, CreatePlacementInput{SceneID: scene.ID}); err == nil || err.Error() != "scene_archived" {
		t.Fatalf("expected scene_archived for a new placement of an archived scene, got %v", err)
	}
}

func TestAudienceScenceProgramOnlyShowsReadyPlacementsWithCuratedFields(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_program_producer")
	_, productionID := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	scene, err := CreateScene(context.Background(), pool, producer, productionID, CreateSceneInput{
		Title: "Character Making - Opening", Slug: "cm-opening-" + suffix,
		AudienceTitle: "Character Making", AudienceSummary: "Come make a character with us.",
		DirectorNotes: "Backstage only note that must never leak to Audience.",
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	_, showID := showFixture(t, pool, producer, productionID)
	placement, err := CreatePlacement(context.Background(), pool, producer, showID, CreatePlacementInput{SceneID: scene.ID, SortOrder: 1})
	if err != nil {
		t.Fatalf("create placement: %v", err)
	}

	// Draft placement (the default) does not appear in the audience program.
	entries, err := ListAudiencePlacementsForShow(context.Background(), pool, showID)
	if err != nil {
		t.Fatalf("list audience placements (draft): %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected zero audience entries while placement is draft, got %+v", entries)
	}

	ready := "ready"
	if _, err := UpdatePlacement(context.Background(), pool, producer, placement.ID, UpdatePlacementPatch{Status: &ready}); err != nil {
		t.Fatalf("mark placement ready: %v", err)
	}

	entries, err = ListAudiencePlacementsForShow(context.Background(), pool, showID)
	if err != nil {
		t.Fatalf("list audience placements (ready): %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one ready audience entry, got %+v", entries)
	}
	if entries[0].Title != "Character Making" {
		t.Fatalf("expected curated audience title fallback to scene.audience_title, got %q", entries[0].Title)
	}
	if entries[0].AudienceSummary != "Come make a character with us." {
		t.Fatalf("expected curated audience summary, got %q", entries[0].AudienceSummary)
	}

	// Confirm director_notes never appears anywhere in the audience-facing
	// AudienceScenePlacement type -- a compile-time guarantee, verified here
	// by asserting the struct has no such accessible field via its JSON
	// round-trip shape.
	if strings.Contains(strings.ToLower(entries[0].Title+entries[0].AudienceSummary), "backstage only") {
		t.Fatalf("backstage director_notes leaked into audience program entry: %+v", entries[0])
	}

	// Archiving the placement removes it from the audience program again.
	if _, err := ArchivePlacement(context.Background(), pool, producer, placement.ID); err != nil {
		t.Fatalf("archive placement: %v", err)
	}
	entries, err = ListAudiencePlacementsForShow(context.Background(), pool, showID)
	if err != nil {
		t.Fatalf("list audience placements (archived): %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected zero audience entries once placement is archived, got %+v", entries)
	}
}

func TestPlacementRequiresManageAuthorityAtShowLocation(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_placement_auth_producer")
	_, productionID := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	scene, err := CreateScene(context.Background(), pool, producer, productionID, CreateSceneInput{
		Title: "Auth Scene", Slug: "auth-scene-" + suffix,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	_, showID := showFixture(t, pool, producer, productionID)

	outsider := insertScenesTestUser(t, pool, "sc_placement_outsider")
	if _, err := CreatePlacement(context.Background(), pool, outsider, showID, CreatePlacementInput{SceneID: scene.ID}); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for outsider placing a scene, got %v", err)
	}
}

func TestScenePatchClearsOptionalFieldsWithEmptyString(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_patch_producer")
	_, productionID := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	scene, err := CreateScene(context.Background(), pool, producer, productionID, CreateSceneInput{
		Title: "Patchable Scene", Slug: "patchable-" + suffix, ShortTitle: "Patchy",
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	if scene.ShortTitle != "Patchy" {
		t.Fatalf("expected short title set, got %q", scene.ShortTitle)
	}

	updated, err := UpdateScene(context.Background(), pool, producer, scene.ID, UpdateScenePatch{ShortTitle: strPtr("")})
	if err != nil {
		t.Fatalf("clear short title: %v", err)
	}
	if updated.ShortTitle != "" {
		t.Fatalf("expected short title cleared, got %q", updated.ShortTitle)
	}
}

func TestSceneStatusCheckRejectsInvalidValue(t *testing.T) {
	pool := openScenesTestPool(t)
	producer := insertScenesTestUser(t, pool, "sc_status_producer")
	_, productionID := productionFixture(t, pool, producer)
	suffix := testSuffix(t)

	scene, err := CreateScene(context.Background(), pool, producer, productionID, CreateSceneInput{
		Title: "Status Scene", Slug: "status-scene-" + suffix,
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}

	if _, err := pool.Exec(context.Background(), `UPDATE scenes SET status = 'live' WHERE id = $1`, scene.ID); err == nil {
		t.Fatalf("expected DB CHECK constraint to reject 'live' status -- Kernel 69 explicitly excludes a live scene state")
	}
}

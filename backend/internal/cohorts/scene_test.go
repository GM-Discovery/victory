package cohorts

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/scenes"
)

// sceneFixture creates a reusable Scene at the Show's location and stages
// it into the Show, returning the placement id ActivateSceneForCohort
// expects.
func sceneFixture(t *testing.T, pool *pgxpool.Pool, director, showRunID, showID, title, slug string) string {
	t.Helper()
	ctx := context.Background()
	locationID := locationForShowRun(t, pool, showRunID)

	sc, err := scenes.CreateScene(ctx, pool, director, locationID, "", scenes.CreateSceneInput{
		Title: title, Slug: slug,
	})
	if err != nil {
		t.Fatalf("create scene %s: %v", slug, err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM scenes WHERE id = $1`, sc.ID) })

	p, err := scenes.CreatePlacement(ctx, pool, director, showID, scenes.CreatePlacementInput{SceneID: sc.ID})
	if err != nil {
		t.Fatalf("place scene %s into show: %v", slug, err)
	}
	return p.ID
}

func TestCohortSceneIsolationDoesNotAffectOtherCohorts(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "co_scene_director")
	showRunID, showID := showFixture(t, pool, director)
	ctx := context.Background()

	cohort1, err := CreateCohort(ctx, pool, director, showID)
	if err != nil {
		t.Fatalf("create cohort 1: %v", err)
	}
	cohort2, err := CreateCohort(ctx, pool, director, showID)
	if err != nil {
		t.Fatalf("create cohort 2: %v", err)
	}

	placementA := sceneFixture(t, pool, director, showRunID, showID, "Scene A", "scene-a-"+testSuffix(t))
	placementB := sceneFixture(t, pool, director, showRunID, showID, "Scene B", "scene-b-"+testSuffix(t))
	placementC := sceneFixture(t, pool, director, showRunID, showID, "Scene C", "scene-c-"+testSuffix(t))

	if _, err := ActivateSceneForCohort(ctx, pool, director, showID, cohort1.ID, placementA); err != nil {
		t.Fatalf("activate scene A for cohort 1: %v", err)
	}
	if _, err := ActivateSceneForCohort(ctx, pool, director, showID, cohort2.ID, placementB); err != nil {
		t.Fatalf("activate scene B for cohort 2: %v", err)
	}

	c1, err := LoadCohortByID(ctx, pool, cohort1.ID)
	if err != nil {
		t.Fatalf("load cohort 1: %v", err)
	}
	c2, err := LoadCohortByID(ctx, pool, cohort2.ID)
	if err != nil {
		t.Fatalf("load cohort 2: %v", err)
	}
	if c1.CurrentShowScenePlacementID == nil || *c1.CurrentShowScenePlacementID != placementA {
		t.Fatalf("expected cohort 1 on placement A, got %+v", c1.CurrentShowScenePlacementID)
	}
	if c2.CurrentShowScenePlacementID == nil || *c2.CurrentShowScenePlacementID != placementB {
		t.Fatalf("expected cohort 2 on placement B, got %+v", c2.CurrentShowScenePlacementID)
	}

	// Moving cohort 1 to Scene C must leave cohort 2 on Scene B, untouched.
	if _, err := ActivateSceneForCohort(ctx, pool, director, showID, cohort1.ID, placementC); err != nil {
		t.Fatalf("move cohort 1 to scene C: %v", err)
	}
	c1, err = LoadCohortByID(ctx, pool, cohort1.ID)
	if err != nil {
		t.Fatalf("reload cohort 1: %v", err)
	}
	c2, err = LoadCohortByID(ctx, pool, cohort2.ID)
	if err != nil {
		t.Fatalf("reload cohort 2: %v", err)
	}
	if c1.CurrentShowScenePlacementID == nil || *c1.CurrentShowScenePlacementID != placementC {
		t.Fatalf("expected cohort 1 now on placement C, got %+v", c1.CurrentShowScenePlacementID)
	}
	if c2.CurrentShowScenePlacementID == nil || *c2.CurrentShowScenePlacementID != placementB {
		t.Fatalf("expected cohort 2 to remain on placement B after cohort 1 moved, got %+v", c2.CurrentShowScenePlacementID)
	}
}

func TestActivateSceneForCohortRejectsPlacementFromAnotherShow(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "co_scene_cross_director")
	showRunID, showID := showFixture(t, pool, director)
	_, otherShowID := showFixture(t, pool, director)
	ctx := context.Background()

	cohort1, err := CreateCohort(ctx, pool, director, showID)
	if err != nil {
		t.Fatalf("create cohort: %v", err)
	}
	foreignPlacement := sceneFixture(t, pool, director, showRunID, otherShowID, "Foreign Scene", "foreign-scene-"+testSuffix(t))

	if _, err := ActivateSceneForCohort(ctx, pool, director, showID, cohort1.ID, foreignPlacement); err == nil || err.Error() != "placement_show_mismatch" {
		t.Fatalf("expected placement_show_mismatch, got %v", err)
	}
}

func TestUngroupedViewerResolvesNoCohortPlacement(t *testing.T) {
	pool := openTestPool(t)
	director := insertTestUser(t, pool, "co_ungrouped_director")
	showRunID, showID := showFixture(t, pool, director)
	locationID := locationForShowRun(t, pool, showRunID)
	alice, _ := playerFixture(t, pool, director, showRunID, locationID, "co_ungrouped_alice")
	ctx := context.Background()

	placementID, cohortID, err := ResolveCurrentPlacementForViewer(ctx, pool, showID, alice)
	if err != nil {
		t.Fatalf("resolve placement for ungrouped viewer: %v", err)
	}
	if placementID != "" || cohortID != "" {
		t.Fatalf("expected no cohort placement for an ungrouped viewer, got placement=%q cohort=%q", placementID, cohortID)
	}
}


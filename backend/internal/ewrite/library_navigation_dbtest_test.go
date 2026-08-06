package ewrite

// Kernel 79 Goal F (ruleset-wide Library navigation): proves the new
// collection-subtree scoping shared by SearchPublications and
// SearchDirectoryEntries actually confines a search to one Ruleset's
// descendants -- a Series two levels down must be included, a sibling
// Ruleset must not. HandleLibraryCollection/HandleLibraryPublication's
// breadcrumb wiring has no unit-level function to call directly (same as
// the pre-existing HandleLibraryTree/HandleLibraryPublication, which have
// never had httptest coverage in this package) -- proven live instead, per
// this package's established pattern for the Library reading surface.

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
)

func TestSearchPublicationsScopedToCollection(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_navsearch_crew")
	grantRole(t, pool, loc, crew, "crew")

	rulesetA := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Nav Ruleset A")
	seriesA := mustCreateCollection(t, pool, crew, loc, rulesetA.ID, "series", "Nav Series A")
	moduleA := mustCreateCollection(t, pool, crew, loc, seriesA.ID, "module", "Nav Module A")
	pubA := mustCreatePublication(t, pool, crew, moduleA.ID, "Deep Module Publication")
	mustSave(t, pool, crew, pubA.ID, "# Deep Module Publication\n\nquorlathorn appears here, two levels down", "")
	if _, err := PublishPublication(ctx, pool, crew, pubA.ID); err != nil {
		t.Fatalf("publish A: %v", err)
	}

	rulesetB := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Nav Ruleset B")
	pubB := mustCreatePublication(t, pool, crew, rulesetB.ID, "Sibling Ruleset Publication")
	mustSave(t, pool, crew, pubB.ID, "# Sibling Ruleset Publication\n\nquorlathorn also appears here, unrelated ruleset", "")
	if _, err := PublishPublication(ctx, pool, crew, pubB.ID); err != nil {
		t.Fatalf("publish B: %v", err)
	}

	unscoped, err := SearchPublications(ctx, pool, crew, false, "quorlathorn", "", 10)
	if err != nil {
		t.Fatalf("unscoped search: %v", err)
	}
	if len(unscoped) != 2 {
		t.Fatalf("expected both publications unscoped, got %+v", unscoped)
	}

	scopedToA, err := SearchPublications(ctx, pool, crew, false, "quorlathorn", rulesetA.ID, 10)
	if err != nil {
		t.Fatalf("scoped search (ruleset): %v", err)
	}
	if len(scopedToA) != 1 || scopedToA[0].PublicationID != pubA.ID {
		t.Fatalf("expected only the deep Module publication under Ruleset A, got %+v", scopedToA)
	}

	// Scoping to the Series (one level down, not the Ruleset root) must
	// still reach the Module publication two levels below it.
	scopedToSeries, err := SearchPublications(ctx, pool, crew, false, "quorlathorn", seriesA.ID, 10)
	if err != nil {
		t.Fatalf("scoped search (series): %v", err)
	}
	if len(scopedToSeries) != 1 || scopedToSeries[0].PublicationID != pubA.ID {
		t.Fatalf("expected the Module publication reachable from its Series, got %+v", scopedToSeries)
	}

	scopedToB, err := SearchPublications(ctx, pool, crew, false, "quorlathorn", rulesetB.ID, 10)
	if err != nil {
		t.Fatalf("scoped search (ruleset B): %v", err)
	}
	if len(scopedToB) != 1 || scopedToB[0].PublicationID != pubB.ID {
		t.Fatalf("Ruleset B's scoped search must not see Ruleset A's publication, got %+v", scopedToB)
	}
}

func TestSearchDirectoryEntriesScopedToCollection(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_navdirsearch_crew")
	grantRole(t, pool, loc, crew, "crew")

	rulesetA := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Nav Dir Ruleset A")
	dirA := insertTestDirectory(t, pool, rulesetA.ID, "skill", "Ruleset A Directory")
	insertTestDirectoryEntry(t, pool, dirA, "SKILL_NAV_A", "Quorlathorn Sense", "Awareness")

	rulesetB := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Nav Dir Ruleset B")
	dirB := insertTestDirectory(t, pool, rulesetB.ID, "skill", "Ruleset B Directory")
	insertTestDirectoryEntry(t, pool, dirB, "SKILL_NAV_B", "Quorlathorn Craft", "Craft")

	unscoped, err := SearchDirectoryEntries(ctx, pool, crew, "quorlathorn", "", 10)
	if err != nil {
		t.Fatalf("unscoped: %v", err)
	}
	if len(unscoped) != 2 {
		t.Fatalf("expected both directory entries unscoped, got %+v", unscoped)
	}

	scopedToA, err := SearchDirectoryEntries(ctx, pool, crew, "quorlathorn", rulesetA.ID, 10)
	if err != nil {
		t.Fatalf("scoped to A: %v", err)
	}
	if len(scopedToA) != 1 || scopedToA[0].CanonicalName != "Quorlathorn Sense" {
		t.Fatalf("expected only Ruleset A's entry, got %+v", scopedToA)
	}
}

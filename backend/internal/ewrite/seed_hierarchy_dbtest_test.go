package ewrite

// Kernel 79: proves the Socio Series hierarchy (Core Rulebook / Quickstart
// / Niava) is created correctly, that the pre-existing Core Rulebook
// publication is reparented without losing its identity, and that the
// full boot-seed sequence is idempotent -- this is the regression test for
// the real bug found while building this kernel: reparenting the Core
// Rulebook publication out from directly-under-the-ruleset broke
// EnsureCanonicalSocioManuscript's own join-based existence check, which
// then tried to recreate a duplicate on every subsequent boot.

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
)

func TestSocioSeriesHierarchyAndManuscriptSeedsAreIdempotent(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)

	runBootSequence := func() {
		if err := EnsureCanonicalSocioManuscript(ctx, pool); err != nil {
			t.Fatalf("EnsureCanonicalSocioManuscript: %v", err)
		}
		if err := EnsureSocioSeriesHierarchy(ctx, pool); err != nil {
			t.Fatalf("EnsureSocioSeriesHierarchy: %v", err)
		}
		if err := EnsureQuickstartManuscript(ctx, pool); err != nil {
			t.Fatalf("EnsureQuickstartManuscript: %v", err)
		}
		if err := EnsureNiavaManuscript(ctx, pool); err != nil {
			t.Fatalf("EnsureNiavaManuscript: %v", err)
		}
	}

	// First boot (a totally fresh victory_test database already ran this
	// via the compiled binary per reset-test-database.sh; calling it
	// directly here makes the test meaningful regardless of DB history).
	runBootSequence()

	var rulesetID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text FROM ewrite_collections WHERE location_id = $1 AND slug = $2 AND kind = 'ruleset'
	`, loc, socioSeedRulesetSlug).Scan(&rulesetID); err != nil {
		t.Fatalf("ruleset not found: %v", err)
	}

	for _, series := range []struct{ slug, title string }{
		{socioCoreRulebookSeriesSlug, socioCoreRulebookSeriesTitle},
		{socioQuickstartSeriesSlug, socioQuickstartSeriesTitle},
		{socioNiavaSeriesSlug, socioNiavaSeriesTitle},
	} {
		var parentID, title string
		if err := pool.QueryRow(ctx, `
			SELECT COALESCE(parent_id::text, ''), title FROM ewrite_collections
			WHERE location_id = $1 AND slug = $2 AND kind = 'series'
		`, loc, series.slug).Scan(&parentID, &title); err != nil {
			t.Fatalf("series %q not found: %v", series.slug, err)
		}
		if parentID != rulesetID {
			t.Fatalf("series %q parent = %q, want ruleset %q", series.slug, parentID, rulesetID)
		}
		if title != series.title {
			t.Fatalf("series %q title = %q, want %q", series.slug, title, series.title)
		}
	}

	var coreRulebookID, coreRulebookCollectionSlug string
	if err := pool.QueryRow(ctx, `
		SELECT p.id::text, c.slug
		FROM ewrite_publications p JOIN ewrite_collections c ON c.id = p.collection_id
		WHERE p.location_id = $1 AND p.slug = $2
	`, loc, socioSeedPublicationSlug).Scan(&coreRulebookID, &coreRulebookCollectionSlug); err != nil {
		t.Fatalf("core rulebook publication not found: %v", err)
	}
	if coreRulebookCollectionSlug != socioCoreRulebookSeriesSlug {
		t.Fatalf("core rulebook publication sits under collection slug %q, want %q", coreRulebookCollectionSlug, socioCoreRulebookSeriesSlug)
	}

	for _, want := range []struct{ pubSlug, seriesSlug string }{
		{socioQuickstartPublicationSlug, socioQuickstartSeriesSlug},
		{socioNiavaPublicationSlug, socioNiavaSeriesSlug},
	} {
		var wordCount int
		var collectionSlug string
		if err := pool.QueryRow(ctx, `
			SELECT p.word_count, c.slug
			FROM ewrite_publications p JOIN ewrite_collections c ON c.id = p.collection_id
			WHERE p.location_id = $1 AND p.slug = $2
		`, loc, want.pubSlug).Scan(&wordCount, &collectionSlug); err != nil {
			t.Fatalf("publication %q not found: %v", want.pubSlug, err)
		}
		if collectionSlug != want.seriesSlug {
			t.Fatalf("publication %q sits under collection slug %q, want %q", want.pubSlug, collectionSlug, want.seriesSlug)
		}
		if wordCount < 1000 {
			t.Fatalf("publication %q word count = %d, expected the real manuscript's content", want.pubSlug, wordCount)
		}
	}

	// Second boot must be a pure no-op: no duplicate series, no duplicate
	// publications, no error -- this is the regression test for the bug.
	runBootSequence()

	var seriesCount, pubCount, coreRulebookCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM ewrite_collections WHERE location_id = $1 AND kind = 'series'
	`, loc).Scan(&seriesCount); err != nil {
		t.Fatalf("count series: %v", err)
	}
	if seriesCount != 3 {
		t.Fatalf("expected exactly 3 series after two boots, got %d", seriesCount)
	}
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM ewrite_publications WHERE location_id = $1
	`, loc).Scan(&pubCount); err != nil {
		t.Fatalf("count publications: %v", err)
	}
	if pubCount != 3 {
		t.Fatalf("expected exactly 3 publications after two boots, got %d", pubCount)
	}
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM ewrite_publications WHERE location_id = $1 AND slug = $2
	`, loc, socioSeedPublicationSlug).Scan(&coreRulebookCount); err != nil {
		t.Fatalf("count core rulebook: %v", err)
	}
	if coreRulebookCount != 1 {
		t.Fatalf("expected exactly 1 core rulebook publication after two boots, got %d", coreRulebookCount)
	}

	var afterID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text FROM ewrite_publications WHERE location_id = $1 AND slug = $2
	`, loc, socioSeedPublicationSlug).Scan(&afterID); err != nil {
		t.Fatalf("reload core rulebook: %v", err)
	}
	if afterID != coreRulebookID {
		t.Fatalf("core rulebook publication ID changed across boots: %q -> %q (must be stable so object-links survive)", coreRulebookID, afterID)
	}
}

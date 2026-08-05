package ewrite

// Kernel 79A: proves EnsureSkillDirectory auto-matches real Core Rulebook
// sections by exact title ("{Name} ({AttributeName})", the manuscript's own
// catalogue-appendix heading shape -- verified directly against
// seed/socio-v1.1.md before writing this, not assumed), that a catalogue
// entry with no matching heading comes back unlinked rather than erroring,
// and that a second boot never reverts a Crew+ member's manual re-curation
// (create-if-absent-ONLY, same rule as EnsureCanonicalSocioManuscript).

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
)

func TestEnsureSkillDirectorySeedsAndIsIdempotent(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_skilldir_crew")
	grantRole(t, pool, loc, crew, "crew")

	if err := EnsureCanonicalSocioManuscript(ctx, pool); err != nil {
		t.Fatalf("EnsureCanonicalSocioManuscript: %v", err)
	}
	if err := EnsureSocioSeriesHierarchy(ctx, pool); err != nil {
		t.Fatalf("EnsureSocioSeriesHierarchy: %v", err)
	}

	catalogue := []SkillCatalogueEntry{
		{ID: "SKILL_AWARENESS_ALERTNESS", Name: "Alertness", AttributeName: "Awareness", CardDescription: "Fast-twitch attention to sudden change"},
		{ID: "SKILL_MIGHT_INTIMIDATION_PHYSICAL", Name: "Intimidation (Physical)", AttributeName: "Might", CardDescription: "Physical menace"},
		{ID: "SKILL_TEST_NOT_IN_MANUSCRIPT", Name: "Wholly Fictional Skill", AttributeName: "Lore", CardDescription: "Does not exist in the real manuscript"},
	}

	if err := EnsureSkillDirectory(ctx, pool, catalogue); err != nil {
		t.Fatalf("EnsureSkillDirectory (first boot): %v", err)
	}

	var directoryID string
	if err := pool.QueryRow(ctx, `
		SELECT d.id::text FROM ewrite_directories d
		JOIN ewrite_collections c ON c.id = d.collection_id
		WHERE c.location_id = $1 AND c.slug = $2 AND d.slug = $3
	`, loc, socioSeedRulesetSlug, skillDirectorySlug).Scan(&directoryID); err != nil {
		t.Fatalf("skill directory not found: %v", err)
	}

	// Not asserting an exact entry count: if this test database's real
	// binary has already booted once (scripts/test/setup-test-database.sh
	// does exactly that), the directory already holds the real 100-skill
	// catalogue and this call is a no-op for any external_ref that
	// collides with a real one -- by design, see EnsureSkillDirectory's
	// ON CONFLICT DO NOTHING. Assert on the three refs by name instead.
	entries, err := ListDirectoryEntries(ctx, pool, crew, directoryID, "", "")
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	byRef := map[string]DirectoryEntry{}
	for _, e := range entries {
		byRef[e.ExternalRef] = e
	}
	for _, ref := range []string{"SKILL_AWARENESS_ALERTNESS", "SKILL_MIGHT_INTIMIDATION_PHYSICAL", "SKILL_TEST_NOT_IN_MANUSCRIPT"} {
		if _, ok := byRef[ref]; !ok {
			t.Fatalf("expected entry %q to exist after seeding, got entries %+v", ref, byRef)
		}
	}

	alertness := byRef["SKILL_AWARENESS_ALERTNESS"]
	if alertness.LinkStatus != "linked" || alertness.TargetSectionTitle != "Alertness (Awareness)" {
		t.Fatalf("expected Alertness auto-matched to its real manuscript section, got %+v", alertness)
	}
	intimidation := byRef["SKILL_MIGHT_INTIMIDATION_PHYSICAL"]
	if intimidation.LinkStatus != "linked" || intimidation.TargetSectionTitle != "Intimidation (Physical) (Might)" {
		t.Fatalf("expected disambiguated Intimidation (Physical) auto-matched, got %+v", intimidation)
	}
	fictional := byRef["SKILL_TEST_NOT_IN_MANUSCRIPT"]
	if fictional.LinkStatus != "unlinked" {
		t.Fatalf("expected a catalogue entry with no matching heading to stay unlinked, got %+v", fictional)
	}

	// Crew+ manually re-curates the fictional entry's link.
	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Manual Curation Ruleset")
	manualPub := mustCreatePublication(t, pool, crew, ruleset.ID, "Manually Curated Rules")
	mustSave(t, pool, crew, manualPub.ID, "# Manually Curated Rules\n\ncontent", "")
	if _, err := PublishPublication(ctx, pool, crew, manualPub.ID); err != nil {
		t.Fatalf("publish manual pub: %v", err)
	}
	if _, err := SetDirectoryEntryLink(ctx, pool, crew, fictional.ID, manualPub.ID, ""); err != nil {
		t.Fatalf("manual curation: %v", err)
	}

	// Second boot (catalogue unchanged, same idempotent seed run again) must
	// not touch the manually-curated entry or duplicate anything.
	if err := EnsureSkillDirectory(ctx, pool, catalogue); err != nil {
		t.Fatalf("EnsureSkillDirectory (second boot): %v", err)
	}
	entries, err = ListDirectoryEntries(ctx, pool, crew, directoryID, "", "")
	if err != nil {
		t.Fatalf("list entries after second boot: %v", err)
	}
	for _, e := range entries {
		if e.ExternalRef == "SKILL_TEST_NOT_IN_MANUSCRIPT" {
			if e.LinkStatus != "linked" || e.TargetPublicationID != manualPub.ID {
				t.Fatalf("second boot must not revert Crew+'s manual curation, got %+v", e)
			}
		}
	}
}


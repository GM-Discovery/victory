package ewrite

// Kernel 79A tests: the reusable directory abstraction (spec 5) --
// entry link-status resolution (linked/unlinked/hidden), the dual
// authority check on curating a link, safe degradation on a target
// mismatch, and the Character-skill rule-link resolver that feeds
// characters.ListCharacterSkills.

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

func insertTestDirectory(t *testing.T, pool *pgxpool.Pool, collectionID, directoryType, title string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO ewrite_directories (collection_id, directory_type, title, slug)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, collectionID, directoryType, title, "dir-"+testSuffix(t)).Scan(&id); err != nil {
		t.Fatalf("insert directory: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM ewrite_directories WHERE id = $1`, id)
	})
	return id
}

func insertTestDirectoryEntry(t *testing.T, pool *pgxpool.Pool, directoryID, externalRef, name, category string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO ewrite_directory_entries (directory_id, external_ref, canonical_name, category, aliases)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text
	`, directoryID, externalRef, name, category, []string{"alias-of-" + name}).Scan(&id); err != nil {
		t.Fatalf("insert directory entry: %v", err)
	}
	return id
}

// TestDirectoryEntryLinkStatusResolution proves the three link-status
// states (spec 6.4): unlinked (no target curated), hidden (a target
// exists but this reader can't read it -- a draft publication must never
// leak its title through the directory), and linked (target resolved).
func TestDirectoryEntryLinkStatusResolution(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_dir_crew")
	cast := insertTestUser(t, pool, "ew_dir_cast")
	grantRole(t, pool, loc, crew, "crew")
	grantRole(t, pool, loc, cast, "cast")

	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Directory Test Ruleset")
	pub := mustCreatePublication(t, pool, crew, ruleset.ID, "Directory Rules")
	mustSave(t, pool, crew, pub.ID, "# Directory Rules\n\n## Alertness\n\nfast-twitch rules", "")

	sections, err := LoadSections(ctx, pool, pub.ID)
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	var alertness Section
	for _, s := range sections {
		if s.Title == "Alertness" {
			alertness = s
		}
	}
	if alertness.ID == "" {
		t.Fatal("alertness section missing")
	}

	dir := insertTestDirectory(t, pool, ruleset.ID, "skill", "Test Skill Directory")
	unlinkedID := insertTestDirectoryEntry(t, pool, dir, "SKILL_TEST_UNLINKED", "Zzz Unlinked Skill", "Awareness")
	linkedID := insertTestDirectoryEntry(t, pool, dir, "SKILL_TEST_ALERTNESS", "Alertness", "Awareness")

	// Draft target: link exists but must resolve to "hidden", never leak
	// the publication title to a reader without edit authority.
	if _, err := SetDirectoryEntryLink(ctx, pool, crew, linkedID, pub.ID, alertness.ID); err != nil {
		t.Fatalf("set link: %v", err)
	}

	entries, err := ListDirectoryEntries(ctx, pool, cast, dir, "", "")
	if err != nil {
		t.Fatalf("list entries as cast: %v", err)
	}
	statuses := map[string]DirectoryEntry{}
	for _, e := range entries {
		statuses[e.ID] = e
	}
	if statuses[unlinkedID].LinkStatus != "unlinked" {
		t.Fatalf("expected unlinked, got %q", statuses[unlinkedID].LinkStatus)
	}
	if got := statuses[linkedID]; got.LinkStatus != "hidden" || got.TargetPublicationTitle != "" || got.TargetPublicationID != "" {
		t.Fatalf("draft target must resolve hidden with no leaked fields, got %+v", got)
	}

	if _, err := PublishPublication(ctx, pool, crew, pub.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}

	entries, err = ListDirectoryEntries(ctx, pool, cast, dir, "", "")
	if err != nil {
		t.Fatalf("list entries after publish: %v", err)
	}
	for _, e := range entries {
		if e.ID == linkedID {
			if e.LinkStatus != "linked" || e.TargetPublicationTitle != "Directory Rules" || e.TargetSectionAnchor != "alertness" {
				t.Fatalf("expected resolved linked entry, got %+v", e)
			}
		}
	}

	// Search matches canonical_name and alias substrings.
	found, err := ListDirectoryEntries(ctx, pool, cast, dir, "alert", "")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(found) != 1 || found[0].ID != linkedID {
		t.Fatalf("expected search 'alert' to match only Alertness, got %+v", found)
	}
	found, err = ListDirectoryEntries(ctx, pool, cast, dir, "alias-of-zzz", "")
	if err != nil {
		t.Fatalf("alias search: %v", err)
	}
	if len(found) != 1 || found[0].ID != unlinkedID {
		t.Fatalf("expected alias search to match Zzz Unlinked Skill, got %+v", found)
	}

	// Category filter.
	found, err = ListDirectoryEntries(ctx, pool, cast, dir, "", "Awareness")
	if err != nil {
		t.Fatalf("category filter: %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("expected both entries under Awareness, got %d", len(found))
	}
	found, err = ListDirectoryEntries(ctx, pool, cast, dir, "", "Might")
	if err != nil {
		t.Fatalf("category filter (no match): %v", err)
	}
	if len(found) != 0 {
		t.Fatalf("expected zero entries under Might, got %d", len(found))
	}
}

// TestSetDirectoryEntryLinkAuthority proves the dual gate: Crew+ at the
// directory's own location may curate it, but only when they also hold
// edit authority on the target publication -- an author must not wire
// another location's private draft into this directory.
func TestSetDirectoryEntryLinkAuthority(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	otherLoc := insertTestLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_dirauth_crew")
	cast := insertTestUser(t, pool, "ew_dirauth_cast")
	otherCrew := insertTestUser(t, pool, "ew_dirauth_other_crew")
	grantRole(t, pool, loc, crew, "crew")
	grantRole(t, pool, loc, cast, "cast")
	grantRole(t, pool, otherLoc, otherCrew, "crew")

	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Auth Ruleset")
	pub := mustCreatePublication(t, pool, crew, ruleset.ID, "Auth Rules")
	mustSave(t, pool, crew, pub.ID, "# Auth Rules\n\ncontent", "")

	otherRuleset := mustCreateCollection(t, pool, otherCrew, otherLoc, "", "ruleset", "Other Ruleset")
	otherPub := mustCreatePublication(t, pool, otherCrew, otherRuleset.ID, "Other Rules")
	mustSave(t, pool, otherCrew, otherPub.ID, "# Other Rules\n\ncontent", "")

	dir := insertTestDirectory(t, pool, ruleset.ID, "skill", "Auth Directory")
	entry := insertTestDirectoryEntry(t, pool, dir, "SKILL_AUTH_TEST", "Auth Test Skill", "Craft")

	// Cast (not Crew+) cannot curate this directory at all.
	if _, err := SetDirectoryEntryLink(ctx, pool, cast, entry, pub.ID, ""); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected cast denial, got %v", err)
	}

	// Crew+ at the directory's location, but no edit authority on the
	// OTHER location's publication -- must not be able to wire it in.
	if _, err := SetDirectoryEntryLink(ctx, pool, crew, entry, otherPub.ID, ""); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected cross-location denial, got %v", err)
	}

	// Crew+ with edit authority on their own publication: succeeds, and
	// resolves "linked" even pre-publish since CanReadPublication treats
	// edit authority as draft-read authority (package doc in authority.go).
	updated, err := SetDirectoryEntryLink(ctx, pool, crew, entry, pub.ID, "")
	if err != nil {
		t.Fatalf("set link: %v", err)
	}
	if updated.LinkStatus != "linked" || updated.TargetPublicationID != pub.ID {
		t.Fatalf("expected crew's own draft to resolve linked, got %+v", updated)
	}
}

// TestSetDirectoryEntryLinkClearAndMismatch proves clearing a link back to
// "unlinked" and that a section from a different publication than the one
// being linked is rejected outright rather than silently mismatched.
func TestSetDirectoryEntryLinkClearAndMismatch(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_dirclear_crew")
	grantRole(t, pool, loc, crew, "crew")

	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Clear Ruleset")
	pubA := mustCreatePublication(t, pool, crew, ruleset.ID, "Pub A")
	mustSave(t, pool, crew, pubA.ID, "# Pub A\n\n## Section A\n\ntext", "")
	pubB := mustCreatePublication(t, pool, crew, ruleset.ID, "Pub B")
	mustSave(t, pool, crew, pubB.ID, "# Pub B\n\n## Section B\n\ntext", "")

	sectionsB, err := LoadSections(ctx, pool, pubB.ID)
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	var sectionB Section
	for _, s := range sectionsB {
		if s.Title == "Section B" {
			sectionB = s
		}
	}

	dir := insertTestDirectory(t, pool, ruleset.ID, "skill", "Clear Directory")
	entry := insertTestDirectoryEntry(t, pool, dir, "SKILL_CLEAR_TEST", "Clear Test Skill", "Grace")

	// pubA + a section that belongs to pubB must be rejected.
	if _, err := SetDirectoryEntryLink(ctx, pool, crew, entry, pubA.ID, sectionB.ID); err == nil || err.Error() != "section_publication_mismatch" {
		t.Fatalf("expected section_publication_mismatch, got %v", err)
	}

	if _, err := SetDirectoryEntryLink(ctx, pool, crew, entry, pubA.ID, ""); err != nil {
		t.Fatalf("set link: %v", err)
	}
	if _, err := SetDirectoryEntryLink(ctx, pool, crew, entry, "", ""); err != nil {
		t.Fatalf("clear link: %v", err)
	}
	entries, err := ListDirectoryEntries(ctx, pool, crew, dir, "", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 1 || entries[0].LinkStatus != "unlinked" {
		t.Fatalf("expected cleared entry to read back unlinked, got %+v", entries)
	}
}

// TestRuleLinksForCharacterSkills proves the Character-sheet resolver:
// only a published target resolves, keyed by the catalogue skill ID
// (external_ref), across the 'skill' directory type specifically.
func TestRuleLinksForCharacterSkills(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_charskill_crew")
	grantRole(t, pool, loc, crew, "crew")

	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Char Skill Ruleset")
	pub := mustCreatePublication(t, pool, crew, ruleset.ID, "Char Skill Rules")
	mustSave(t, pool, crew, pub.ID, "# Char Skill Rules\n\n## Insight\n\ntext", "")
	sections, err := LoadSections(ctx, pool, pub.ID)
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	var insight Section
	for _, s := range sections {
		if s.Title == "Insight" {
			insight = s
		}
	}

	// Deliberately NOT a real characters.Chapter4Skills ID -- the real
	// catalogue is already boot-seeded into the canonical Skill Directory
	// on this test database (setup-test-database.sh boots the real
	// binary), so a real ID here would collide with that live entry and
	// make this test's own draft/published assertions meaningless.
	const testSkillRef = "SKILL_TEST_RULELINK_INSIGHT"
	dir := insertTestDirectory(t, pool, ruleset.ID, "skill", "Char Skill Directory")
	entry := insertTestDirectoryEntry(t, pool, dir, testSkillRef, "Insight", "Awareness")
	if _, err := SetDirectoryEntryLink(ctx, pool, crew, entry, pub.ID, insight.ID); err != nil {
		t.Fatalf("set link: %v", err)
	}

	links, err := RuleLinksForCharacterSkills(ctx, pool, []string{testSkillRef, "SKILL_NOT_LINKED"})
	if err != nil {
		t.Fatalf("resolve (draft): %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("draft publication must not resolve into character payloads: %+v", links)
	}

	if _, err := PublishPublication(ctx, pool, crew, pub.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	links, err = RuleLinksForCharacterSkills(ctx, pool, []string{testSkillRef, "SKILL_NOT_LINKED"})
	if err != nil {
		t.Fatalf("resolve (published): %v", err)
	}
	l, ok := links[testSkillRef]
	if !ok || l.SectionAnchor != "insight" || l.PublicationID != pub.ID {
		t.Fatalf("expected resolved rule link for Insight, got %+v", links)
	}
	if _, ok := links["SKILL_NOT_LINKED"]; ok {
		t.Fatalf("unlinked skill must not appear in the result map")
	}
}

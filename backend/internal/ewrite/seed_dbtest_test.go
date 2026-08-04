package ewrite

// Kernel 78: proves the canonical Socio seed creates real, readable,
// searchable content, and -- critically -- never touches it again once
// created (protecting a Producer's later edits from a reseed on redeploy).

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
)

func TestEnsureCanonicalSocioManuscriptSeedsAndIsIdempotent(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)

	// A totally fresh victory_test database created by reset-test-database.sh
	// already runs this at boot (fresh-install.sh proves that path); here we
	// call it directly so the test is meaningful even against a database
	// that was reset without going through the compiled binary.
	if err := EnsureCanonicalSocioManuscript(ctx, pool); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var pubID, status, visibility string
	var wordCount int
	if err := pool.QueryRow(ctx, `
		SELECT p.id::text, p.status, p.visibility, p.word_count
		FROM ewrite_publications p
		JOIN ewrite_collections c ON c.id = p.collection_id
		WHERE c.location_id = $1 AND c.slug = $2 AND p.slug = $3
	`, loc, socioSeedRulesetSlug, socioSeedPublicationSlug).Scan(&pubID, &status, &visibility, &wordCount); err != nil {
		t.Fatalf("seeded publication not found: %v", err)
	}
	if status != "published" || visibility != "public" {
		t.Fatalf("expected published/public, got %s/%s", status, visibility)
	}
	if wordCount < 40000 {
		t.Fatalf("expected the real manuscript's word count, got %d", wordCount)
	}

	sections, err := LoadSections(ctx, pool, pubID)
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	if len(sections) < 700 {
		t.Fatalf("expected 700+ seeded sections, got %d", len(sections))
	}

	// Simulate a Producer's edit (the 4d12/5d12 fix, or any edit), then
	// re-run the seed as a second boot would. The edit must survive.
	p, err := LoadPublication(ctx, pool, pubID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	crew := insertTestUser(t, pool, "ew_seed_producer")
	grantRole(t, pool, loc, crew, "producer")
	if ok, err := CanEditPublication(ctx, pool, crew, p); err != nil || !ok {
		t.Fatalf("expected a Producer at the seed location to have edit authority, ok=%v err=%v", ok, err)
	}
	edited := mustSave(t, pool, crew, pubID, p.SourceMarkdown+"\n\nAN EDIT A PRODUCER MADE", p.CurrentRevisionID)
	if edited.RevisionNumber != 2 {
		t.Fatalf("expected the edit to be revision 2, got %d", edited.RevisionNumber)
	}

	if err := EnsureCanonicalSocioManuscript(ctx, pool); err != nil {
		t.Fatalf("second seed call: %v", err)
	}

	after, err := LoadPublication(ctx, pool, pubID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if after.CurrentRevisionID != edited.RevisionID {
		t.Fatal("re-running the seed must not revert a Producer's edit")
	}

	var count int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM ewrite_publications p
		JOIN ewrite_collections c ON c.id = p.collection_id
		WHERE c.location_id = $1 AND c.slug = $2 AND p.slug = $3
	`, loc, socioSeedRulesetSlug, socioSeedPublicationSlug).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one seeded publication, got %d", count)
	}
}

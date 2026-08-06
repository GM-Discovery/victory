package ewrite

// Kernel 78 store/authority/revision/visibility tests (spec 17.1-17.3,
// 17.6, 17.7 partial). Requires TEST_DATABASE_URL (internal/dbtest safety
// gate); fixtures are created per test and removed via t.Cleanup.

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

func testSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
}

func insertTestUser(t *testing.T, pool *pgxpool.Pool, prefix string) string {
	t.Helper()
	ctx := context.Background()
	handle := prefix + "_" + testSuffix(t)
	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name) VALUES ($1, $2) RETURNING id::text
	`, handle, prefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		// eWrite rows first (publications RESTRICT their collections), then
		// memberships, then the user.
		_, _ = pool.Exec(ctx, `DELETE FROM ewrite_publications WHERE created_by = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM ewrite_collections WHERE created_by = $1 AND NOT EXISTS (SELECT 1 FROM ewrite_collections ch WHERE ch.parent_id = ewrite_collections.id) AND NOT EXISTS (SELECT 1 FROM ewrite_publications p WHERE p.collection_id = ewrite_collections.id)`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM ewrite_collections WHERE created_by = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func grantRole(t *testing.T, pool *pgxpool.Pool, locationID, userID, role string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, $3::location_role, TRUE)
		ON CONFLICT (location_id, user_id, role) DO UPDATE SET active = TRUE
	`, locationID, userID, role); err != nil {
		t.Fatalf("grant role %q: %v", role, err)
	}
}

func amurrayLocation(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&id); err != nil {
		t.Fatalf("load amurray-family: %v", err)
	}
	return id
}

// insertTestLocation creates a disposable second location for
// cross-production denial tests.
func insertTestLocation(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	slug := "ewrite-test-loc-" + testSuffix(t)
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO locations (slug, name) VALUES ($1, 'eWrite Test Location') RETURNING id::text
	`, slug).Scan(&id); err != nil {
		t.Fatalf("insert test location: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, id)
	})
	return id
}

func mustCreateCollection(t *testing.T, pool *pgxpool.Pool, userID, locationID, parentID, kind, title string) *Collection {
	t.Helper()
	c, err := CreateCollection(context.Background(), pool, userID, locationID, parentID, kind, title, "", "production")
	if err != nil {
		t.Fatalf("create %s %q: %v", kind, title, err)
	}
	return c
}

func mustCreatePublication(t *testing.T, pool *pgxpool.Pool, userID, collectionID, title string) *Publication {
	t.Helper()
	p, err := CreatePublication(context.Background(), pool, userID, collectionID, title, "", "production")
	if err != nil {
		t.Fatalf("create publication %q: %v", title, err)
	}
	return p
}

func mustSave(t *testing.T, pool *pgxpool.Pool, userID, pubID, source, base string) *SaveResult {
	t.Helper()
	result, conflict, err := SavePublicationSource(context.Background(), pool, userID, pubID, source, base)
	if err != nil {
		t.Fatalf("save: %v (conflict %+v)", err, conflict)
	}
	return result
}

// --- 17.1 Hierarchy ---

func TestHierarchyKindRules(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_hier_crew")
	grantRole(t, pool, loc, crew, "crew")

	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Socio Test Ruleset")
	series := mustCreateCollection(t, pool, crew, loc, ruleset.ID, "series", "Core Rulebook")
	module := mustCreateCollection(t, pool, crew, loc, series.ID, "module", "Character Rules")

	if _, err := CreateCollection(ctx, pool, crew, loc, series.ID, "ruleset", "Bad Root", "", "production"); err == nil || err.Error() != "ruleset_must_be_root" {
		t.Fatalf("expected ruleset_must_be_root, got %v", err)
	}
	if _, err := CreateCollection(ctx, pool, crew, loc, module.ID, "series", "Bad Series", "", "production"); err == nil || err.Error() != "series_parent_must_be_ruleset" {
		t.Fatalf("expected series_parent_must_be_ruleset, got %v", err)
	}
	if _, err := CreateCollection(ctx, pool, crew, loc, "", "module", "Bad Module", "", "production"); err == nil || err.Error() != "module_parent_must_be_ruleset_or_series" {
		t.Fatalf("expected module_parent_must_be_ruleset_or_series, got %v", err)
	}
	// Omitted level, deliberate: module directly under ruleset.
	if _, err := CreateCollection(ctx, pool, crew, loc, ruleset.ID, "module", "Direct Module", "", "production"); err != nil {
		t.Fatalf("module under ruleset must be allowed: %v", err)
	}

	// Publications attach at any level.
	mustCreatePublication(t, pool, crew, module.ID, "Social Stances")
	mustCreatePublication(t, pool, crew, ruleset.ID, "Standalone Article")

	// Duplicate titles get deduped slugs, not errors.
	p1 := mustCreatePublication(t, pool, crew, module.ID, "Overview")
	p2 := mustCreatePublication(t, pool, crew, module.ID, "Overview")
	if p1.Slug == p2.Slug {
		t.Fatalf("expected deduped slugs, got %q twice", p1.Slug)
	}

	// Non-empty collection refuses deletion.
	if err := DeleteCollection(ctx, pool, crew, module.ID); err == nil || err.Error() != "collection_not_empty" {
		t.Fatalf("expected collection_not_empty, got %v", err)
	}
}

// --- 17.2 Authority ---

func TestAuthoringAuthority(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	otherLoc := insertTestLocation(t, pool)

	crew := insertTestUser(t, pool, "ew_auth_crew")
	crew2 := insertTestUser(t, pool, "ew_auth_crew2")
	director := insertTestUser(t, pool, "ew_auth_director")
	cast := insertTestUser(t, pool, "ew_auth_cast")
	audience := insertTestUser(t, pool, "ew_auth_audience")
	outsider := insertTestUser(t, pool, "ew_auth_outsider")
	grantRole(t, pool, loc, crew, "crew")
	grantRole(t, pool, loc, crew2, "crew")
	grantRole(t, pool, loc, director, "director")
	grantRole(t, pool, loc, cast, "cast")
	grantRole(t, pool, loc, audience, "audience")
	grantRole(t, pool, otherLoc, outsider, "crew")

	// Cast and Audience are denied authoring (spec 17.2).
	for _, denied := range []string{cast, audience} {
		if _, err := CreateCollection(ctx, pool, denied, loc, "", "ruleset", "Nope", "", "production"); err == nil || err.Error() != "not_authorized" {
			t.Fatalf("expected cast/audience authoring denial, got %v", err)
		}
	}
	// Crew outside scope denied: outsider is crew at otherLoc, not here.
	if _, err := CreateCollection(ctx, pool, outsider, loc, "", "ruleset", "Nope", "", "production"); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected cross-location authoring denial, got %v", err)
	}

	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Authority Ruleset")
	pub := mustCreatePublication(t, pool, crew, ruleset.ID, "Authority Pub")

	// Another crew member may not edit someone else's draft...
	if _, err := PatchPublication(ctx, pool, crew2, pub.ID, map[string]any{"summary": "x"}); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected crew2 edit denial, got %v", err)
	}
	// ...but a director may.
	if _, err := PatchPublication(ctx, pool, director, pub.ID, map[string]any{"summary": "директор edit"}); err != nil {
		t.Fatalf("director edit: %v", err)
	}

	// Named editor grant: edit works, publish still denied.
	var crew2Handle string
	if err := pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, crew2).Scan(&crew2Handle); err != nil {
		t.Fatalf("crew2 handle: %v", err)
	}
	if _, err := AddEditor(ctx, pool, director, pub.ID, crew2Handle, "edit"); err != nil {
		t.Fatalf("add editor: %v", err)
	}
	if _, err := PatchPublication(ctx, pool, crew2, pub.ID, map[string]any{"summary": "granted edit"}); err != nil {
		t.Fatalf("granted edit: %v", err)
	}
	mustSave(t, pool, crew2, pub.ID, "# Authority Pub\n\nbody", "")
	if _, err := PublishPublication(ctx, pool, crew2, pub.ID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected publish denial for edit-only grant, got %v", err)
	}
	if _, err := AddEditor(ctx, pool, director, pub.ID, crew2Handle, "publish"); err != nil {
		t.Fatalf("add publish grant: %v", err)
	}
	if _, err := PublishPublication(ctx, pool, crew2, pub.ID); err != nil {
		t.Fatalf("publish with grant: %v", err)
	}

	// Cast cannot save even with the publication id in hand (client
	// manipulation is not authority).
	if _, _, err := SavePublicationSource(ctx, pool, cast, pub.ID, "# hacked", ""); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected cast save denial, got %v", err)
	}
}

// --- 17.6 Revisions and conflict ---

func TestSaveConflictAndRevisions(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_rev_crew")
	director := insertTestUser(t, pool, "ew_rev_director")
	grantRole(t, pool, loc, crew, "crew")
	grantRole(t, pool, loc, director, "director")

	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Revision Ruleset")
	pub := mustCreatePublication(t, pool, crew, ruleset.ID, "Revision Pub")

	r1 := mustSave(t, pool, crew, pub.ID, "# One\n\nfirst", "")
	if r1.RevisionNumber != 1 {
		t.Fatalf("expected revision 1, got %d", r1.RevisionNumber)
	}
	r2 := mustSave(t, pool, crew, pub.ID, "# One\n\nsecond", r1.RevisionID)
	if r2.RevisionNumber != 2 {
		t.Fatalf("expected revision 2, got %d", r2.RevisionNumber)
	}

	// Stale base (User A still on r1 after director saved r2): 409, and
	// the newer content is untouched.
	_, conflict, err := SavePublicationSource(ctx, pool, director, pub.ID, "# One\n\nstale overwrite", r1.RevisionID)
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("expected revision conflict, got %v", err)
	}
	if conflict == nil || conflict.CurrentRevisionID != r2.RevisionID || conflict.CurrentRevisionNumber != 2 {
		t.Fatalf("conflict payload wrong: %+v", conflict)
	}
	p, err := LoadPublication(ctx, pool, pub.ID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if p.SourceMarkdown != "# One\n\nsecond" {
		t.Fatalf("conflict must not overwrite newer work; source is %q", p.SourceMarkdown)
	}

	// No-change save: defined as a no-op, no new revision.
	r3 := mustSave(t, pool, crew, pub.ID, "# One\n\nsecond", r2.RevisionID)
	if !r3.NoChange || r3.RevisionNumber != 2 {
		t.Fatalf("expected no-change save, got %+v", r3)
	}

	revs, err := ListRevisions(ctx, pool, pub.ID)
	if err != nil {
		t.Fatalf("list revisions: %v", err)
	}
	if len(revs) != 2 || revs[0].RevisionNumber != 2 || revs[1].RevisionNumber != 1 {
		t.Fatalf("expected append-forward revisions [2,1], got %+v", revs)
	}
	// Old source remains loadable (restore path).
	old, err := LoadRevision(ctx, pool, revs[1].ID)
	if err != nil {
		t.Fatalf("load revision: %v", err)
	}
	if old.SourceMarkdown != "# One\n\nfirst" {
		t.Fatalf("old revision source lost: %q", old.SourceMarkdown)
	}
}

// --- 17.7 Sections, anchors, aliases ---

func TestSectionsStableAcrossRenameWithAlias(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_sec_crew")
	grantRole(t, pool, loc, crew, "crew")

	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Sections Ruleset")
	pub := mustCreatePublication(t, pool, crew, ruleset.ID, "Sections Pub")

	r1 := mustSave(t, pool, crew, pub.ID, "# Doc\n\n## Defensive Stance\n\nbody\n\n### Taking Defensive Stance\n\nmore", "")
	sections, err := LoadSections(ctx, pool, pub.ID)
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	if len(sections) != 3 {
		t.Fatalf("expected 3 sections, got %+v", sections)
	}
	var stance Section
	for _, s := range sections {
		if s.Title == "Defensive Stance" {
			stance = s
		}
	}
	if stance.ID == "" || stance.Anchor != "defensive-stance" {
		t.Fatalf("stance section wrong: %+v", stance)
	}
	// Subsection parents to its section.
	for _, s := range sections {
		if s.Title == "Taking Defensive Stance" && s.ParentSectionID != stance.ID {
			t.Fatalf("expected subsection parent %s, got %+v", stance.ID, s)
		}
	}

	// Same title, new explicit anchor: row UUID survives, old anchor
	// becomes an alias.
	mustSave(t, pool, crew, pub.ID, "# Doc\n\n## Defensive Stance {#stance-defensive}\n\nbody\n\n### Taking Defensive Stance\n\nmore", r1.RevisionID)
	sections2, err := LoadSections(ctx, pool, pub.ID)
	if err != nil {
		t.Fatalf("sections2: %v", err)
	}
	var stance2 Section
	for _, s := range sections2 {
		if s.Title == "Defensive Stance" {
			stance2 = s
		}
	}
	if stance2.ID != stance.ID {
		t.Fatalf("section identity lost on anchor change: %s -> %s", stance.ID, stance2.ID)
	}
	if stance2.Anchor != "stance-defensive" || !stance2.AnchorExplicit {
		t.Fatalf("anchor not updated: %+v", stance2)
	}
	resolved, found, err := ResolveAnchor(ctx, pool, pub.ID, "defensive-stance")
	if err != nil || !found || resolved != "stance-defensive" {
		t.Fatalf("old anchor must alias to new: %q %v %v", resolved, found, err)
	}
}

// --- 17.3 Visibility (reader side) ---

func TestReadVisibilityAndSearch(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	otherLoc := insertTestLocation(t, pool)

	crew := insertTestUser(t, pool, "ew_vis_crew")
	member := insertTestUser(t, pool, "ew_vis_member")
	outsider := insertTestUser(t, pool, "ew_vis_outsider")
	grantRole(t, pool, loc, crew, "crew")
	grantRole(t, pool, loc, member, "audience")
	grantRole(t, pool, otherLoc, outsider, "crew")

	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Visibility Ruleset")

	prodPub := mustCreatePublication(t, pool, crew, ruleset.ID, "Production Rules Zebrawood")
	mustSave(t, pool, crew, prodPub.ID, "# Production Rules Zebrawood\n\nthe secret zebrawood mechanics", "")
	if _, err := PublishPublication(ctx, pool, crew, prodPub.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}

	authPub := mustCreatePublication(t, pool, crew, ruleset.ID, "Open Rules Quillfeather")
	if _, err := PatchPublication(ctx, pool, crew, authPub.ID, map[string]any{"visibility": "authenticated"}); err != nil {
		t.Fatalf("patch visibility: %v", err)
	}
	mustSave(t, pool, crew, authPub.ID, "# Open Rules Quillfeather\n\npublic quillfeather lore", "")
	if _, err := PublishPublication(ctx, pool, crew, authPub.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}

	draftPub := mustCreatePublication(t, pool, crew, ruleset.ID, "Draft Rules Cinderglass")
	mustSave(t, pool, crew, draftPub.ID, "# Draft Rules Cinderglass\n\nunfinished cinderglass notes", "")

	check := func(userID, pubID string, want bool, label string) {
		t.Helper()
		p, err := LoadPublication(ctx, pool, pubID)
		if err != nil {
			t.Fatalf("%s load: %v", label, err)
		}
		ok, err := CanReadPublication(ctx, pool, userID, p)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if ok != want {
			t.Fatalf("%s: expected %v, got %v", label, want, ok)
		}
	}

	check(member, prodPub.ID, true, "member reads production pub")
	check(outsider, prodPub.ID, false, "Production B crew denied Production A pub")
	check(outsider, authPub.ID, true, "authenticated pub readable across productions")
	check(member, draftPub.ID, false, "audience denied draft")
	check("", authPub.ID, false, "anonymous denied (public deferral)")

	// Search respects the same predicate; drafts are structurally excluded.
	memberResults, err := SearchPublications(ctx, pool, member, false, "zebrawood", "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(memberResults) != 1 || memberResults[0].PublicationID != prodPub.ID {
		t.Fatalf("member should find production pub: %+v", memberResults)
	}
	if !strings.Contains(memberResults[0].Snippet, "<mark>") {
		t.Fatalf("expected highlighted snippet, got %q", memberResults[0].Snippet)
	}
	outsiderResults, err := SearchPublications(ctx, pool, outsider, false, "zebrawood", "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(outsiderResults) != 0 {
		t.Fatalf("outsider must not find production pub: %+v", outsiderResults)
	}
	draftResults, err := SearchPublications(ctx, pool, member, false, "cinderglass", "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(draftResults) != 0 {
		t.Fatalf("draft must never be discoverable through search: %+v", draftResults)
	}
}

// --- 17.8 Export round trip ---

func TestExportRoundTrip(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_exp_crew")
	grantRole(t, pool, loc, crew, "crew")

	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Export Ruleset")
	series := mustCreateCollection(t, pool, crew, loc, ruleset.ID, "series", "Export Series")
	pub := mustCreatePublication(t, pool, crew, series.ID, "Export Pub")
	source := "# Export Pub {#export-pub}\n\n## Section One\n\nSocio\\- content with [link](#section-one).\n"
	mustSave(t, pool, crew, pub.ID, source, "")

	p, err := LoadPublication(ctx, pool, pub.ID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	data, err := BuildPublicationExport(ctx, pool, p, true)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip: %v", err)
	}
	var gotSource, gotMeta []byte
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		if strings.HasSuffix(f.Name, ".md") {
			gotSource = b
		}
		if f.Name == "metadata.json" {
			gotMeta = b
		}
	}
	if string(gotSource) != source {
		t.Fatalf("export source not byte-preserved:\nwant %q\ngot  %q", source, gotSource)
	}
	meta := string(gotMeta)
	for _, needle := range []string{"Export Ruleset", "Export Series", "export-pub", "Section One", `"status": "draft"`} {
		if !strings.Contains(meta, needle) {
			t.Fatalf("metadata missing %q:\n%s", needle, meta)
		}
	}
}

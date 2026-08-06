package ewrite

// Kernel 79 Goal E: hierarchical export. Proves a Ruleset export reaches a
// Publication two levels down (Series -> Module), that a draft publication
// and a Production-private publication the requester can't read are both
// silently omitted (never partially included, spec 11.5), that the zip's
// internal paths and manifest agree, and that checksums are real (recomputed
// and compared, not just present).

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"

	"victory/backend/internal/dbtest"
)

func TestBuildCollectionExportHierarchyAndVisibility(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_export_crew")
	// member is Crew+'s "ordinary reader" foil: an active member of the
	// same location, but neither the draft's creator nor holding any
	// editor grant on it -- the actual boundary spec 11.5 cares about
	// ("drafts require editor authority"), not merely "some role at this
	// location." Using the creator (crew) for this assertion would prove
	// nothing, since a creator always has edit authority over their own
	// work regardless of role.
	member := insertTestUser(t, pool, "ew_export_member")
	outsider := insertTestUser(t, pool, "ew_export_outsider")
	grantRole(t, pool, loc, crew, "crew")
	grantRole(t, pool, loc, member, "audience")
	otherLoc := insertTestLocation(t, pool)
	grantRole(t, pool, otherLoc, outsider, "crew")

	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Export Ruleset")
	series := mustCreateCollection(t, pool, crew, loc, ruleset.ID, "series", "Export Series")
	module := mustCreateCollection(t, pool, crew, loc, series.ID, "module", "Export Module")

	deepPub := mustCreatePublication(t, pool, crew, module.ID, "Deep Export Publication")
	mustSave(t, pool, crew, deepPub.ID, "# Deep Export Publication\n\n## Buried Rule\n\ntext", "")
	if _, err := PublishPublication(ctx, pool, crew, deepPub.ID); err != nil {
		t.Fatalf("publish deep pub: %v", err)
	}

	draftPub := mustCreatePublication(t, pool, crew, series.ID, "Still A Draft")
	mustSave(t, pool, crew, draftPub.ID, "# Still A Draft\n\nunfinished", "")
	// never published

	prodPub := mustCreatePublication(t, pool, crew, ruleset.ID, "Production Private Publication")
	mustSave(t, pool, crew, prodPub.ID, "# Production Private Publication\n\nsecret", "")
	if _, err := PatchPublication(ctx, pool, crew, prodPub.ID, map[string]any{"visibility": "production"}); err != nil {
		t.Fatalf("patch visibility: %v", err)
	}
	if _, err := PublishPublication(ctx, pool, crew, prodPub.ID); err != nil {
		t.Fatalf("publish production pub: %v", err)
	}

	// member (an ordinary audience-role reader at loc, not the draft's
	// author) exports: sees the deep pub and the production-visibility pub
	// (any active member reads production content, any role), never the
	// draft (no editor authority).
	data, manifest, err := BuildCollectionExport(ctx, pool, member, ruleset.ID)
	if err != nil {
		t.Fatalf("build export (member): %v", err)
	}
	if manifest.RootTitle != "Export Ruleset" {
		t.Fatalf("expected root title, got %+v", manifest)
	}
	includedIDs := map[string]bool{}
	for _, p := range manifest.Publications {
		includedIDs[p.ID] = true
	}
	if !includedIDs[deepPub.ID] {
		t.Fatalf("expected deep Module publication included for member, got %+v", manifest.Publications)
	}
	if !includedIDs[prodPub.ID] {
		t.Fatalf("expected production-visibility publication included for its own member, got %+v", manifest.Publications)
	}
	if includedIDs[draftPub.ID] {
		t.Fatalf("draft publication must never be included, got %+v", manifest.Publications)
	}
	found := false
	for _, omitted := range manifest.OmittedPublications {
		if omitted == draftPub.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected draft publication named in omitted_publications, got %+v", manifest.OmittedPublications)
	}

	// Verify the zip actually contains the deep publication at its
	// hierarchy-derived path, and that its checksum is real.
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	var deepRef *CollectionExportPubRef
	for i := range manifest.Publications {
		if manifest.Publications[i].ID == deepPub.ID {
			deepRef = &manifest.Publications[i]
		}
	}
	if deepRef == nil {
		t.Fatal("deep pub ref missing from manifest")
	}
	wantPath := ruleset.Slug + "/" + series.Slug + "/" + module.Slug + "/" + deepPub.Slug + ".md"
	if deepRef.MarkdownPath != wantPath {
		t.Fatalf("expected hierarchy-derived path %q, got %q", wantPath, deepRef.MarkdownPath)
	}
	var zf *zip.File
	for _, f := range zr.File {
		if f.Name == deepRef.MarkdownPath {
			zf = f
		}
	}
	if zf == nil {
		t.Fatalf("markdown file %q not found in zip", deepRef.MarkdownPath)
	}
	rc, err := zf.Open()
	if err != nil {
		t.Fatalf("open zip entry: %v", err)
	}
	content, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("read zip entry: %v", err)
	}
	sum := sha256.Sum256(content)
	gotChecksum := hex.EncodeToString(sum[:])
	wantChecksum, ok := manifest.Checksums[deepRef.MarkdownPath]
	if !ok || wantChecksum != gotChecksum {
		t.Fatalf("checksum mismatch: manifest=%q recomputed=%q", wantChecksum, gotChecksum)
	}

	if len(content) == 0 || content[0] != '#' {
		t.Fatalf("expected raw markdown starting with '#', got %q", string(content[:min(20, len(content))]))
	}

	// outsider (Crew+ at an unrelated location, no membership at loc) gets
	// an export with nothing readable in it at all -- not an error, just
	// empty, mirroring how Library browsing degrades for the same actor.
	_, outsiderManifest, err := BuildCollectionExport(ctx, pool, outsider, ruleset.ID)
	if err != nil {
		t.Fatalf("build export (outsider): %v", err)
	}
	if len(outsiderManifest.Publications) != 0 {
		t.Fatalf("expected outsider to see zero publications, got %+v", outsiderManifest.Publications)
	}
	if len(outsiderManifest.OmittedPublications) != 3 {
		t.Fatalf("expected all 3 publications omitted for outsider, got %+v", outsiderManifest.OmittedPublications)
	}
}

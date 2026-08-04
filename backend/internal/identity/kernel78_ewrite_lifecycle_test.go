package identity

// Kernel 78: account deletion and export integrate with eWrite (spec 12.3,
// 12.4, 17.8, 17.9). Follows account_deletion_test.go's fixture idioms.

import (
	"archive/zip"
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func readZipForTest(t *testing.T, zipPath string) ([]string, map[string]string) {
	t.Helper()
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open export zip: %v", err)
	}
	defer zr.Close()
	var names []string
	contents := map[string]string{}
	for _, f := range zr.File {
		names = append(names, f.Name)
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		contents[f.Name] = string(b)
	}
	return names, contents
}

func TestAccountDeletionHandlesEwriteAuthorship(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	handle := "del_ewrite_" + time.Now().UTC().Format("150405.000000")
	userID := insertAccountTestUser(t, pool, handle, "Delete Me Author")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	locationID := resolveAccountTestLocationID(t, pool, "amurray-family")
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'crew', TRUE)
	`, locationID, userID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}

	var collectionID, publishedID, draftID string
	suffix := time.Now().UTC().Format("150405.000000")
	if err := pool.QueryRow(ctx, `
		INSERT INTO ewrite_collections (location_id, kind, title, slug, created_by)
		VALUES ($1, 'ruleset', 'Lifecycle Ruleset', $2, $3)
		RETURNING id::text
	`, locationID, "lifecycle-ruleset-"+suffix, userID).Scan(&collectionID); err != nil {
		t.Fatalf("insert collection: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO ewrite_publications (collection_id, location_id, title, slug, source_markdown, status, created_by, updated_by, published_at)
		VALUES ($1, $2, 'Published Work', 'published-work', '# Published Work', 'published', $3, $3, NOW())
		RETURNING id::text
	`, collectionID, locationID, userID).Scan(&publishedID); err != nil {
		t.Fatalf("insert published: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO ewrite_publications (collection_id, location_id, title, slug, source_markdown, status, created_by, updated_by)
		VALUES ($1, $2, 'Sole Draft', 'sole-draft', '# Sole Draft', 'draft', $3, $3)
		RETURNING id::text
	`, collectionID, locationID, userID).Scan(&draftID); err != nil {
		t.Fatalf("insert draft: %v", err)
	}
	for _, pubID := range []string{publishedID, draftID} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO ewrite_revisions (publication_id, revision_number, source_markdown, created_by)
			VALUES ($1, 1, 'rev source', $2)
		`, pubID, userID); err != nil {
			t.Fatalf("insert revision: %v", err)
		}
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM ewrite_publications WHERE collection_id = $1`, collectionID)
		_, _ = pool.Exec(ctx, `DELETE FROM ewrite_collections WHERE id = $1`, collectionID)
	})

	// Deletion plan counts the eWritings.
	plan, err := BuildDeletionPlan(ctx, pool, userID)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.PrivateRecordCounts["ewritings"] != 2 {
		t.Fatalf("expected 2 ewritings in plan, got %+v", plan.PrivateRecordCounts)
	}

	if _, _, err := executeDeletion(ctx, pool, userID, t.TempDir()); err != nil {
		t.Fatalf("executeDeletion: %v", err)
	}

	// Published work survives, reassigned to the tombstone; revision
	// authorship anonymized; sole-owned draft gone; no ownerless rows.
	var createdBy string
	if err := pool.QueryRow(ctx, `SELECT COALESCE(created_by::text, 'NULL') FROM ewrite_publications WHERE id = $1`, publishedID).Scan(&createdBy); err != nil {
		t.Fatalf("published work must survive deletion: %v", err)
	}
	if createdBy != tombstoneUserID {
		t.Fatalf("expected published work reassigned to tombstone, got %q", createdBy)
	}
	var draftCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM ewrite_publications WHERE id = $1`, draftID).Scan(&draftCount); err != nil {
		t.Fatalf("check draft: %v", err)
	}
	if draftCount != 0 {
		t.Fatal("sole-owned draft must be deleted with its author")
	}
	var revAuthor string
	if err := pool.QueryRow(ctx, `SELECT COALESCE(created_by::text, 'NULL') FROM ewrite_revisions WHERE publication_id = $1`, publishedID).Scan(&revAuthor); err != nil {
		t.Fatalf("check revision: %v", err)
	}
	if revAuthor != tombstoneUserID {
		t.Fatalf("expected revision authorship anonymized to tombstone, got %q", revAuthor)
	}
	var orphaned int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM ewrite_collections WHERE id = $1 AND created_by IS NULL`, collectionID).Scan(&orphaned); err != nil {
		t.Fatalf("check collection: %v", err)
	}
	if orphaned != 0 {
		t.Fatal("collection must be tombstone-reassigned, not ownerless")
	}
}

func TestAccountExportIncludesEwrite(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	handle := "exp_ewrite_" + time.Now().UTC().Format("150405.000000")
	userID := insertAccountTestUser(t, pool, handle, "Export Me Author")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	locationID := resolveAccountTestLocationID(t, pool, "amurray-family")

	var collectionID, pubID string
	suffix := time.Now().UTC().Format("150405.000000")
	if err := pool.QueryRow(ctx, `
		INSERT INTO ewrite_collections (location_id, kind, title, slug, created_by)
		VALUES ($1, 'ruleset', 'Export Ruleset', $2, $3)
		RETURNING id::text
	`, locationID, "export-ruleset-"+suffix, userID).Scan(&collectionID); err != nil {
		t.Fatalf("insert collection: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO ewrite_publications (collection_id, location_id, title, slug, summary, source_markdown, status, created_by, updated_by)
		VALUES ($1, $2, 'My Export Work', 'my-export-work', 'summary here', '# My Export Work'||chr(10)||'distinctive-marker-quillfeather', 'draft', $3, $3)
		RETURNING id::text
	`, collectionID, locationID, userID).Scan(&pubID); err != nil {
		t.Fatalf("insert publication: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO ewrite_revisions (publication_id, revision_number, source_markdown, created_by)
		VALUES ($1, 1, 'rev source', $2)
	`, pubID, userID); err != nil {
		t.Fatalf("insert revision: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM ewrite_publications WHERE collection_id = $1`, collectionID)
		_, _ = pool.Exec(ctx, `DELETE FROM ewrite_collections WHERE id = $1`, collectionID)
	})

	zipPath, size, err := buildExportArchive(ctx, pool, userID, "test-job-ewrite-"+suffix, t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("buildExportArchive: %v", err)
	}
	if size <= 0 {
		t.Fatal("expected non-empty export")
	}
	names, contents := readZipForTest(t, zipPath)

	var mdName string
	for _, n := range names {
		if strings.HasPrefix(n, "ewrite/") && strings.HasSuffix(n, "/publication.md") {
			mdName = n
		}
	}
	if mdName == "" {
		t.Fatalf("expected an ewrite/*/publication.md in export, got %v", names)
	}
	if !strings.Contains(contents[mdName], "distinctive-marker-quillfeather") {
		t.Fatalf("export markdown must be the source: %q", contents[mdName])
	}
	if !strings.Contains(contents["manifest.json"], `"format_version": 2`) {
		t.Fatalf("expected format_version 2 in manifest: %s", contents["manifest.json"])
	}
	if !strings.Contains(contents["manifest.json"], `"ewrite_publications": 1`) {
		t.Fatalf("expected ewrite count in manifest: %s", contents["manifest.json"])
	}
}

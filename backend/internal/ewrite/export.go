package ewrite

// Per-publication export (kernel Goal F / spec 12.1): a zip containing the
// exact source Markdown (byte-preserved -- what import accepted is what
// export returns, spec 7.3) plus a metadata manifest with the hierarchy
// path, section map, and link manifest. Built entirely in memory; nothing
// under storage/ or exports/ (the backup exclusion trap).

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ExportMetadata is metadata.json inside the export zip.
type ExportMetadata struct {
	Title           string    `json:"title"`
	Slug            string    `json:"slug"`
	Summary         string    `json:"summary"`
	HierarchyPath   []string  `json:"hierarchy_path"`
	Status          string    `json:"status,omitempty"`
	Visibility      string    `json:"visibility"`
	WordCount       int       `json:"word_count"`
	PublishedAt     any       `json:"published_at,omitempty"`
	ExportedAt      time.Time `json:"exported_at"`
	Sections        []Section `json:"sections"`
	InternalLinks   []string  `json:"internal_links"`
	ExternalLinks   []string  `json:"external_links"`
	ImageReferences []string  `json:"image_references"`
	FormatVersion   int       `json:"format_version"`
}

// collectionPath walks parents up to the root, returning titles from root
// to the publication's own collection.
func collectionPath(ctx context.Context, pool queryer, collectionID string) ([]string, error) {
	var path []string
	id := collectionID
	for i := 0; id != "" && i < 10; i++ {
		var title, parent string
		if err := pool.QueryRow(ctx, `
			SELECT title, COALESCE(parent_id::text, '') FROM ewrite_collections WHERE id = $1
		`, id).Scan(&title, &parent); err != nil {
			return nil, err
		}
		path = append([]string{title}, path...)
		id = parent
	}
	return path, nil
}

// collectionSlugPath is collectionPath's filesystem-safe twin: slugs
// instead of titles, for building the zip's internal directory layout.
func collectionSlugPath(ctx context.Context, pool queryer, collectionID string) ([]string, error) {
	var path []string
	id := collectionID
	for i := 0; id != "" && i < 10; i++ {
		var slug, parent string
		if err := pool.QueryRow(ctx, `
			SELECT slug, COALESCE(parent_id::text, '') FROM ewrite_collections WHERE id = $1
		`, id).Scan(&slug, &parent); err != nil {
			return nil, err
		}
		path = append([]string{slug}, path...)
		id = parent
	}
	return path, nil
}

// CollectionExportManifest is manifest.json at the root of a Module/Series/
// Ruleset export package (spec 11.2-11.3). Hierarchy is reconstructable
// from Collections' parent_id chain plus each Publication's CollectionID
// and Path, without guessing (spec 11.4).
type CollectionExportManifest struct {
	FormatVersion       int                      `json:"format_version"`
	RootCollectionID    string                   `json:"root_collection_id"`
	RootKind            string                   `json:"root_kind"`
	RootTitle           string                   `json:"root_title"`
	RootSlug            string                   `json:"root_slug"`
	ExportedAt          time.Time                `json:"exported_at"`
	Collections         []Collection             `json:"collections"`
	Publications        []CollectionExportPubRef `json:"publications"`
	Directories         []CollectionExportDirRef `json:"directories"`
	OmittedPublications []string                 `json:"omitted_publications,omitempty"`
	Checksums           map[string]string        `json:"checksums"`
}

// CollectionExportPubRef locates one included publication's files inside
// the zip and its place in the hierarchy.
type CollectionExportPubRef struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Slug          string   `json:"slug"`
	CollectionID  string   `json:"collection_id"`
	HierarchyPath []string `json:"hierarchy_path"`
	MarkdownPath  string   `json:"markdown_path"`
	MetadataPath  string   `json:"metadata_path"`
}

// CollectionExportDirRef embeds one directory's visibility-safe entries
// (spec 11.3 "directory entries") -- resolved through the same
// ListDirectoryEntries used everywhere else, so an export can't leak a
// hidden target that browsing already protects.
type CollectionExportDirRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
	Path  string `json:"path"`
}

// BuildCollectionExport assembles a Module/Series/Ruleset export package
// (spec 11): every publication the requesting user can read, at every
// depth of the subtree rooted at collectionID, preserving hierarchy via
// each publication's own collection-slug path inside the zip, plus
// directories and a manifest with checksums. A publication the requester
// cannot read (spec 11.5: Production-private content must not appear in
// another Production's export; drafts require editor authority) is
// silently omitted from the zip and named in the manifest's
// omitted_publications, never included partially.
//
// No image bytes are embedded -- BuildPublicationExport (the single-
// publication export, Kernel 78) never has either; only ImageReferences
// are listed in each publication's own metadata.json. Same precedent, not
// a new gap.
func BuildCollectionExport(ctx context.Context, pool *pgxpool.Pool, userID, rootCollectionID string) ([]byte, *CollectionExportManifest, error) {
	var root Collection
	if err := pool.QueryRow(ctx, `
		SELECT id::text, location_id::text, COALESCE(parent_id::text, ''), kind, title, slug, summary, sort_order, visibility, created_at, updated_at
		FROM ewrite_collections WHERE id = $1
	`, rootCollectionID).Scan(&root.ID, &root.LocationID, &root.ParentID, &root.Kind, &root.Title, &root.Slug, &root.Summary, &root.SortOrder, &root.Visibility, &root.CreatedAt, &root.UpdatedAt); err != nil {
		return nil, nil, errors.New("collection_not_found")
	}

	subtreeRows, err := pool.Query(ctx, collectionSubtreeCTEAt(1)+`
		SELECT c.id::text, c.location_id::text, COALESCE(c.parent_id::text, ''), c.kind, c.title, c.slug, c.summary, c.sort_order, c.visibility, c.created_at, c.updated_at
		FROM ewrite_collections c WHERE c.id IN (SELECT id FROM sub)
		ORDER BY c.sort_order, c.title
	`, rootCollectionID)
	if err != nil {
		return nil, nil, err
	}
	var collections []Collection
	subtreeIDs := []string{}
	for subtreeRows.Next() {
		var c Collection
		if err := subtreeRows.Scan(&c.ID, &c.LocationID, &c.ParentID, &c.Kind, &c.Title, &c.Slug, &c.Summary, &c.SortOrder, &c.Visibility, &c.CreatedAt, &c.UpdatedAt); err != nil {
			subtreeRows.Close()
			return nil, nil, err
		}
		collections = append(collections, c)
		subtreeIDs = append(subtreeIDs, c.ID)
	}
	subtreeRows.Close()
	if err := subtreeRows.Err(); err != nil {
		return nil, nil, err
	}

	pubRows, err := pool.Query(ctx, `
		SELECT id::text FROM ewrite_publications WHERE collection_id = ANY($1)
	`, subtreeIDs)
	if err != nil {
		return nil, nil, err
	}
	var pubIDs []string
	for pubRows.Next() {
		var id string
		if err := pubRows.Scan(&id); err != nil {
			pubRows.Close()
			return nil, nil, err
		}
		pubIDs = append(pubIDs, id)
	}
	pubRows.Close()
	if err := pubRows.Err(); err != nil {
		return nil, nil, err
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	checksums := map[string]string{}
	manifest := &CollectionExportManifest{
		FormatVersion:       1,
		RootCollectionID:    root.ID,
		RootKind:            root.Kind,
		RootTitle:           root.Title,
		RootSlug:            root.Slug,
		ExportedAt:          time.Now().UTC(),
		Collections:         collections,
		Publications:        []CollectionExportPubRef{},
		Directories:         []CollectionExportDirRef{},
		OmittedPublications: []string{},
		Checksums:           checksums,
	}

	writeFile := func(path string, data []byte) error {
		f, err := zw.Create(path)
		if err != nil {
			return err
		}
		if _, err := f.Write(data); err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		checksums[path] = hex.EncodeToString(sum[:])
		return nil
	}

	for _, pubID := range pubIDs {
		p, err := LoadPublication(ctx, pool, pubID)
		if err != nil {
			return nil, nil, err
		}
		ok, err := CanReadPublication(ctx, pool, userID, p)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			manifest.OmittedPublications = append(manifest.OmittedPublications, p.ID)
			continue
		}
		canEdit, err := CanEditPublication(ctx, pool, userID, p)
		if err != nil {
			return nil, nil, err
		}

		slugPath, err := collectionSlugPath(ctx, pool, p.CollectionID)
		if err != nil {
			return nil, nil, err
		}
		res, err := Render(p.SourceMarkdown)
		if err != nil {
			return nil, nil, err
		}
		sections, err := LoadSections(ctx, pool, p.ID)
		if err != nil {
			return nil, nil, err
		}
		meta := ExportMetadata{
			Title: p.Title, Slug: p.Slug, Summary: p.Summary,
			HierarchyPath: slugPath, Visibility: p.Visibility, WordCount: p.WordCount,
			ExportedAt: manifest.ExportedAt, Sections: sections,
			InternalLinks: res.InternalLinks, ExternalLinks: res.ExternalLinks, ImageReferences: res.ImageRefs,
			FormatVersion: 1,
		}
		if canEdit {
			meta.Status = p.Status
		}
		if p.PublishedAt != nil {
			meta.PublishedAt = p.PublishedAt
		}
		if meta.InternalLinks == nil {
			meta.InternalLinks = []string{}
		}
		if meta.ExternalLinks == nil {
			meta.ExternalLinks = []string{}
		}
		if meta.ImageReferences == nil {
			meta.ImageReferences = []string{}
		}

		base := strings.Join(append(append([]string{}, slugPath...), p.Slug), "/")
		mdPath := base + ".md"
		metaPath := base + ".metadata.json"
		metaJSON, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			return nil, nil, err
		}
		if err := writeFile(mdPath, []byte(p.SourceMarkdown)); err != nil {
			return nil, nil, err
		}
		if err := writeFile(metaPath, metaJSON); err != nil {
			return nil, nil, err
		}
		manifest.Publications = append(manifest.Publications, CollectionExportPubRef{
			ID: p.ID, Title: p.Title, Slug: p.Slug, CollectionID: p.CollectionID,
			HierarchyPath: slugPath, MarkdownPath: mdPath, MetadataPath: metaPath,
		})
	}

	dirs, err := ListDirectoriesForCollection(ctx, pool, rootCollectionID)
	if err != nil {
		return nil, nil, err
	}
	for _, d := range dirs {
		entries, err := ListDirectoryEntries(ctx, pool, userID, d.ID, "", "")
		if err != nil {
			return nil, nil, err
		}
		dirJSON, err := json.MarshalIndent(map[string]any{"directory": d, "entries": entries}, "", "  ")
		if err != nil {
			return nil, nil, err
		}
		dirPath := "directories/" + d.Slug + ".json"
		if err := writeFile(dirPath, dirJSON); err != nil {
			return nil, nil, err
		}
		manifest.Directories = append(manifest.Directories, CollectionExportDirRef{ID: d.ID, Title: d.Title, Slug: d.Slug, Path: dirPath})
	}

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	mf, err := zw.Create("manifest.json")
	if err != nil {
		return nil, nil, err
	}
	if _, err := mf.Write(manifestJSON); err != nil {
		return nil, nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), manifest, nil
}

// HandleLibraryCollectionExport serves GET
// /api/library/collections/{collection_id}/export.
func HandleLibraryCollectionExport(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		id := strings.TrimSpace(r.PathValue("collection_id"))
		data, manifest, err := BuildCollectionExport(ctx, pool, userID, id)
		if err != nil {
			writeError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-export.zip"`, manifest.RootSlug))
		w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")
		_, _ = w.Write(data)
	}
}

// BuildPublicationExport assembles the zip. includeStatus controls whether
// draft/published state is included (spec 12.1 "publication status where
// authorized" -- readers exporting a published work don't need to see
// archival state machinery, editors do).
func BuildPublicationExport(ctx context.Context, pool *pgxpool.Pool, p *Publication, includeStatus bool) ([]byte, error) {
	res, err := Render(p.SourceMarkdown)
	if err != nil {
		return nil, err
	}
	sections, err := LoadSections(ctx, pool, p.ID)
	if err != nil {
		return nil, err
	}
	path, err := collectionPath(ctx, pool, p.CollectionID)
	if err != nil {
		return nil, err
	}

	meta := ExportMetadata{
		Title:           p.Title,
		Slug:            p.Slug,
		Summary:         p.Summary,
		HierarchyPath:   path,
		Visibility:      p.Visibility,
		WordCount:       p.WordCount,
		ExportedAt:      time.Now().UTC(),
		Sections:        sections,
		InternalLinks:   res.InternalLinks,
		ExternalLinks:   res.ExternalLinks,
		ImageReferences: res.ImageRefs,
		FormatVersion:   1,
	}
	if includeStatus {
		meta.Status = p.Status
	}
	if p.PublishedAt != nil {
		meta.PublishedAt = p.PublishedAt
	}
	if meta.InternalLinks == nil {
		meta.InternalLinks = []string{}
	}
	if meta.ExternalLinks == nil {
		meta.ExternalLinks = []string{}
	}
	if meta.ImageReferences == nil {
		meta.ImageReferences = []string{}
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	mdFile, err := zw.Create(p.Slug + ".md")
	if err != nil {
		return nil, err
	}
	if _, err := mdFile.Write([]byte(p.SourceMarkdown)); err != nil {
		return nil, err
	}

	metaFile, err := zw.Create("metadata.json")
	if err != nil {
		return nil, err
	}
	enc := json.NewEncoder(metaFile)
	enc.SetIndent("", "  ")
	if err := enc.Encode(meta); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// HandleLibraryExport serves GET /api/library/publications/{publication_id}/export.
// Readable-by-caller is the gate; editors additionally see status.
func HandleLibraryExport(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		id := strings.TrimSpace(r.PathValue("publication_id"))
		p, err := LoadPublication(ctx, pool, id)
		if err != nil {
			writeError(w, err)
			return
		}
		if ok, err := CanReadPublication(ctx, pool, userID, p); err != nil {
			writeError(w, err)
			return
		} else if !ok {
			writeError(w, errors.New("publication_not_found"))
			return
		}
		canEdit, err := CanEditPublication(ctx, pool, userID, p)
		if err != nil {
			writeError(w, err)
			return
		}
		data, err := BuildPublicationExport(ctx, pool, p, canEdit)
		if err != nil {
			writeError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-export.zip"`, p.Slug))
		w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")
		_, _ = w.Write(data)
	}
}

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

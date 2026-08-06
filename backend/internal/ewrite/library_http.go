package ewrite

// Library reading surface (/api/library/*). Published content only; every
// payload is filtered by the caller's effective visibility. Drafts and
// archived works are structurally excluded by the status predicate -- they
// cannot leak through here regardless of any authority bug above this
// layer (visibility filtering AND invocation authorization, per
// operator-notes.md).
//
// 'public' visibility is served to authenticated readers only in Kernel 78
// (recorded deferral of the first anonymous content API).

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

// visiblePublicationsPredicate is the shared WHERE fragment for reader
// queries. $userID is injected as the placeholder named in the caller's
// argument list. A publication is readable when published AND:
//   - visibility public/authenticated (any signed-in reader), or
//   - visibility production and the reader holds an ACTIVE membership at
//     the publication's location (any role including audience).
//
// Operator short-circuits are applied by callers where relevant.
const visiblePublicationsClause = `
	p.status = 'published'
	AND (
		p.visibility IN ('public', 'authenticated')
		OR (
			p.visibility = 'production'
			AND EXISTS (
				SELECT 1 FROM location_memberships lm
				WHERE lm.location_id = p.location_id
				  AND lm.user_id = $1
				  AND lm.active = TRUE
			)
		)
		OR $2::boolean
	)`

func isOperatorFlag(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
	return access.IsOperatorUser(ctx, pool, userID)
}

// LibraryTree is the Library bootstrap payload: collections that contain
// at least one readable publication (plus their ancestors), and the
// readable publications themselves (metadata only).
type LibraryTree struct {
	Collections  []Collection  `json:"collections"`
	Publications []Publication `json:"publications"`
}

// HandleLibraryTree serves GET /api/library/tree.
func HandleLibraryTree(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		operator, err := isOperatorFlag(ctx, pool, userID)
		if err != nil {
			writeError(w, err)
			return
		}

		tree := LibraryTree{Collections: []Collection{}, Publications: []Publication{}}

		rows, err := pool.Query(ctx, `
			SELECT p.id::text, p.collection_id::text, p.location_id::text, p.title, p.slug, p.summary,
			       p.word_count, p.status, p.visibility, p.published_at, p.created_at, p.updated_at
			FROM ewrite_publications p
			WHERE `+visiblePublicationsClause+`
			ORDER BY p.sort_order, p.published_at`, userID, operator)
		if err != nil {
			writeError(w, err)
			return
		}
		collectionIDs := map[string]bool{}
		for rows.Next() {
			var p Publication
			if err := rows.Scan(&p.ID, &p.CollectionID, &p.LocationID, &p.Title, &p.Slug, &p.Summary,
				&p.WordCount, &p.Status, &p.Visibility, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
				rows.Close()
				writeError(w, err)
				return
			}
			tree.Publications = append(tree.Publications, p)
			collectionIDs[p.CollectionID] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			writeError(w, err)
			return
		}

		if len(collectionIDs) > 0 {
			ids := make([]string, 0, len(collectionIDs))
			for id := range collectionIDs {
				ids = append(ids, id)
			}
			// Ancestors via recursive CTE so Ruleset/Series shells render
			// even when only a deep Module holds readable content.
			crows, err := pool.Query(ctx, `
				WITH RECURSIVE reachable AS (
					SELECT c.* FROM ewrite_collections c WHERE c.id = ANY($1)
					UNION
					SELECT parent.* FROM ewrite_collections parent
					JOIN reachable child ON child.parent_id = parent.id
				)
				SELECT id::text, location_id::text, COALESCE(parent_id::text, ''), kind, title, slug, summary,
				       sort_order, visibility, created_at, updated_at
				FROM reachable
				ORDER BY sort_order, title`, ids)
			if err != nil {
				writeError(w, err)
				return
			}
			for crows.Next() {
				var c Collection
				if err := crows.Scan(&c.ID, &c.LocationID, &c.ParentID, &c.Kind, &c.Title, &c.Slug, &c.Summary,
					&c.SortOrder, &c.Visibility, &c.CreatedAt, &c.UpdatedAt); err != nil {
					crows.Close()
					writeError(w, err)
					return
				}
				tree.Collections = append(tree.Collections, c)
			}
			crows.Close()
			if err := crows.Err(); err != nil {
				writeError(w, err)
				return
			}
		}

		writeOK(w, tree)
	}
}

// HandleLibraryPublication serves GET /api/library/publications/{publication_id}.
// Query param ?anchor= resolves a requested section anchor (alias-aware);
// the payload reports resolution so the reader can land exactly or show a
// clear "section not found" notice (spec 6.3) without guessing.
func HandleLibraryPublication(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
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
			// Deliberately publication_not_found, not forbidden: a denial
			// must not confirm that a hidden title exists (spec 6.3).
			writeError(w, errors.New("publication_not_found"))
			return
		}

		sections, err := LoadSections(ctx, pool, id)
		if err != nil {
			writeError(w, err)
			return
		}

		ancestorRows, err := pool.Query(ctx, `
			WITH RECURSIVE chain AS (
				SELECT id, parent_id, kind, title, 0 AS depth FROM ewrite_collections WHERE id = $1
				UNION ALL
				SELECT c.id, c.parent_id, c.kind, c.title, chain.depth + 1
				FROM ewrite_collections c JOIN chain ON chain.parent_id = c.id
			)
			SELECT id::text, kind, title FROM chain ORDER BY depth DESC
		`, p.CollectionID)
		if err != nil {
			writeError(w, err)
			return
		}
		ancestors := []CollectionAncestor{}
		for ancestorRows.Next() {
			var a CollectionAncestor
			if err := ancestorRows.Scan(&a.ID, &a.Kind, &a.Title); err != nil {
				ancestorRows.Close()
				writeError(w, err)
				return
			}
			ancestors = append(ancestors, a)
		}
		ancestorRows.Close()
		if err := ancestorRows.Err(); err != nil {
			writeError(w, err)
			return
		}

		payload := map[string]any{
			"publication": readerProjection(p),
			"sections":    sections,
			"ancestors":   ancestors,
		}

		if anchor := strings.TrimSpace(r.URL.Query().Get("anchor")); anchor != "" {
			resolved, found, err := ResolveAnchor(ctx, pool, id, anchor)
			if err != nil {
				writeError(w, err)
				return
			}
			payload["requested_anchor"] = anchor
			payload["resolved_anchor"] = resolved
			payload["anchor_found"] = found
		}

		// Previous/next among readable siblings in the same collection.
		operator, err := isOperatorFlag(ctx, pool, userID)
		if err != nil {
			writeError(w, err)
			return
		}
		siblings, err := pool.Query(ctx, `
			SELECT p.id::text, p.title
			FROM ewrite_publications p
			WHERE p.collection_id = $3 AND `+visiblePublicationsClause+`
			ORDER BY p.sort_order, p.published_at`, userID, operator, p.CollectionID)
		if err != nil {
			writeError(w, err)
			return
		}
		type sibling struct{ ID, Title string }
		var ordered []sibling
		for siblings.Next() {
			var s sibling
			if err := siblings.Scan(&s.ID, &s.Title); err != nil {
				siblings.Close()
				writeError(w, err)
				return
			}
			ordered = append(ordered, s)
		}
		siblings.Close()
		if err := siblings.Err(); err != nil {
			writeError(w, err)
			return
		}
		for i, s := range ordered {
			if s.ID == p.ID {
				if i > 0 {
					payload["previous"] = map[string]string{"id": ordered[i-1].ID, "title": ordered[i-1].Title}
				}
				if i+1 < len(ordered) {
					payload["next"] = map[string]string{"id": ordered[i+1].ID, "title": ordered[i+1].Title}
				}
				break
			}
		}

		writeOK(w, payload)
	}
}

// CollectionAncestor is one breadcrumb step, root-first.
type CollectionAncestor struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
}

// HandleLibraryCollection serves GET /api/library/collections/{collection_id}
// -- the Ruleset/Series/Module landing page (spec 9.1-9.3): breadcrumb,
// direct child collections, readable publications, and directories,
// generic across all three collection kinds since the shape is identical.
func HandleLibraryCollection(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		id := strings.TrimSpace(r.PathValue("collection_id"))

		var c Collection
		if err := pool.QueryRow(ctx, `
			SELECT id::text, location_id::text, COALESCE(parent_id::text, ''), kind, title, slug, summary, sort_order, visibility, created_at, updated_at
			FROM ewrite_collections WHERE id = $1
		`, id).Scan(&c.ID, &c.LocationID, &c.ParentID, &c.Kind, &c.Title, &c.Slug, &c.Summary, &c.SortOrder, &c.Visibility, &c.CreatedAt, &c.UpdatedAt); errors.Is(err, pgx.ErrNoRows) {
			writeError(w, errors.New("collection_not_found"))
			return
		} else if err != nil {
			writeError(w, err)
			return
		}

		ancestorRows, err := pool.Query(ctx, `
			WITH RECURSIVE chain AS (
				SELECT id, parent_id, kind, title, 0 AS depth FROM ewrite_collections WHERE id = $1
				UNION ALL
				SELECT p.id, p.parent_id, p.kind, p.title, chain.depth + 1
				FROM ewrite_collections p JOIN chain ON chain.parent_id = p.id
			)
			SELECT id::text, kind, title FROM chain WHERE id <> $1 ORDER BY depth DESC
		`, id)
		if err != nil {
			writeError(w, err)
			return
		}
		ancestors := []CollectionAncestor{}
		for ancestorRows.Next() {
			var a CollectionAncestor
			if err := ancestorRows.Scan(&a.ID, &a.Kind, &a.Title); err != nil {
				ancestorRows.Close()
				writeError(w, err)
				return
			}
			ancestors = append(ancestors, a)
		}
		ancestorRows.Close()
		if err := ancestorRows.Err(); err != nil {
			writeError(w, err)
			return
		}

		children := []Collection{}
		childRows, err := pool.Query(ctx, `
			SELECT id::text, location_id::text, COALESCE(parent_id::text, ''), kind, title, slug, summary, sort_order, visibility, created_at, updated_at
			FROM ewrite_collections WHERE parent_id = $1 ORDER BY sort_order, title
		`, id)
		if err != nil {
			writeError(w, err)
			return
		}
		for childRows.Next() {
			var cc Collection
			if err := childRows.Scan(&cc.ID, &cc.LocationID, &cc.ParentID, &cc.Kind, &cc.Title, &cc.Slug, &cc.Summary, &cc.SortOrder, &cc.Visibility, &cc.CreatedAt, &cc.UpdatedAt); err != nil {
				childRows.Close()
				writeError(w, err)
				return
			}
			children = append(children, cc)
		}
		childRows.Close()
		if err := childRows.Err(); err != nil {
			writeError(w, err)
			return
		}

		operator, err := isOperatorFlag(ctx, pool, userID)
		if err != nil {
			writeError(w, err)
			return
		}
		pubs := []Publication{}
		pubRows, err := pool.Query(ctx, `
			SELECT p.id::text, p.collection_id::text, p.location_id::text, p.title, p.slug, p.summary,
			       p.word_count, p.status, p.visibility, p.published_at, p.created_at, p.updated_at
			FROM ewrite_publications p
			WHERE p.collection_id = $3 AND `+visiblePublicationsClause+`
			ORDER BY p.sort_order, p.published_at`, userID, operator, id)
		if err != nil {
			writeError(w, err)
			return
		}
		for pubRows.Next() {
			var p Publication
			if err := pubRows.Scan(&p.ID, &p.CollectionID, &p.LocationID, &p.Title, &p.Slug, &p.Summary,
				&p.WordCount, &p.Status, &p.Visibility, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
				pubRows.Close()
				writeError(w, err)
				return
			}
			pubs = append(pubs, p)
		}
		pubRows.Close()
		if err := pubRows.Err(); err != nil {
			writeError(w, err)
			return
		}

		directories, err := ListDirectoriesForCollection(ctx, pool, id)
		if err != nil {
			writeError(w, err)
			return
		}

		writeOK(w, map[string]any{
			"collection":   c,
			"ancestors":    ancestors,
			"children":     children,
			"publications": pubs,
			"directories":  directories,
		})
	}
}

// readerProjection strips authoring-only fields from the reader payload:
// no source markdown (export serves that deliberately), no updated_by
// identity, no current_revision_id (spec 6.5 "private source metadata").
func readerProjection(p *Publication) map[string]any {
	return map[string]any{
		"id":            p.ID,
		"collection_id": p.CollectionID,
		"title":         p.Title,
		"slug":          p.Slug,
		"summary":       p.Summary,
		"rendered_html": p.RenderedHTML,
		"word_count":    p.WordCount,
		"visibility":    p.Visibility,
		"published_at":  p.PublishedAt,
	}
}

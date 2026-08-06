package ewrite

// Library search: Victory's first PostgreSQL full-text search.
// websearch_to_tsquery gives readers ordinary quoted/OR/minus syntax with
// no parse errors on junk input; ts_rank orders; ts_headline builds
// highlighted snippets from the clean extracted search_text (never raw
// Markdown, never HTML). Results run through the same visibility predicate
// as every reader query -- drafts are structurally invisible (spec 17.3
// "unpublished content not discoverable through search").

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SearchResult is one hit with a highlighted snippet.
type SearchResult struct {
	PublicationID string  `json:"publication_id"`
	Title         string  `json:"title"`
	Summary       string  `json:"summary"`
	Snippet       string  `json:"snippet"`
	Rank          float32 `json:"rank"`
}

// DirectorySearchResult is one directory-entry hit (spec 9.5 "directory
// matches") -- a compact index result, not full rule text, resolved
// through the exact same visibility-safe path as ListDirectoryEntries so a
// search box cannot become a second way to leak a hidden target.
type DirectorySearchResult struct {
	EntryID             string `json:"entry_id"`
	DirectoryID         string `json:"directory_id"`
	DirectoryTitle      string `json:"directory_title"`
	CanonicalName       string `json:"canonical_name"`
	Category            string `json:"category,omitempty"`
	CompactSummary      string `json:"compact_summary,omitempty"`
	LinkStatus          string `json:"link_status"`
	TargetPublicationID string `json:"target_publication_id,omitempty"`
	TargetSectionAnchor string `json:"target_section_anchor,omitempty"`
}

// collectionSubtreeClause resolves to every collection ID in the subtree
// rooted at $N (inclusive) -- shared by scoped publication search and
// scoped directory search so "search within this Ruleset" means the same
// thing in both.
const collectionSubtreeCTE = `
	WITH RECURSIVE sub AS (
		SELECT id FROM ewrite_collections WHERE id = $%d
		UNION ALL
		SELECT c.id FROM ewrite_collections c JOIN sub ON c.parent_id = sub.id
	)
`

// SearchPublications runs the visibility-filtered FTS query, optionally
// scoped to one Ruleset/Series/Module's subtree (spec 9.5, "Ruleset-scoped
// search"). An empty collectionID searches the whole Library, matching
// pre-Kernel-79A behavior exactly.
func SearchPublications(ctx context.Context, pool *pgxpool.Pool, userID string, operator bool, query, collectionID string, limit int) ([]SearchResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	sql := `
		SELECT p.id::text, p.title, p.summary,
		       ts_headline('english', left(p.search_text, 200000), q,
		                   'StartSel=<mark>, StopSel=</mark>, MaxWords=40, MinWords=15, MaxFragments=2, FragmentDelimiter= … ') AS snippet,
		       ts_rank(p.search_tsv, q) AS rank
		FROM ewrite_publications p, websearch_to_tsquery('english', $3) q
		WHERE p.search_tsv @@ q AND ` + visiblePublicationsClause
	args := []any{userID, operator, query}
	if collectionID != "" {
		sql = collectionSubtreeCTEAt(5) + sql + ` AND p.collection_id IN (SELECT id FROM sub)`
		args = append(args, limit, collectionID)
		sql += ` ORDER BY rank DESC, p.published_at DESC LIMIT $4`
	} else {
		args = append(args, limit)
		sql += ` ORDER BY rank DESC, p.published_at DESC LIMIT $4`
	}
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SearchResult{}
	for rows.Next() {
		var s SearchResult
		if err := rows.Scan(&s.PublicationID, &s.Title, &s.Summary, &s.Snippet, &s.Rank); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func collectionSubtreeCTEAt(placeholder int) string {
	return fmt.Sprintf(collectionSubtreeCTE, placeholder)
}

// SearchDirectoryEntries finds directory entries (across all directories,
// or scoped to one Ruleset's subtree) matching query, resolved through
// ListDirectoryEntries's own visibility-safe logic -- reused rather than
// duplicated so a hidden target can't leak through search results that
// forgot the rule the browse view already enforces.
func SearchDirectoryEntries(ctx context.Context, pool *pgxpool.Pool, userID, query, collectionID string, limit int) ([]DirectorySearchResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	var rows pgx.Rows
	var err error
	if collectionID == "" {
		rows, err = pool.Query(ctx, `SELECT id::text, title FROM ewrite_directories ORDER BY title`)
	} else {
		rows, err = pool.Query(ctx, collectionSubtreeCTEAt(1)+`
			SELECT d.id::text, d.title FROM ewrite_directories d WHERE d.collection_id IN (SELECT id FROM sub) ORDER BY d.title
		`, collectionID)
	}
	if err != nil {
		return nil, err
	}
	type dirRow struct{ id, title string }
	var dirs []dirRow
	for rows.Next() {
		var d dirRow
		if err := rows.Scan(&d.id, &d.title); err != nil {
			rows.Close()
			return nil, err
		}
		dirs = append(dirs, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := []DirectorySearchResult{}
	for _, d := range dirs {
		entries, err := ListDirectoryEntries(ctx, pool, userID, d.id, query, "")
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if len(out) >= limit {
				return out, nil
			}
			out = append(out, DirectorySearchResult{
				EntryID: e.ID, DirectoryID: d.id, DirectoryTitle: d.title,
				CanonicalName: e.CanonicalName, Category: e.Category, CompactSummary: e.CompactSummary,
				LinkStatus: e.LinkStatus, TargetPublicationID: e.TargetPublicationID, TargetSectionAnchor: e.TargetSectionAnchor,
			})
		}
	}
	return out, nil
}

// HandleLibrarySearch serves GET /api/library/search?q=...&collection_id=
func HandleLibrarySearch(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			writeError(w, errors.New("query_required"))
			return
		}
		if len(query) > 200 {
			query = query[:200]
		}
		collectionID := strings.TrimSpace(r.URL.Query().Get("collection_id"))
		operator, err := isOperatorFlag(ctx, pool, userID)
		if err != nil {
			writeError(w, err)
			return
		}
		results, err := SearchPublications(ctx, pool, userID, operator, query, collectionID, 20)
		if err != nil {
			writeError(w, err)
			return
		}
		dirResults, err := SearchDirectoryEntries(ctx, pool, userID, query, collectionID, 10)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"results": results, "directory_results": dirResults, "query": query})
	}
}

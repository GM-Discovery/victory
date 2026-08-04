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
	"net/http"
	"strings"
	"time"

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

// SearchPublications runs the visibility-filtered FTS query.
func SearchPublications(ctx context.Context, pool *pgxpool.Pool, userID string, operator bool, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := pool.Query(ctx, `
		SELECT p.id::text, p.title, p.summary,
		       ts_headline('english', left(p.search_text, 200000), q,
		                   'StartSel=<mark>, StopSel=</mark>, MaxWords=40, MinWords=15, MaxFragments=2, FragmentDelimiter= … ') AS snippet,
		       ts_rank(p.search_tsv, q) AS rank
		FROM ewrite_publications p, websearch_to_tsquery('english', $3) q
		WHERE p.search_tsv @@ q AND `+visiblePublicationsClause+`
		ORDER BY rank DESC, p.published_at DESC
		LIMIT $4
	`, userID, operator, query, limit)
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

// HandleLibrarySearch serves GET /api/library/search?q=...
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
		operator, err := isOperatorFlag(ctx, pool, userID)
		if err != nil {
			writeError(w, err)
			return
		}
		results, err := SearchPublications(ctx, pool, userID, operator, query, 20)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"results": results, "query": query})
	}
}

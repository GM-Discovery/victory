package ewrite

// Save, conflict, publish. The heart of Goal G: an ordinary save must
// never silently erase a newer revision.
//
// Save protocol: the client submits {source_markdown, base_revision_id}
// where base_revision_id is the current_revision_id it loaded. Inside a
// transaction holding FOR UPDATE on the publication row:
//
//   base != current  ->  ErrRevisionConflict (HTTP 409) carrying the
//                        current revision info. The handler never touches
//                        the submitted source; the client keeps it in the
//                        textarea and localStorage.
//   base == current  ->  render -> INSERT revision (max+1, append-forward)
//                        -> UPDATE publication (source + rendered caches +
//                        current_revision_id) -> reconcile sections ->
//                        commit.
//
// rendered_html/search_text/word_count are regenerated inside this same
// transaction -- the only write path for source -- so the cache can never
// be stale (spec 5.6 "avoid half-published state" generalized to saves).
//
// Restore (spec 14.2) is deliberately client-flow: the UI loads an old
// revision's source into the editor and saves it as a NEW revision through
// this same path. History is never rewritten; no dedicated restore
// endpoint exists to bypass the conflict check.

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// queryer lets section/anchor loaders run against either the pool or an
// open transaction.
type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// RevisionConflict carries what the client needs to resolve a 409.
type RevisionConflict struct {
	CurrentRevisionID     string `json:"current_revision_id"`
	CurrentRevisionNumber int    `json:"current_revision_number"`
	UpdatedBy             string `json:"updated_by,omitempty"`
	UpdatedByName         string `json:"updated_by_name,omitempty"`
	UpdatedAt             string `json:"updated_at"`
}

var ErrRevisionConflict = errors.New("revision_conflict")

// SaveResult reports a successful save.
type SaveResult struct {
	Publication    *Publication `json:"publication"`
	RevisionID     string       `json:"revision_id"`
	RevisionNumber int          `json:"revision_number"`
	NoChange       bool         `json:"no_change"`
	Report         ImportReport `json:"report"`
}

// SavePublicationSource is the single write path for source_markdown.
func SavePublicationSource(ctx context.Context, pool *pgxpool.Pool, userID, publicationID, source, baseRevisionID string) (*SaveResult, *RevisionConflict, error) {
	p, err := LoadPublication(ctx, pool, publicationID)
	if err != nil {
		return nil, nil, err
	}
	if ok, err := CanEditPublication(ctx, pool, userID, p); err != nil {
		return nil, nil, err
	} else if !ok {
		return nil, nil, errors.New("not_authorized")
	}

	res, err := Render(source)
	if err != nil {
		return nil, nil, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	var currentRevisionID, currentSource string
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(current_revision_id::text, ''), source_markdown
		FROM ewrite_publications WHERE id = $1 FOR UPDATE
	`, publicationID).Scan(&currentRevisionID, &currentSource); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(baseRevisionID) != currentRevisionID {
		conflict, err := loadConflict(ctx, tx, publicationID, currentRevisionID)
		if err != nil {
			return nil, nil, err
		}
		return nil, conflict, ErrRevisionConflict
	}

	// No-change save: defined as a no-op (spec 17.6 "no-change save
	// behavior defined"). No new revision, no timestamp churn.
	if currentSource == source && currentRevisionID != "" {
		var num int
		if err := tx.QueryRow(ctx, `SELECT revision_number FROM ewrite_revisions WHERE id = $1`, currentRevisionID).Scan(&num); err != nil {
			return nil, nil, err
		}
		return &SaveResult{Publication: p, RevisionID: currentRevisionID, RevisionNumber: num, NoChange: true, Report: buildImportReport(source, res)}, nil, nil
	}

	var revisionID string
	var revisionNumber int
	if err := tx.QueryRow(ctx, `
		INSERT INTO ewrite_revisions (publication_id, revision_number, source_markdown, base_revision_id, created_by)
		SELECT $1, COALESCE(MAX(revision_number), 0) + 1, $2, NULLIF($3, '')::uuid, $4
		FROM ewrite_revisions WHERE publication_id = $1
		RETURNING id::text, revision_number
	`, publicationID, source, baseRevisionID, userID).Scan(&revisionID, &revisionNumber); err != nil {
		return nil, nil, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE ewrite_publications
		SET source_markdown = $2, rendered_html = $3, search_text = $4, word_count = $5,
		    current_revision_id = $6, updated_by = $7, updated_at = NOW()
		WHERE id = $1
	`, publicationID, source, res.HTML, res.PlainText, res.WordCount, revisionID, userID); err != nil {
		return nil, nil, err
	}

	if err := reconcileSections(ctx, tx, publicationID, res.Outline); err != nil {
		return nil, nil, err
	}

	if err := reconcilePublicationAssetRefs(ctx, tx, publicationID, res.ImageRefs); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	saved, err := LoadPublication(ctx, pool, publicationID)
	if err != nil {
		return nil, nil, err
	}
	return &SaveResult{Publication: saved, RevisionID: revisionID, RevisionNumber: revisionNumber, Report: buildImportReport(source, res)}, nil, nil
}

func loadConflict(ctx context.Context, q queryer, publicationID, currentRevisionID string) (*RevisionConflict, error) {
	c := &RevisionConflict{CurrentRevisionID: currentRevisionID}
	if currentRevisionID == "" {
		return c, nil
	}
	var updatedBy, updatedByName, updatedAt string
	err := q.QueryRow(ctx, `
		SELECT r.revision_number, COALESCE(r.created_by::text, ''), COALESCE(u.display_name, ''), r.created_at::text
		FROM ewrite_revisions r
		LEFT JOIN users u ON u.id = r.created_by
		WHERE r.id = $1
	`, currentRevisionID).Scan(&c.CurrentRevisionNumber, &updatedBy, &updatedByName, &updatedAt)
	if err != nil {
		return nil, err
	}
	c.UpdatedBy = updatedBy
	c.UpdatedByName = updatedByName
	c.UpdatedAt = updatedAt
	return c, nil
}

// PublishPublication flips a draft/archived publication to published.
// Content itself is untouched: rendered_html is already current by the
// save-path invariant, so publish is a pure state transition and cannot
// half-publish (spec 5.6).
func PublishPublication(ctx context.Context, pool *pgxpool.Pool, userID, publicationID string) (*Publication, error) {
	p, err := LoadPublication(ctx, pool, publicationID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanPublishPublication(ctx, pool, userID, p); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("not_authorized")
	}
	if p.CurrentRevisionID == "" {
		return nil, errors.New("nothing_to_publish")
	}
	if _, err := pool.Exec(ctx, `
		UPDATE ewrite_publications
		SET status = 'published', published_at = COALESCE(published_at, NOW()), updated_by = $2, updated_at = NOW()
		WHERE id = $1
	`, publicationID, userID); err != nil {
		return nil, err
	}
	return LoadPublication(ctx, pool, publicationID)
}

// UnpublishPublication returns a published work to draft. Nothing is
// destroyed (spec 5.6 "unpublishing must not destroy the work").
func UnpublishPublication(ctx context.Context, pool *pgxpool.Pool, userID, publicationID string) (*Publication, error) {
	p, err := LoadPublication(ctx, pool, publicationID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanPublishPublication(ctx, pool, userID, p); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("not_authorized")
	}
	if _, err := pool.Exec(ctx, `
		UPDATE ewrite_publications
		SET status = 'draft', updated_by = $2, updated_at = NOW()
		WHERE id = $1
	`, publicationID, userID); err != nil {
		return nil, err
	}
	return LoadPublication(ctx, pool, publicationID)
}

// ListRevisions returns revision metadata, newest first. Source bodies are
// omitted; LoadRevision fetches one.
func ListRevisions(ctx context.Context, pool *pgxpool.Pool, publicationID string) ([]Revision, error) {
	rows, err := pool.Query(ctx, `
		SELECT r.id::text, r.publication_id::text, r.revision_number,
		       COALESCE(r.base_revision_id::text, ''), COALESCE(r.created_by::text, ''),
		       COALESCE(u.display_name, ''), r.created_at, length(r.source_markdown)
		FROM ewrite_revisions r
		LEFT JOIN users u ON u.id = r.created_by
		WHERE r.publication_id = $1
		ORDER BY r.revision_number DESC
	`, publicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Revision{}
	for rows.Next() {
		var rev Revision
		if err := rows.Scan(&rev.ID, &rev.PublicationID, &rev.RevisionNumber, &rev.BaseRevisionID, &rev.CreatedBy, &rev.CreatedByName, &rev.CreatedAt, &rev.ByteSize); err != nil {
			return nil, err
		}
		out = append(out, rev)
	}
	return out, rows.Err()
}

// LoadRevision fetches one revision including source.
func LoadRevision(ctx context.Context, pool *pgxpool.Pool, revisionID string) (*Revision, error) {
	var rev Revision
	err := pool.QueryRow(ctx, `
		SELECT r.id::text, r.publication_id::text, r.revision_number, r.source_markdown,
		       COALESCE(r.base_revision_id::text, ''), COALESCE(r.created_by::text, ''),
		       COALESCE(u.display_name, ''), r.created_at, length(r.source_markdown)
		FROM ewrite_revisions r
		LEFT JOIN users u ON u.id = r.created_by
		WHERE r.id = $1
	`, revisionID).Scan(&rev.ID, &rev.PublicationID, &rev.RevisionNumber, &rev.SourceMarkdown, &rev.BaseRevisionID, &rev.CreatedBy, &rev.CreatedByName, &rev.CreatedAt, &rev.ByteSize)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("revision_not_found")
	}
	if err != nil {
		return nil, err
	}
	return &rev, nil
}

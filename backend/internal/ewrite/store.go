package ewrite

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

var validVisibilities = map[string]bool{"public": true, "authenticated": true, "production": true}
var validCollectionKinds = map[string]bool{"ruleset": true, "series": true, "module": true}

// validateParentKind enforces the visible hierarchy on the typed tree:
// ruleset is a root; series sits under a ruleset; module sits under a
// ruleset or a series (the omitted-Series case is deliberate -- spec 1.2
// says levels may be skipped, and a small game with one module should not
// need a ceremonial Series).
func validateParentKind(childKind, parentKind string) error {
	switch childKind {
	case "ruleset":
		if parentKind != "" {
			return errors.New("ruleset_must_be_root")
		}
	case "series":
		if parentKind != "ruleset" {
			return errors.New("series_parent_must_be_ruleset")
		}
	case "module":
		if parentKind != "ruleset" && parentKind != "series" {
			return errors.New("module_parent_must_be_ruleset_or_series")
		}
	default:
		return errors.New("invalid_collection_kind")
	}
	return nil
}

// uniqueSlug generates a slug from the title and dedupes with -2/-3
// suffixes inside the given uniqueness scope (spec 17.1 "duplicate slug
// handled safely"). scopeQuery must be an EXISTS query with $1 = candidate.
func uniqueSlug(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, title, scopeQuery string, scopeArgs ...any) (string, error) {
	base := slugifyAnchor(title)
	if base == "" {
		base = "untitled"
	}
	candidate := base
	for i := 2; ; i++ {
		var exists bool
		args := append([]any{candidate}, scopeArgs...)
		if err := q.QueryRow(ctx, scopeQuery, args...).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

// CreateCollection validates authority (Crew+ at the location), the parent
// kind rule, and visibility, then inserts. The client's location_id and
// parent_id are row selectors only -- authority is derived server-side.
func CreateCollection(ctx context.Context, pool *pgxpool.Pool, userID, locationID, parentID, kind, title, summary, visibility string) (*Collection, error) {
	locationID = strings.TrimSpace(locationID)
	title = strings.TrimSpace(title)
	kind = strings.ToLower(strings.TrimSpace(kind))
	if title == "" {
		return nil, errors.New("title_required")
	}
	if !validCollectionKinds[kind] {
		return nil, errors.New("invalid_collection_kind")
	}
	if visibility == "" {
		visibility = "production"
	}
	if !validVisibilities[visibility] {
		return nil, errors.New("invalid_visibility")
	}

	parentKind := ""
	parentID = strings.TrimSpace(parentID)
	if parentID != "" {
		var parentLocation string
		if err := pool.QueryRow(ctx, `SELECT kind, location_id::text FROM ewrite_collections WHERE id = $1`, parentID).Scan(&parentKind, &parentLocation); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, errors.New("parent_collection_not_found")
			}
			return nil, err
		}
		if locationID == "" {
			locationID = parentLocation
		} else if locationID != parentLocation {
			return nil, errors.New("parent_collection_location_mismatch")
		}
	}
	if locationID == "" {
		return nil, errors.New("location_required")
	}
	if err := validateParentKind(kind, parentKind); err != nil {
		return nil, err
	}

	if ok, err := CanAuthorInScope(ctx, pool, userID, locationID); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("not_authorized")
	}

	slug, err := uniqueSlug(ctx, pool, title,
		`SELECT EXISTS(SELECT 1 FROM ewrite_collections WHERE slug = $1 AND location_id = $2)`, locationID)
	if err != nil {
		return nil, err
	}

	var c Collection
	var parent, createdBy *string
	err = pool.QueryRow(ctx, `
		INSERT INTO ewrite_collections (location_id, parent_id, kind, title, slug, summary, visibility, created_by)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7, $8)
		RETURNING id::text, location_id::text, COALESCE(parent_id::text, ''), kind, title, slug, summary, sort_order, visibility, created_at, updated_at
	`, locationID, parentID, kind, title, slug, strings.TrimSpace(summary), visibility, userID).Scan(
		&c.ID, &c.LocationID, &c.ParentID, &c.Kind, &c.Title, &c.Slug, &c.Summary, &c.SortOrder, &c.Visibility, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = parent
	_ = createdBy
	c.CreatedBy = userID
	return &c, nil
}

func loadCollection(ctx context.Context, pool *pgxpool.Pool, id string) (*Collection, error) {
	var c Collection
	err := pool.QueryRow(ctx, `
		SELECT id::text, location_id::text, COALESCE(parent_id::text, ''), kind, title, slug, summary, sort_order, visibility,
		       COALESCE(created_by::text, ''), created_at, updated_at
		FROM ewrite_collections WHERE id = $1
	`, id).Scan(&c.ID, &c.LocationID, &c.ParentID, &c.Kind, &c.Title, &c.Slug, &c.Summary, &c.SortOrder, &c.Visibility, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("collection_not_found")
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// PatchCollection updates title/summary/sort_order/visibility/parent.
// Re-parenting revalidates the kind rule and same-location rule.
func PatchCollection(ctx context.Context, pool *pgxpool.Pool, userID, collectionID string, fields map[string]any) (*Collection, error) {
	c, err := loadCollection(ctx, pool, collectionID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanAuthorInScope(ctx, pool, userID, c.LocationID); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("not_authorized")
	}

	sets := []string{"updated_at = NOW()"}
	args := []any{collectionID}
	add := func(expr string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf(expr, len(args)))
	}

	if v, ok := fields["title"].(string); ok {
		v = strings.TrimSpace(v)
		if v == "" {
			return nil, errors.New("title_required")
		}
		add("title = $%d", v)
	}
	if v, ok := fields["summary"].(string); ok {
		add("summary = $%d", strings.TrimSpace(v))
	}
	if v, ok := fields["sort_order"].(float64); ok {
		add("sort_order = $%d", int(v))
	}
	if v, ok := fields["visibility"].(string); ok {
		if !validVisibilities[v] {
			return nil, errors.New("invalid_visibility")
		}
		add("visibility = $%d", v)
	}
	if v, ok := fields["parent_id"].(string); ok {
		v = strings.TrimSpace(v)
		parentKind := ""
		if v != "" {
			var parentLocation string
			if err := pool.QueryRow(ctx, `SELECT kind, location_id::text FROM ewrite_collections WHERE id = $1`, v).Scan(&parentKind, &parentLocation); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return nil, errors.New("parent_collection_not_found")
				}
				return nil, err
			}
			if parentLocation != c.LocationID {
				return nil, errors.New("parent_collection_location_mismatch")
			}
			if v == c.ID {
				return nil, errors.New("collection_cannot_parent_itself")
			}
		}
		if err := validateParentKind(c.Kind, parentKind); err != nil {
			return nil, err
		}
		add("parent_id = NULLIF($%d, '')::uuid", v)
	}

	query := fmt.Sprintf(`UPDATE ewrite_collections SET %s WHERE id = $1`, strings.Join(sets, ", "))
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		return nil, err
	}
	return loadCollection(ctx, pool, collectionID)
}

// DeleteCollection deletes an empty collection. The RESTRICT foreign keys
// turn a non-empty delete into a loud failure, surfaced as
// collection_not_empty.
func DeleteCollection(ctx context.Context, pool *pgxpool.Pool, userID, collectionID string) error {
	c, err := loadCollection(ctx, pool, collectionID)
	if err != nil {
		return err
	}
	if ok, err := CanAuthorInScope(ctx, pool, userID, c.LocationID); err != nil {
		return err
	} else if !ok {
		return errors.New("not_authorized")
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ewrite_collections WHERE id = $1`, collectionID); err != nil {
		if strings.Contains(err.Error(), "violates foreign key constraint") {
			return errors.New("collection_not_empty")
		}
		return err
	}
	return nil
}

// CreatePublication creates an empty draft under a collection. Location is
// derived from the collection row, never from the client.
func CreatePublication(ctx context.Context, pool *pgxpool.Pool, userID, collectionID, title, summary, visibility string) (*Publication, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("title_required")
	}
	c, err := loadCollection(ctx, pool, strings.TrimSpace(collectionID))
	if err != nil {
		return nil, err
	}
	if visibility == "" {
		visibility = "production"
	}
	if !validVisibilities[visibility] {
		return nil, errors.New("invalid_visibility")
	}
	if ok, err := CanAuthorInScope(ctx, pool, userID, c.LocationID); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("not_authorized")
	}

	slug, err := uniqueSlug(ctx, pool, title,
		`SELECT EXISTS(SELECT 1 FROM ewrite_publications WHERE slug = $1 AND collection_id = $2)`, c.ID)
	if err != nil {
		return nil, err
	}

	var p Publication
	err = pool.QueryRow(ctx, `
		INSERT INTO ewrite_publications (collection_id, location_id, title, slug, summary, visibility, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id::text, created_at, updated_at
	`, c.ID, c.LocationID, title, slug, strings.TrimSpace(summary), visibility, userID).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.CollectionID = c.ID
	p.LocationID = c.LocationID
	p.Title = title
	p.Slug = slug
	p.Summary = strings.TrimSpace(summary)
	p.Status = "draft"
	p.Visibility = visibility
	p.CreatedBy = userID
	p.UpdatedBy = userID
	return &p, nil
}

const publicationColumns = `
	id::text, collection_id::text, location_id::text, title, slug, summary,
	source_markdown, rendered_html, word_count, status, visibility,
	COALESCE(current_revision_id::text, ''), COALESCE(created_by::text, ''),
	COALESCE(updated_by::text, ''), published_at, created_at, updated_at`

func scanPublication(row pgx.Row) (*Publication, error) {
	var p Publication
	err := row.Scan(&p.ID, &p.CollectionID, &p.LocationID, &p.Title, &p.Slug, &p.Summary,
		&p.SourceMarkdown, &p.RenderedHTML, &p.WordCount, &p.Status, &p.Visibility,
		&p.CurrentRevisionID, &p.CreatedBy, &p.UpdatedBy, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("publication_not_found")
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// LoadPublication fetches one publication with source. No authority check
// here -- callers gate with the Can* helper appropriate to their surface.
func LoadPublication(ctx context.Context, pool *pgxpool.Pool, id string) (*Publication, error) {
	return scanPublication(pool.QueryRow(ctx,
		`SELECT `+publicationColumns+` FROM ewrite_publications WHERE id = $1`, id))
}

// PatchPublication updates metadata fields (title/slug via title change,
// summary, visibility, collection move, draft<->archived). Publish state
// transitions live in revisions.go.
func PatchPublication(ctx context.Context, pool *pgxpool.Pool, userID, publicationID string, fields map[string]any) (*Publication, error) {
	p, err := LoadPublication(ctx, pool, publicationID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditPublication(ctx, pool, userID, p); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("not_authorized")
	}

	sets := []string{"updated_at = NOW()", "updated_by = $2"}
	args := []any{publicationID, userID}
	add := func(expr string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf(expr, len(args)))
	}

	if v, ok := fields["title"].(string); ok {
		v = strings.TrimSpace(v)
		if v == "" {
			return nil, errors.New("title_required")
		}
		add("title = $%d", v)
	}
	if v, ok := fields["summary"].(string); ok {
		add("summary = $%d", strings.TrimSpace(v))
	}
	if v, ok := fields["visibility"].(string); ok {
		if !validVisibilities[v] {
			return nil, errors.New("invalid_visibility")
		}
		add("visibility = $%d", v)
	}
	if v, ok := fields["sort_order"].(float64); ok {
		add("sort_order = $%d", int(v))
	}
	if v, ok := fields["collection_id"].(string); ok {
		v = strings.TrimSpace(v)
		target, err := loadCollection(ctx, pool, v)
		if err != nil {
			return nil, err
		}
		if target.LocationID != p.LocationID {
			// Cross-location moves would silently re-scope reading
			// authority; refuse rather than surprise (spec 10.4).
			return nil, errors.New("collection_location_mismatch")
		}
		add("collection_id = $%d", v)
	}
	if v, ok := fields["status"].(string); ok {
		switch {
		case v == "archived" && p.Status != "published":
			add("status = $%d", v)
		case v == "draft" && p.Status == "archived":
			add("status = $%d", v)
		default:
			return nil, errors.New("invalid_status_transition")
		}
	}

	query := fmt.Sprintf(`UPDATE ewrite_publications SET %s WHERE id = $1`, strings.Join(sets, ", "))
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		return nil, err
	}
	return LoadPublication(ctx, pool, publicationID)
}

// DeletePublication removes a non-published publication and (via CASCADE)
// its revisions, sections, aliases, grants, and object links.
func DeletePublication(ctx context.Context, pool *pgxpool.Pool, userID, publicationID string) error {
	p, err := LoadPublication(ctx, pool, publicationID)
	if err != nil {
		return err
	}
	if p.Status == "published" {
		return errors.New("unpublish_before_delete")
	}
	if ok, err := CanDeletePublication(ctx, pool, userID, p); err != nil {
		return err
	} else if !ok {
		return errors.New("not_authorized")
	}
	_, err = pool.Exec(ctx, `DELETE FROM ewrite_publications WHERE id = $1`, publicationID)
	return err
}

// AuthoringTree is the Writer's Room bootstrap payload: every collection
// and publication at locations where the caller holds an authoring role.
// Draft sources are omitted (metadata only) -- the editor loads source per
// publication.
type AuthoringTree struct {
	Locations    []TreeLocation `json:"locations"`
	RecentDrafts []Publication  `json:"recent_drafts"`
}

type TreeLocation struct {
	LocationID   string        `json:"location_id"`
	LocationName string        `json:"location_name"`
	Collections  []Collection  `json:"collections"`
	Publications []Publication `json:"publications"`
}

// authoringLocationIDs returns locations where the user may author:
// active producer/director/crew membership, or every location holding
// eWrite content for the Operator.
func authoringLocationIDs(ctx context.Context, pool *pgxpool.Pool, userID string) ([]string, []string, error) {
	operator, err := access.IsOperatorUser(ctx, pool, userID)
	if err != nil {
		return nil, nil, err
	}
	query := `
		SELECT l.id::text, l.name
		FROM locations l
		JOIN location_memberships lm ON lm.location_id = l.id
		WHERE lm.user_id = $1 AND lm.active = TRUE AND lm.role IN ('producer','director','crew')
		ORDER BY l.name`
	args := []any{userID}
	if operator {
		query = `SELECT l.id::text, l.name FROM locations l ORDER BY l.name`
		args = nil
	}
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var ids, names []string
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, nil, err
		}
		ids = append(ids, id)
		names = append(names, name)
	}
	return ids, names, rows.Err()
}

// LoadAuthoringTree builds the Writer's Room dashboard payload. Returns
// not_authorized when the user can author nowhere -- the venue shell shows
// the forbidden screen off this.
func LoadAuthoringTree(ctx context.Context, pool *pgxpool.Pool, userID string) (*AuthoringTree, error) {
	ids, names, err := authoringLocationIDs(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, errors.New("not_authorized")
	}

	tree := &AuthoringTree{RecentDrafts: []Publication{}}
	for i, locID := range ids {
		loc := TreeLocation{LocationID: locID, LocationName: names[i], Collections: []Collection{}, Publications: []Publication{}}

		rows, err := pool.Query(ctx, `
			SELECT id::text, location_id::text, COALESCE(parent_id::text, ''), kind, title, slug, summary, sort_order, visibility,
			       COALESCE(created_by::text, ''), created_at, updated_at
			FROM ewrite_collections WHERE location_id = $1
			ORDER BY sort_order, title`, locID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var c Collection
			if err := rows.Scan(&c.ID, &c.LocationID, &c.ParentID, &c.Kind, &c.Title, &c.Slug, &c.Summary, &c.SortOrder, &c.Visibility, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
				rows.Close()
				return nil, err
			}
			loc.Collections = append(loc.Collections, c)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}

		prows, err := pool.Query(ctx, `
			SELECT id::text, collection_id::text, location_id::text, title, slug, summary, word_count, status, visibility,
			       COALESCE(current_revision_id::text, ''), COALESCE(created_by::text, ''), COALESCE(updated_by::text, ''),
			       published_at, created_at, updated_at
			FROM ewrite_publications WHERE location_id = $1
			ORDER BY updated_at DESC`, locID)
		if err != nil {
			return nil, err
		}
		for prows.Next() {
			var p Publication
			if err := prows.Scan(&p.ID, &p.CollectionID, &p.LocationID, &p.Title, &p.Slug, &p.Summary, &p.WordCount, &p.Status, &p.Visibility,
				&p.CurrentRevisionID, &p.CreatedBy, &p.UpdatedBy, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
				prows.Close()
				return nil, err
			}
			loc.Publications = append(loc.Publications, p)
		}
		prows.Close()
		if err := prows.Err(); err != nil {
			return nil, err
		}

		tree.Locations = append(tree.Locations, loc)

		for _, p := range loc.Publications {
			if p.Status == "draft" && len(tree.RecentDrafts) < 10 {
				tree.RecentDrafts = append(tree.RecentDrafts, p)
			}
		}
	}
	return tree, nil
}

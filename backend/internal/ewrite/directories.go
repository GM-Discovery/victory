package ewrite

// Reusable directory abstraction (kernel Goal B, spec 5): compact
// navigational metadata for a dense rules domain, first proven by the
// Skill Directory (seed.go's EnsureSkillDirectory). Directory management
// (Crew+ create/curate) mirrors the object-link authority split in
// links.go: CanAuthorInScope at the directory's own location, CanEditPublication
// on whatever target an entry is being pointed at.

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ListDirectoriesForCollection lists directories under a Ruleset (or any
// collection) for the Library/Writer's Room landing surfaces, with a cheap
// entry count for the compact-index affordance.
func ListDirectoriesForCollection(ctx context.Context, pool queryer, collectionID string) ([]Directory, error) {
	rows, err := pool.Query(ctx, `
		SELECT d.id::text, d.collection_id::text, d.directory_type, d.title, d.slug, d.summary, d.created_at, d.updated_at,
		       (SELECT COUNT(*) FROM ewrite_directory_entries e WHERE e.directory_id = d.id)
		FROM ewrite_directories d
		WHERE d.collection_id = $1
		ORDER BY d.title
	`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Directory{}
	for rows.Next() {
		var d Directory
		if err := rows.Scan(&d.ID, &d.CollectionID, &d.DirectoryType, &d.Title, &d.Slug, &d.Summary, &d.CreatedAt, &d.UpdatedAt, &d.EntryCount); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// LoadDirectory loads one directory by ID, along with the location it's
// scoped to (via its collection) so callers can run authority checks.
func LoadDirectory(ctx context.Context, pool queryer, id string) (*Directory, string, error) {
	var d Directory
	var locationID string
	err := pool.QueryRow(ctx, `
		SELECT d.id::text, d.collection_id::text, d.directory_type, d.title, d.slug, d.summary, d.created_at, d.updated_at, c.location_id::text
		FROM ewrite_directories d
		JOIN ewrite_collections c ON c.id = d.collection_id
		WHERE d.id = $1
	`, strings.TrimSpace(id)).Scan(&d.ID, &d.CollectionID, &d.DirectoryType, &d.Title, &d.Slug, &d.Summary, &d.CreatedAt, &d.UpdatedAt, &locationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", errors.New("directory_not_found")
	}
	if err != nil {
		return nil, "", err
	}
	return &d, locationID, nil
}

// ListDirectoryEntries returns a directory's entries, optionally filtered
// by a case-insensitive substring match against name/aliases and an exact
// category match. Every entry's target link is resolved through
// CanReadPublication for the requesting user -- an entry whose target
// exists but is unreadable comes back "hidden" with no target fields at
// all (spec 5.3: a directory must not reveal a hidden target's title).
func ListDirectoryEntries(ctx context.Context, pool *pgxpool.Pool, actorUserID, directoryID, search, category string) ([]DirectoryEntry, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return nil, errors.New("not_authenticated")
	}
	directoryID = strings.TrimSpace(directoryID)
	if directoryID == "" {
		return nil, errors.New("directory_id_required")
	}

	rows, err := pool.Query(ctx, `
		SELECT e.id::text, e.directory_id::text, e.external_ref, e.canonical_name, e.aliases, e.compact_summary, e.category, e.sort_key,
		       COALESCE(e.target_publication_id::text, ''), COALESCE(e.target_section_id::text, '')
		FROM ewrite_directory_entries e
		WHERE e.directory_id = $1
		  AND ($2 = '' OR e.category = $2)
		  AND ($3 = '' OR e.canonical_name ILIKE '%' || $3 || '%' OR EXISTS (
		        SELECT 1 FROM unnest(e.aliases) a WHERE a ILIKE '%' || $3 || '%'
		      ))
		ORDER BY e.canonical_name, e.sort_key
	`, directoryID, strings.TrimSpace(category), strings.TrimSpace(search))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []DirectoryEntry{}
	targetPubIDs := map[string]bool{}
	for rows.Next() {
		var e DirectoryEntry
		if err := rows.Scan(&e.ID, &e.DirectoryID, &e.ExternalRef, &e.CanonicalName, &e.Aliases, &e.CompactSummary, &e.Category, &e.SortKey, &e.TargetPublicationID, &e.TargetSectionID); err != nil {
			return nil, err
		}
		if e.TargetPublicationID != "" {
			targetPubIDs[e.TargetPublicationID] = true
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Resolve each distinct target publication's readability once, not
	// once per entry -- the Skill Directory's 100 entries mostly share one
	// publication (Core Rulebook).
	readable := map[string]*Publication{}
	for pubID := range targetPubIDs {
		p, err := LoadPublication(ctx, pool, pubID)
		if err != nil {
			continue
		}
		if ok, err := CanReadPublication(ctx, pool, actorUserID, p); err == nil && ok {
			readable[pubID] = p
		}
	}

	sectionIDs := []string{}
	for _, e := range entries {
		if e.TargetSectionID != "" && readable[e.TargetPublicationID] != nil {
			sectionIDs = append(sectionIDs, e.TargetSectionID)
		}
	}
	sections := map[string]Section{}
	if len(sectionIDs) > 0 {
		srows, err := pool.Query(ctx, `SELECT id::text, anchor, title FROM ewrite_sections WHERE id = ANY($1)`, sectionIDs)
		if err != nil {
			return nil, err
		}
		for srows.Next() {
			var id, anchor, title string
			if err := srows.Scan(&id, &anchor, &title); err != nil {
				srows.Close()
				return nil, err
			}
			sections[id] = Section{Anchor: anchor, Title: title}
		}
		srows.Close()
	}

	for i := range entries {
		e := &entries[i]
		switch {
		case e.TargetPublicationID == "":
			e.LinkStatus = "unlinked"
		case readable[e.TargetPublicationID] == nil:
			e.LinkStatus = "hidden"
			e.TargetPublicationID = ""
			e.TargetSectionID = ""
		default:
			e.LinkStatus = "linked"
			e.TargetPublicationTitle = readable[e.TargetPublicationID].Title
			if s, ok := sections[e.TargetSectionID]; ok {
				e.TargetSectionAnchor = s.Anchor
				e.TargetSectionTitle = s.Title
			} else {
				e.TargetSectionID = ""
			}
		}
	}

	return entries, nil
}

// SetDirectoryEntryLink assigns (or clears, when publicationID is empty)
// the curated target for one directory entry. Authority is a dual check
// like SetEquipmentItemRuleLink: Crew+ at the directory's own location
// (may curate this directory) AND edit authority on the target publication
// (may point readers at this rule) -- an author must not wire another
// production's private draft into a public directory.
func SetDirectoryEntryLink(ctx context.Context, pool *pgxpool.Pool, actorUserID, entryID, publicationID, sectionID string) (*DirectoryEntry, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	entryID = strings.TrimSpace(entryID)
	publicationID = strings.TrimSpace(publicationID)
	sectionID = strings.TrimSpace(sectionID)
	if actorUserID == "" {
		return nil, errors.New("not_authenticated")
	}
	if entryID == "" {
		return nil, errors.New("entry_id_required")
	}

	var directoryID string
	if err := pool.QueryRow(ctx, `SELECT directory_id::text FROM ewrite_directory_entries WHERE id = $1`, entryID).Scan(&directoryID); errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("entry_not_found")
	} else if err != nil {
		return nil, err
	}

	_, directoryLocationID, err := LoadDirectory(ctx, pool, directoryID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanAuthorInScope(ctx, pool, actorUserID, directoryLocationID); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("not_authorized")
	}

	if publicationID == "" {
		if _, err := pool.Exec(ctx, `
			UPDATE ewrite_directory_entries SET target_publication_id = NULL, target_section_id = NULL, updated_at = NOW()
			WHERE id = $1
		`, entryID); err != nil {
			return nil, err
		}
	} else {
		p, err := LoadPublication(ctx, pool, publicationID)
		if err != nil {
			return nil, err
		}
		if ok, err := CanEditPublication(ctx, pool, actorUserID, p); err != nil {
			return nil, err
		} else if !ok {
			return nil, errors.New("not_authorized")
		}
		if sectionID != "" {
			var sectionPub string
			err := pool.QueryRow(ctx, `SELECT publication_id::text FROM ewrite_sections WHERE id = $1`, sectionID).Scan(&sectionPub)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, errors.New("section_not_found")
			}
			if err != nil {
				return nil, err
			}
			if sectionPub != p.ID {
				return nil, errors.New("section_publication_mismatch")
			}
		}
		if _, err := pool.Exec(ctx, `
			UPDATE ewrite_directory_entries
			SET target_publication_id = $2, target_section_id = NULLIF($3, '')::uuid, updated_at = NOW()
			WHERE id = $1
		`, entryID, p.ID, sectionID); err != nil {
			return nil, err
		}
	}

	entries, err := ListDirectoryEntries(ctx, pool, actorUserID, directoryID, "", "")
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].ID == entryID {
			return &entries[i], nil
		}
	}
	return nil, errors.New("entry_not_found")
}

// RuleLinksForCharacterSkills resolves Skill Directory rule links for a set
// of catalogue skill IDs (characters.Chapter4Skill.ID) in one query,
// mirroring RuleLinksForEquipmentItems's shallow status-only filter: this
// is deep-link metadata attached to an already-authority-gated payload
// (the caller already ran CanEditCard), not protected content itself -- the
// reader route re-enforces real access when the link is opened.
func RuleLinksForCharacterSkills(ctx context.Context, pool queryer, skillIDs []string) (map[string]RuleLink, error) {
	out := map[string]RuleLink{}
	if len(skillIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT e.external_ref, p.id::text, p.title, COALESCE(s.anchor, ''), COALESCE(s.title, '')
		FROM ewrite_directory_entries e
		JOIN ewrite_directories d ON d.id = e.directory_id AND d.directory_type = 'skill'
		JOIN ewrite_publications p ON p.id = e.target_publication_id
		LEFT JOIN ewrite_sections s ON s.id = e.target_section_id
		WHERE e.external_ref = ANY($1) AND p.status = 'published'
	`, skillIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var skillID string
		var l RuleLink
		if err := rows.Scan(&skillID, &l.PublicationID, &l.PublicationTitle, &l.SectionAnchor, &l.SectionTitle); err != nil {
			return nil, err
		}
		out[skillID] = l
	}
	return out, rows.Err()
}

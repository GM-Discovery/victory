package ewrite

// Section reconciliation: turn a render outline into stable ewrite_sections
// rows. The contract (Construction/eWrite/ewrite-link-anchor-contract.md):
//
//  1. Existing sections are matched by ANCHOR first -- an unchanged anchor
//     keeps its row UUID, so object links pointing at the section survive
//     every ordinary edit.
//  2. Leftover old sections are matched to leftover new headings by
//     (title, level) in document order -- this is the "title kept, anchor
//     changed" case (e.g. an explicit {#id} was added). The row is updated
//     in place to the new anchor and the OLD anchor becomes an alias
//     pointing at the same section, so previously-copied links keep
//     working (spec 9.1 "prior anchor may redirect or alias").
//  3. Unmatched new headings insert new rows; unmatched old sections are
//     deleted (their aliases cascade away with them).
//  4. An alias colliding with a live anchor is dropped -- live wins,
//     nothing silently redirects to unrelated content (spec 9.4).
//
// Runs inside the caller's save/publish transaction.

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type sectionRow struct {
	id     string
	title  string
	level  int
	anchor string
}

func reconcileSections(ctx context.Context, tx pgx.Tx, publicationID string, outline []Heading) error {
	rows, err := tx.Query(ctx, `
		SELECT id::text, title, heading_level, anchor
		FROM ewrite_sections WHERE publication_id = $1
		ORDER BY sort_order`, publicationID)
	if err != nil {
		return err
	}
	var existing []sectionRow
	for rows.Next() {
		var s sectionRow
		if err := rows.Scan(&s.id, &s.title, &s.level, &s.anchor); err != nil {
			rows.Close()
			return err
		}
		existing = append(existing, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	byAnchor := make(map[string]*sectionRow, len(existing))
	for i := range existing {
		byAnchor[existing[i].anchor] = &existing[i]
	}

	matched := make(map[string]bool)        // section id -> matched
	sectionIDs := make([]string, len(outline)) // outline index -> section id
	type aliasAdd struct {
		oldAnchor string
		sectionID string
	}
	var aliases []aliasAdd

	// Pass 1: anchor matches.
	for i, h := range outline {
		if s, ok := byAnchor[h.Anchor]; ok && !matched[s.id] {
			sectionIDs[i] = s.id
			matched[s.id] = true
		}
	}

	// Pass 2: (title, level) matches in document order -> anchor changed.
	for i, h := range outline {
		if sectionIDs[i] != "" {
			continue
		}
		for j := range existing {
			s := &existing[j]
			if matched[s.id] || s.title != h.Title || s.level != h.Level {
				continue
			}
			sectionIDs[i] = s.id
			matched[s.id] = true
			aliases = append(aliases, aliasAdd{oldAnchor: s.anchor, sectionID: s.id})
			break
		}
	}

	// Delete unmatched old sections (aliases cascade).
	for _, s := range existing {
		if !matched[s.id] {
			if _, err := tx.Exec(ctx, `DELETE FROM ewrite_sections WHERE id = $1`, s.id); err != nil {
				return err
			}
		}
	}

	// Upsert matched/new rows with fresh parentage and ordering. Parent is
	// the nearest preceding heading with a smaller level.
	//
	// Two-step anchor write: matched rows first move to a temporary
	// collision-proof anchor, then take their final one -- otherwise a pure
	// anchor swap between two sections would trip the UNIQUE(publication_id,
	// anchor) constraint mid-flight.
	for i := range outline {
		if sectionIDs[i] == "" {
			continue
		}
		if _, err := tx.Exec(ctx,
			`UPDATE ewrite_sections SET anchor = 'ewrite-tmp-' || id::text WHERE id = $1`,
			sectionIDs[i]); err != nil {
			return err
		}
	}

	parentFor := func(i int) string {
		for j := i - 1; j >= 0; j-- {
			if outline[j].Level < outline[i].Level {
				return sectionIDs[j]
			}
		}
		return ""
	}

	for i, h := range outline {
		if sectionIDs[i] != "" {
			if _, err := tx.Exec(ctx, `
				UPDATE ewrite_sections
				SET title = $2, heading_level = $3, anchor = $4, anchor_explicit = $5,
				    sort_order = $6, parent_section_id = NULLIF($7, '')::uuid, updated_at = NOW()
				WHERE id = $1
			`, sectionIDs[i], h.Title, h.Level, h.Anchor, h.Explicit, i, parentFor(i)); err != nil {
				return err
			}
			continue
		}
		var id string
		if err := tx.QueryRow(ctx, `
			INSERT INTO ewrite_sections (publication_id, parent_section_id, heading_level, title, anchor, anchor_explicit, sort_order)
			VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7)
			RETURNING id::text
		`, publicationID, parentFor(i), h.Level, h.Title, h.Anchor, h.Explicit, i).Scan(&id); err != nil {
			return err
		}
		sectionIDs[i] = id
	}

	// Record aliases for changed anchors; skip any alias that now collides
	// with a live anchor (live wins).
	for _, a := range aliases {
		var live bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM ewrite_sections WHERE publication_id = $1 AND anchor = $2)`,
			publicationID, a.oldAnchor).Scan(&live); err != nil {
			return err
		}
		if live {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO ewrite_anchor_aliases (publication_id, alias_anchor, section_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (publication_id, alias_anchor) DO UPDATE SET section_id = EXCLUDED.section_id
		`, publicationID, a.oldAnchor, a.sectionID); err != nil {
			return err
		}
	}

	// Drop aliases that a live anchor has since reclaimed.
	if _, err := tx.Exec(ctx, `
		DELETE FROM ewrite_anchor_aliases aa
		USING ewrite_sections s
		WHERE aa.publication_id = $1
		  AND s.publication_id = aa.publication_id
		  AND s.anchor = aa.alias_anchor
	`, publicationID); err != nil {
		return err
	}

	return nil
}

// LoadSections returns a publication's section rows in document order.
func LoadSections(ctx context.Context, pool queryer, publicationID string) ([]Section, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, publication_id::text, COALESCE(parent_section_id::text, ''), heading_level, title, anchor, anchor_explicit, sort_order
		FROM ewrite_sections WHERE publication_id = $1
		ORDER BY sort_order`, publicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sections := []Section{}
	for rows.Next() {
		var s Section
		if err := rows.Scan(&s.ID, &s.PublicationID, &s.ParentSectionID, &s.HeadingLevel, &s.Title, &s.Anchor, &s.AnchorExplicit, &s.SortOrder); err != nil {
			return nil, err
		}
		sections = append(sections, s)
	}
	return sections, rows.Err()
}

// ResolveAnchor resolves a requested anchor for a publication: exact
// section anchor first, then alias. Returns the live anchor and true when
// found.
func ResolveAnchor(ctx context.Context, pool queryer, publicationID, anchor string) (string, bool, error) {
	var live string
	err := pool.QueryRow(ctx, `
		SELECT anchor FROM ewrite_sections WHERE publication_id = $1 AND anchor = $2
	`, publicationID, anchor).Scan(&live)
	if err == nil {
		return live, true, nil
	}
	err = pool.QueryRow(ctx, `
		SELECT s.anchor
		FROM ewrite_anchor_aliases aa
		JOIN ewrite_sections s ON s.id = aa.section_id
		WHERE aa.publication_id = $1 AND aa.alias_anchor = $2
	`, publicationID, anchor).Scan(&live)
	if err == nil {
		return live, true, nil
	}
	return "", false, nil
}

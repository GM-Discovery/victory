package ewrite

// Canonical Socio manuscript seed. Victory ships with real content: every
// fresh install gets the Socio v1.1 core rulebook in the Library, not an
// empty shelf ("you install Victory, you get Socio").
//
// This is Go boot-time content seeding, not a migration -- like
// EnsureKernel16VenueSurface, it is legal DML outside backend/internal/migrate
// (the "no DDL outside migrate" rule doesn't apply; this only inserts rows).
// It has to be Go, not raw SQL: rendered_html/search_text/sections are a
// cache that must come from the real Render() pipeline (goldmark+bluemonday),
// never hand-written into a migration -- the same rule that keeps every
// ordinary save honest applies to seeded content too.
//
// Idempotency is deliberately create-if-absent-ONLY. Once the publication
// exists, this function never touches it again on any later boot. A
// Producer fixing the manuscript's known 4d12/5d12 progression error (or
// any other edit) in the Writer's Room must never be silently reverted by
// a redeploy.
//
// created_by/updated_by are left NULL: the seed is attributed to no
// particular human. CanEditPublication's manager-role branch (Producer/
// Director at the location) still grants edit authority, so any Producer
// can pick it up and maintain it -- exactly the authority a hand-imported
// publication would have if its original author later left.

import (
	_ "embed"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed seed/socio-v1.1.md
var socioManuscriptSource string

const (
	// Every Victory install gets this location unconditionally from
	// migration 002_seed_world.sql -- it is the one canonical default
	// location for the install, not data specific to any one operator.
	socioSeedLocationSlug = "amurray-family"

	socioSeedRulesetSlug  = "socio-stories-of-us"
	socioSeedRulesetTitle = "Socio: Stories of Us"

	socioSeedPublicationSlug  = "core-rulebook"
	socioSeedPublicationTitle = "Core Rulebook"
)

// EnsureCanonicalSocioManuscript seeds the Socio v1.1 core rulebook as a
// published, public-visibility eWriting on first boot. Safe to call on
// every boot: after the first success it is a single indexed existence
// check and nothing more.
func EnsureCanonicalSocioManuscript(ctx context.Context, pool *pgxpool.Pool) error {
	var locationID string
	err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = $1`, socioSeedLocationSlug).Scan(&locationID)
	if errors.Is(err, pgx.ErrNoRows) {
		// No canonical location yet (shouldn't happen post-migration, but
		// this runs at boot on every install) -- nothing to seed onto.
		return nil
	}
	if err != nil {
		return err
	}

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM ewrite_publications p
			JOIN ewrite_collections c ON c.id = p.collection_id
			WHERE c.location_id = $1 AND c.slug = $2 AND p.slug = $3
		)
	`, locationID, socioSeedRulesetSlug, socioSeedPublicationSlug).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}

	res, err := Render(socioManuscriptSource)
	if err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var collectionID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO ewrite_collections (location_id, kind, title, slug, summary, visibility)
		VALUES ($1, 'ruleset', $2, $3, $4, 'public')
		ON CONFLICT (location_id, slug) DO UPDATE SET title = EXCLUDED.title
		RETURNING id::text
	`, locationID, socioSeedRulesetTitle, socioSeedRulesetSlug,
		"An open-source, collaborative tabletop role-playing system. Creative Commons Attribution (CC BY).",
	).Scan(&collectionID); err != nil {
		return err
	}

	var publicationID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO ewrite_publications
			(collection_id, location_id, title, slug, summary, source_markdown, rendered_html, search_text, word_count, status, visibility, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'published', 'public', NOW())
		RETURNING id::text
	`, collectionID, locationID, socioSeedPublicationTitle, socioSeedPublicationSlug,
		"The full Socio- v1.1 core rulebook.", socioManuscriptSource, res.HTML, res.PlainText, res.WordCount,
	).Scan(&publicationID); err != nil {
		return err
	}

	var revisionID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO ewrite_revisions (publication_id, revision_number, source_markdown)
		VALUES ($1, 1, $2)
		RETURNING id::text
	`, publicationID, socioManuscriptSource).Scan(&revisionID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE ewrite_publications SET current_revision_id = $2 WHERE id = $1
	`, publicationID, revisionID); err != nil {
		return err
	}

	if err := reconcileSections(ctx, tx, publicationID, res.Outline); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

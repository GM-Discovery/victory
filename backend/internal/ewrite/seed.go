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
	"context"
	_ "embed"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed seed/socio-v1.1.md
var socioManuscriptSource string

//go:embed seed/quickstart-v1.md
var quickstartManuscriptSource string

//go:embed seed/niava-v1.md
var niavaManuscriptSource string

const (
	// Every Victory install gets this location unconditionally from
	// migration 002_seed_world.sql -- it is the one canonical default
	// location for the install, not data specific to any one operator.
	socioSeedLocationSlug = "amurray-family"

	socioSeedRulesetSlug  = "socio-stories-of-us"
	socioSeedRulesetTitle = "Socio: Stories of Us"

	socioSeedPublicationSlug  = "core-rulebook"
	socioSeedPublicationTitle = "Core Rulebook"

	// Kernel 79: the three Series the ruleset organizes into. The
	// Core Rulebook Series shares its slug with the pre-existing Core
	// Rulebook Publication -- harmless, since collections and
	// publications have independent UNIQUE scopes (location_id, slug)
	// vs (collection_id, slug).
	socioCoreRulebookSeriesSlug  = "core-rulebook"
	socioCoreRulebookSeriesTitle = "Core Rulebook"

	socioQuickstartSeriesSlug  = "quickstart"
	socioQuickstartSeriesTitle = "Quickstart"

	socioQuickstartPublicationSlug  = "the-locked-courtyard"
	socioQuickstartPublicationTitle = "Socio-: The Locked Courtyard & Beyond"

	socioNiavaSeriesSlug  = "niava"
	socioNiavaSeriesTitle = "Niava"

	socioNiavaPublicationSlug  = "niava-setting-supplement"
	socioNiavaPublicationTitle = "Niava Setting Supplement"
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

	// Checked by ewrite_publications.location_id + slug directly (that
	// column is denormalized from the collection root, migration 084) --
	// NOT by joining through a collection with the ruleset's slug. Kernel
	// 79's EnsureSocioSeriesHierarchy reparents this publication out from
	// directly-under-the-ruleset into the Core Rulebook Series right after
	// this function runs; a join-based check anchored to the ruleset's
	// slug would stop recognizing the already-seeded publication on every
	// subsequent boot and attempt to recreate it.
	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM ewrite_publications WHERE location_id = $1 AND slug = $2
		)
	`, locationID, socioSeedPublicationSlug).Scan(&exists); err != nil {
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

// EnsureSocioSeriesHierarchy creates the Core Rulebook / Quickstart / Niava
// Series under the canonical Socio ruleset (Kernel 79 spec 1.2), and moves
// the existing Core Rulebook publication -- seeded directly under the
// ruleset root by EnsureCanonicalSocioManuscript, before this kernel --
// into its new Series. The publication's ID, slug, revisions, sections/
// anchors, and ewrite_object_links rows are all untouched: only
// collection_id moves, so every existing equipment "View rule ->" link
// keeps resolving exactly as before.
//
// Safe to call on every boot: the reparent only matches while
// collection_id still equals the ruleset root (a one-time move), and the
// Series INSERTs are ON CONFLICT no-ops thereafter.
func EnsureSocioSeriesHierarchy(ctx context.Context, pool *pgxpool.Pool) error {
	var locationID string
	err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = $1`, socioSeedLocationSlug).Scan(&locationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	var rulesetID string
	err = pool.QueryRow(ctx, `
		SELECT id::text FROM ewrite_collections WHERE location_id = $1 AND slug = $2 AND kind = 'ruleset'
	`, locationID, socioSeedRulesetSlug).Scan(&rulesetID)
	if errors.Is(err, pgx.ErrNoRows) {
		// The ruleset itself hasn't been seeded yet -- boot runs
		// EnsureCanonicalSocioManuscript first, but guard anyway.
		return nil
	}
	if err != nil {
		return err
	}

	// sortOrder fixes reading order (Core Rulebook, then Quickstart, then
	// the Niava setting supplement). Without it every row ties at the
	// column default of 0 and ewrite_collections' listing queries
	// (`ORDER BY sort_order, title`) fall back to alphabetical title --
	// "Niava" sorts before "Quickstart", which is not the intended order.
	series := []struct {
		slug, title, summary string
		sortOrder            int
	}{
		{socioCoreRulebookSeriesSlug, socioCoreRulebookSeriesTitle, "The complete Socio- v1.1 core rulebook.", 0},
		{socioQuickstartSeriesSlug, socioQuickstartSeriesTitle, "A complete starter guide to Stories of Us in the world of Niava.", 1},
		{socioNiavaSeriesSlug, socioNiavaSeriesTitle, "The Niava setting supplement.", 2},
	}

	var coreRulebookSeriesID string
	for _, s := range series {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO ewrite_collections (location_id, parent_id, kind, title, slug, summary, sort_order, visibility)
			VALUES ($1, $2, 'series', $3, $4, $5, $6, 'public')
			ON CONFLICT (location_id, slug) DO UPDATE SET title = EXCLUDED.title, sort_order = EXCLUDED.sort_order
			RETURNING id::text
		`, locationID, rulesetID, s.title, s.slug, s.summary, s.sortOrder).Scan(&id); err != nil {
			return err
		}
		if s.slug == socioCoreRulebookSeriesSlug {
			coreRulebookSeriesID = id
		}
	}

	if _, err := pool.Exec(ctx, `
		UPDATE ewrite_publications SET collection_id = $1
		WHERE collection_id = $2 AND slug = $3
	`, coreRulebookSeriesID, rulesetID, socioSeedPublicationSlug); err != nil {
		return err
	}

	return nil
}

// ensureSeriesPublication seeds a publication (with its first revision,
// sections, and asset refs) under an existing Series collection if one
// with the given slug doesn't already exist there. Shared by
// EnsureQuickstartManuscript and EnsureNiavaManuscript;
// EnsureCanonicalSocioManuscript predates this helper and is left exactly
// as originally written above.
func ensureSeriesPublication(ctx context.Context, pool *pgxpool.Pool, collectionID, locationID, title, slug, summary, source string) error {
	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM ewrite_publications WHERE collection_id = $1 AND slug = $2)
	`, collectionID, slug).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}

	res, err := Render(source)
	if err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var publicationID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO ewrite_publications
			(collection_id, location_id, title, slug, summary, source_markdown, rendered_html, search_text, word_count, status, visibility, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'published', 'public', NOW())
		RETURNING id::text
	`, collectionID, locationID, title, slug, summary, source, res.HTML, res.PlainText, res.WordCount,
	).Scan(&publicationID); err != nil {
		return err
	}

	var revisionID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO ewrite_revisions (publication_id, revision_number, source_markdown)
		VALUES ($1, 1, $2)
		RETURNING id::text
	`, publicationID, source).Scan(&revisionID); err != nil {
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
	if err := reconcilePublicationAssetRefs(ctx, tx, publicationID, res.ImageRefs); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// EnsureQuickstartManuscript seeds "Socio-: The Locked Courtyard & Beyond"
// under the Quickstart Series. Requires EnsureSocioSeriesHierarchy to have
// run first (boot order in main.go); if the Series doesn't exist yet this
// is a safe no-op, not an error.
func EnsureQuickstartManuscript(ctx context.Context, pool *pgxpool.Pool) error {
	var locationID string
	err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = $1`, socioSeedLocationSlug).Scan(&locationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	var collectionID string
	err = pool.QueryRow(ctx, `
		SELECT id::text FROM ewrite_collections WHERE location_id = $1 AND slug = $2
	`, locationID, socioQuickstartSeriesSlug).Scan(&collectionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	return ensureSeriesPublication(ctx, pool, collectionID, locationID,
		socioQuickstartPublicationTitle, socioQuickstartPublicationSlug,
		"A complete starter guide to Stories of Us in the world of Niava.",
		quickstartManuscriptSource)
}

// EnsureNiavaManuscript seeds "Niava Setting Supplement" under the Niava
// Series. Same boot-order dependency and no-op-if-absent behavior as
// EnsureQuickstartManuscript.
func EnsureNiavaManuscript(ctx context.Context, pool *pgxpool.Pool) error {
	var locationID string
	err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = $1`, socioSeedLocationSlug).Scan(&locationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	var collectionID string
	err = pool.QueryRow(ctx, `
		SELECT id::text FROM ewrite_collections WHERE location_id = $1 AND slug = $2
	`, locationID, socioNiavaSeriesSlug).Scan(&collectionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	return ensureSeriesPublication(ctx, pool, collectionID, locationID,
		socioNiavaPublicationTitle, socioNiavaPublicationSlug,
		"The Niava setting supplement: gazetteer, world dynamics, the plot, NPC reference, and the Nianic language primer.",
		niavaManuscriptSource)
}

package ewrite

// Publication <-> asset reference reconciliation (Kernel 79 Goal D: close
// the Kernel 78 embedded-image visibility gap). Render() already extracts
// every image URL a publication's Markdown references (RenderResult.
// ImageRefs); this file turns that into stable ewrite_publication_assets
// rows so backend/internal/assets can gate an eWrite-bound asset through
// the publication's own CanReadPublication resolution instead of generic
// location membership.
//
// Reconciliation is delete-then-insert per publication, mirroring
// reconcileSections's "small N, simplicity over micro-optimization" idiom
// -- a publication typically embeds a handful of images, not thousands.

import (
	"context"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// assetImageRefPattern captures the asset UUID out of an ImageRef.
// Anything that doesn't match (an external URL, a malformed src) is
// silently ignored here -- bluemonday's assetImageSrcPattern already
// strips it from rendered_html, so no such reference is ever actually
// served as an image regardless.
var assetImageRefPattern = regexp.MustCompile(`^/api/assets/([0-9a-fA-F-]{36})/content(?:\?variant=[a-z0-9]+)?$`)

// reconcilePublicationAssetRefs runs inside the caller's save/publish/seed
// transaction -- the same transaction that writes source_markdown, so the
// reference set can never go stale relative to the content it describes.
func reconcilePublicationAssetRefs(ctx context.Context, tx pgx.Tx, publicationID string, imageRefs []string) error {
	seen := make(map[string]bool, len(imageRefs))
	assetIDs := make([]string, 0, len(imageRefs))
	for _, ref := range imageRefs {
		m := assetImageRefPattern.FindStringSubmatch(ref)
		if m == nil || seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		assetIDs = append(assetIDs, m[1])
	}

	if _, err := tx.Exec(ctx, `DELETE FROM ewrite_publication_assets WHERE publication_id = $1`, publicationID); err != nil {
		return err
	}
	for _, assetID := range assetIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO ewrite_publication_assets (publication_id, asset_id)
			VALUES ($1, $2::uuid)
			ON CONFLICT (publication_id, asset_id) DO NOTHING
		`, publicationID, assetID); err != nil {
			return err
		}
	}
	return nil
}

// BackfillPublicationAssetRefs populates ewrite_publication_assets for any
// publication that predates this table (the live Core Rulebook, seeded
// under Kernel 78, and any publication hand-authored before this kernel
// deployed). Idempotent by construction: a publication with existing rows
// is skipped, and the ordinary save path (revisions.go) keeps rows current
// from here on, so this only ever does real work once per publication.
func BackfillPublicationAssetRefs(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, `
		SELECT p.id::text, p.source_markdown
		FROM ewrite_publications p
		WHERE NOT EXISTS (
			SELECT 1 FROM ewrite_publication_assets epa WHERE epa.publication_id = p.id
		)
	`)
	if err != nil {
		return err
	}
	type pending struct {
		id     string
		source string
	}
	var work []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.source); err != nil {
			rows.Close()
			return err
		}
		work = append(work, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, p := range work {
		res, err := Render(p.source)
		if err != nil {
			return err
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if err := reconcilePublicationAssetRefs(ctx, tx, p.id, res.ImageRefs); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

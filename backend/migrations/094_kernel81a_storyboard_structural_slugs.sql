-- Kernel 81A: deterministic serialized slugs for Storyboard structural
-- objects (columns, bands, rows-within-a-band), so duplicate human-facing
-- labels (e.g. two columns both titled "Scene") never produce an
-- ambiguous serialized identifier in export or any future integration
-- that needs a stable, readable key alongside the UUID.
--
-- `slug` is nullable at the schema level on purpose: this migration runs
-- before the Go-side backfill bootstrap
-- (storyboards.EnsureKernel81AStoryboardSlugsSurface, main.go), so any
-- pre-existing row is briefly NULL between migration-apply and the next
-- server boot completing. Application code always sets slug at creation
-- time going forward; nothing in this migration touches existing titles,
-- labels, IDs, or ordering.
--
-- The unique indexes are partial (`WHERE slug IS NOT NULL`) so the brief
-- NULL window above can never violate uniqueness -- multiple NULLs are
-- always permitted by a partial index that excludes them entirely, unlike
-- a plain UNIQUE constraint where Postgres already treats NULL <> NULL
-- (either would technically work; partial is the more explicit, more
-- honest statement of intent here).
--
-- Row slugs are scoped to (band_id, slug), not the whole board -- matching
-- the existing sort_order_in_band precedent that rows are fundamentally
-- band-scoped, and matching the explicit requirement that row uniqueness
-- is "within a band," not board-wide.
BEGIN;

ALTER TABLE storyboard_columns ADD COLUMN IF NOT EXISTS slug TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_storyboard_columns_storyboard_slug
  ON storyboard_columns(storyboard_id, slug)
  WHERE slug IS NOT NULL;

ALTER TABLE storyboard_bands ADD COLUMN IF NOT EXISTS slug TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_storyboard_bands_storyboard_slug
  ON storyboard_bands(storyboard_id, slug)
  WHERE slug IS NOT NULL;

ALTER TABLE storyboard_rows ADD COLUMN IF NOT EXISTS slug TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_storyboard_rows_band_slug
  ON storyboard_rows(band_id, slug)
  WHERE slug IS NOT NULL;

COMMIT;

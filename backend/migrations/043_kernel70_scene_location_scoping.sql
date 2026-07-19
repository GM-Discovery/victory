BEGIN;

-- Kernel 70: correct Kernel 69's overly narrow Scene ownership rule. A
-- Scene is a reusable standing set comparable to a studio-lot backlot
-- street -- it may originate in one Production, then be redressed and
-- reused by other Productions at the same Victory location. Scene reuse
-- becomes location-scoped instead of Production-exclusive; the originating
-- Production is preserved only as optional provenance.
--
-- This migration is additive/renaming only -- no Scene or Show Scene
-- Placement row is ever deleted, duplicated, or reassigned.
--
-- The rename step below (production_id -> source_production_id) is not
-- naturally idempotent (there is no `RENAME COLUMN IF EXISTS` in
-- PostgreSQL), and this repo's setup/deploy scripts replay every migration
-- file from scratch on every run with no migrations-tracking table. The
-- whole backfill+rename+constraint-swap sequence is therefore wrapped in a
-- single PL/pgSQL block gated on "does scenes.production_id still exist" --
-- PostgreSQL only parses/resolves a statement's column references when
-- that branch actually executes, so on replay (production_id already
-- renamed) this entire block is skipped without ever touching the
-- now-nonexistent column name.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'scenes' AND column_name = 'production_id'
  ) THEN
    -- 1. Add location_id (nullable for now -- backfilled next, then locked
    --    NOT NULL once every row has a value).
    ALTER TABLE scenes ADD COLUMN IF NOT EXISTS location_id UUID REFERENCES locations(id) ON DELETE RESTRICT;

    -- 2. Backfill location_id from each Scene's Production.
    UPDATE scenes
    SET location_id = p.location_id
    FROM productions p
    WHERE scenes.production_id = p.id
      AND scenes.location_id IS NULL;

    -- 3. Guard: fail loudly rather than silently locking NOT NULL over gaps.
    IF EXISTS (SELECT 1 FROM scenes WHERE location_id IS NULL) THEN
      RAISE EXCEPTION 'kernel70 migration: some scenes rows have no resolvable location_id after backfill';
    END IF;

    ALTER TABLE scenes ALTER COLUMN location_id SET NOT NULL;

    -- 4. Guard: fail loudly if the coming location-scoped uniqueness swap
    --    would collide (two Scenes at the same location sharing a slug
    --    across different Productions). Not expected on a fresh Kernel 69
    --    install, but checked rather than assumed.
    IF EXISTS (
      SELECT 1 FROM (
        SELECT location_id, slug FROM scenes GROUP BY location_id, slug HAVING COUNT(*) > 1
      ) dupes
    ) THEN
      RAISE EXCEPTION 'kernel70 migration: (location_id, slug) collisions would violate the new location-scoped uniqueness constraint';
    END IF;

    -- 5. Rename production_id to source_production_id and make it
    --    optional -- the originating Production is remembered as
    --    provenance only, it no longer gates reuse (Kernel 70 SS1.2,
    --    SS3.1).
    ALTER TABLE scenes RENAME COLUMN production_id TO source_production_id;
    ALTER TABLE scenes ALTER COLUMN source_production_id DROP NOT NULL;
    ALTER TABLE scenes ALTER COLUMN source_production_id SET DEFAULT NULL;

    -- 6. Drop the old Production-scoped uniqueness constraint. The
    --    replacement location-scoped constraint is added unconditionally
    --    below, outside this guard, so it's created exactly once whether
    --    this branch ran this time or on an earlier run.
    ALTER TABLE scenes DROP CONSTRAINT IF EXISTS scenes_production_id_slug_key;
  END IF;
END $$;

-- Outside the production_id-existence guard so it applies on every run
-- regardless of whether the rename branch above executed this time.
ALTER TABLE scenes ADD COLUMN IF NOT EXISTS location_id UUID REFERENCES locations(id) ON DELETE RESTRICT;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'scenes_location_id_slug_key'
  ) THEN
    ALTER TABLE scenes ADD CONSTRAINT scenes_location_id_slug_key UNIQUE (location_id, slug);
  END IF;
END $$;

-- idx_scenes_production_id (created by migration 042) is intentionally
-- left alone rather than dropped and recreated under a new name:
-- PostgreSQL keeps an existing index's definition pointing at a renamed
-- column automatically (indexes reference columns by attnum internally,
-- not name), so it continues to index source_production_id correctly
-- under its old name. Migration 042 no longer creates that index on a
-- fresh install (see its own updated comment) -- this location index is
-- the one every install gets going forward.
CREATE INDEX IF NOT EXISTS idx_scenes_location_id ON scenes(location_id);

-- show_scene_placements is intentionally untouched -- placements stay
-- Show-scoped exactly as Kernel 69 defined them; only the reusable Scene's
-- own cross-Production reuse rule changes.

COMMIT;

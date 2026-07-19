BEGIN;

-- Kernel 69: Scene Library and Show Staging Model -- the first real
-- reusable play object. A Scene is a reusable authored/configured
-- Production-scoped object; a Show Scene Placement is the use of that
-- reusable Scene inside one specific Show, with its own ordering and
-- venue-override/audience-copy details. This is deliberately a two-layer
-- model (Scene, then Show Scene Placement) so a Scene can be staged in
-- multiple Shows under the same Production rather than trapped under one
-- Show -- see Construction/Dictionary.txt's "Scene" / "Show Scene
-- Placement" entries.
--
-- No `live` status and no capture/fly/transition/script-hyperlink system --
-- those are explicitly out of scope for this kernel (Construction/Kernels/
-- kernel-69... v0.1.md SS1.7, SS3).

CREATE TABLE IF NOT EXISTS scenes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  production_id UUID NOT NULL REFERENCES productions(id) ON DELETE RESTRICT,
  slug TEXT NOT NULL,
  title TEXT NOT NULL,
  short_title TEXT,
  default_venue_id UUID REFERENCES venues(id) ON DELETE SET NULL,
  audience_title TEXT,
  audience_summary TEXT,
  player_brief TEXT,
  director_notes TEXT,
  operator_notes TEXT,
  source_ref TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  config_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  archived_at TIMESTAMPTZ,
  CONSTRAINT scenes_status_check CHECK (status IN (
    'draft', 'ready', 'retired', 'archived'
  )),
  CONSTRAINT scenes_archived_matches_status CHECK (
    (status = 'archived' AND archived_at IS NOT NULL) OR
    (status <> 'archived' AND archived_at IS NULL)
  ),
  UNIQUE (production_id, slug)
);

-- idx_scenes_production_id used to be created here, indexing what was then
-- called scenes.production_id. Kernel 70 (043_kernel70_scene_location_
-- scoping.sql) renamed that column to source_production_id; PostgreSQL
-- keeps an existing index's definition pointing at a renamed column
-- automatically, so on a database that already has this index nothing
-- needs to happen here. But this repo's setup/deploy scripts replay every
-- migration file from scratch on every run (no migrations-tracking
-- table), and CREATE INDEX IF NOT EXISTS still fully resolves its column
-- list even when the name already exists -- so on replay, a line
-- referencing the pre-rename column name would fail with "column
-- production_id does not exist" even though nothing would have been
-- created. That line is removed here rather than kept and guarded, since
-- migration 043 already creates the location-scoped equivalent this Scene
-- reuse correction actually needs (idx_scenes_location_id); a first-run
-- database simply never gets a same-purpose provenance index, which is an
-- acceptable, non-destructive omission.
CREATE INDEX IF NOT EXISTS idx_scenes_status ON scenes(status);

-- A Show Scene Placement targets a specific Show; removal/archival of a
-- placement never touches the reusable Scene row it points at, and vice
-- versa (archiving a Scene marks it unavailable for *new* placements but
-- does not cascade-remove existing placements -- ON DELETE RESTRICT on
-- scene_id makes an accidental hard delete of a staged Scene impossible).
CREATE TABLE IF NOT EXISTS show_scene_placements (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
  scene_id UUID NOT NULL REFERENCES scenes(id) ON DELETE RESTRICT,
  venue_id UUID REFERENCES venues(id) ON DELETE SET NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  audience_title_override TEXT,
  audience_summary_override TEXT,
  director_notes_override TEXT,
  config_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  archived_at TIMESTAMPTZ,
  CONSTRAINT show_scene_placements_status_check CHECK (status IN (
    'draft', 'ready', 'retired', 'archived'
  )),
  CONSTRAINT show_scene_placements_archived_matches_status CHECK (
    (status = 'archived' AND archived_at IS NOT NULL) OR
    (status <> 'archived' AND archived_at IS NULL)
  ),
  UNIQUE (show_id, scene_id)
);

CREATE INDEX IF NOT EXISTS idx_show_scene_placements_show_id ON show_scene_placements(show_id);
CREATE INDEX IF NOT EXISTS idx_show_scene_placements_scene_id ON show_scene_placements(scene_id);

COMMIT;

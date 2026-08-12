BEGIN;

-- Kernel 87: Shared Cartographic Rendering & In-Character Play.
--
-- Canonical drawing objects live on the same map/stage coordinate space
-- already used by tokens/index cards (world_x/world_y "stage world"
-- pixels -- see frontend/lib/stage-runtime/geometry.js). Scope mirrors
-- Kernel 86's roll-audience model exactly (Show or Cohort, resolved via
-- backend/internal/rollaudience -- reused, not re-implemented): a drawing
-- object with cohort_id NULL is Show-scoped; a drawing object with
-- cohort_id set is visible only to that Cohort's members plus Director+.
--
-- geometry is a small JSON shape per object_type (documented in
-- Construction/Stage/drawing-object-contract.md), always expressed in the
-- same map-relative "world" coordinate space so pan/zoom/fullscreen/
-- Detail View never need to rewrite stored points.

CREATE TABLE IF NOT EXISTS drawing_objects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
  cohort_id UUID REFERENCES show_cohorts(id) ON DELETE CASCADE,
  creator_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  creator_character_id UUID REFERENCES character_cards(id) ON DELETE SET NULL,
  object_type TEXT NOT NULL CHECK (object_type IN (
    'freehand', 'line', 'polyline', 'rectangle', 'ellipse', 'polygon', 'text', 'stamp'
  )),
  geometry JSONB NOT NULL,
  stroke_color TEXT NOT NULL DEFAULT '#2b2622',
  fill_color TEXT,
  stroke_width DOUBLE PRECISION NOT NULL DEFAULT 3,
  opacity DOUBLE PRECISION NOT NULL DEFAULT 1 CHECK (opacity >= 0 AND opacity <= 1),
  line_style TEXT NOT NULL DEFAULT 'solid' CHECK (line_style IN ('solid', 'dashed', 'dotted')),
  rotation DOUBLE PRECISION NOT NULL DEFAULT 0,
  z_order INTEGER NOT NULL DEFAULT 0,
  locked BOOLEAN NOT NULL DEFAULT FALSE,
  text_content TEXT,
  stamp_key TEXT,
  deleted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_drawing_objects_show_id
  ON drawing_objects(show_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_drawing_objects_cohort_id
  ON drawing_objects(cohort_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_drawing_objects_creator
  ON drawing_objects(creator_user_id) WHERE deleted_at IS NULL;

-- One row per Show: who may currently draw (Director Only / Turn-Leader /
-- Freeform, kernel §9), plus the measured-tabletop physical scale and
-- diagonal policy (kernel §7). Show-scoped (not session-scoped) so it
-- survives session replacement, matching every other Show-owned
-- persistent config in this schema (shows.variables_json precedent).
CREATE TABLE IF NOT EXISTS stage_drawing_settings (
  show_id UUID PRIMARY KEY REFERENCES shows(id) ON DELETE CASCADE,
  drawing_mode TEXT NOT NULL DEFAULT 'director_only' CHECK (drawing_mode IN (
    'director_only', 'turn_leader', 'freeform'
  )),
  scale_grid_units DOUBLE PRECISION NOT NULL DEFAULT 1,
  scale_real_units DOUBLE PRECISION NOT NULL DEFAULT 5,
  scale_unit_label TEXT NOT NULL DEFAULT 'ft',
  diagonal_policy TEXT NOT NULL DEFAULT 'alternating_1_2' CHECK (diagonal_policy IN (
    'alternating_1_2', 'every_diagonal_1', 'euclidean'
  )),
  gridless_calibration JSONB,
  updated_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;

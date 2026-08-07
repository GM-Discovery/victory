-- Kernel 82: Storyboards Timeline Mode.
--
-- storyboards.mode/template_version record which built-in template (if
-- any) a board was instantiated from -- a snapshot fact recorded once at
-- creation, never a live link back to the template (spec 4.3: "Boards are
-- snapshots/instances after creation. They do not dynamically inherit
-- later template changes."). The built-in template itself is code-defined
-- Go seed data (timeline_template.go), not a DB row, per spec 4.2's
-- explicit "may be... code-defined seed data" option -- there is nothing
-- here for a board to reference back into, which is what makes "editing
-- an instance can never mutate the template" true by construction rather
-- than by convention.
--
-- storyboard_columns.column_role identifies Beginning/Ending boundary
-- columns structurally (spec 2.2: "Beginning and Ending are structural
-- roles, not merely labels"). Every existing/Blank-mode column defaults
-- to 'ordinary', so this is a no-op for every board that isn't Timeline
-- mode.
--
-- storyboard_reference_fields/storyboard_reference_items implement the
-- configurable Reference Panel (spec 3). Fields carry the same
-- UUID-identity + deterministic-slug + explicit-sort_order shape Kernel
-- 81A already established for columns/bands/rows, scoped per-board
-- (spec 3.5's "internal serialized keys must remain unique in board
-- scope"). Items belong to a field and, for a paired_list field only,
-- carry a `side` ('a' or 'b') selecting which of the field's two ordered
-- sublists they're part of; a plain `list` field's items are all `side =
-- 'single'`.
BEGIN;

ALTER TABLE storyboards
  ADD COLUMN IF NOT EXISTS mode TEXT NOT NULL DEFAULT 'blank'
    CHECK (mode IN ('blank', 'timeline')),
  ADD COLUMN IF NOT EXISTS template_version INTEGER;

ALTER TABLE storyboard_columns
  ADD COLUMN IF NOT EXISTS column_role TEXT NOT NULL DEFAULT 'ordinary'
    CHECK (column_role IN ('beginning', 'ordinary', 'ending'));

-- At most one Beginning and one Ending column per board -- the structural
-- guarantee the boundary-protection logic in columns.go is built on.
CREATE UNIQUE INDEX IF NOT EXISTS idx_storyboard_columns_one_beginning
  ON storyboard_columns(storyboard_id)
  WHERE column_role = 'beginning';

CREATE UNIQUE INDEX IF NOT EXISTS idx_storyboard_columns_one_ending
  ON storyboard_columns(storyboard_id)
  WHERE column_role = 'ending';

CREATE TABLE IF NOT EXISTS storyboard_reference_fields (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  storyboard_id UUID NOT NULL REFERENCES storyboards(id) ON DELETE CASCADE,

  slug TEXT NOT NULL,
  label TEXT NOT NULL,
  field_type TEXT NOT NULL
    CHECK (field_type IN ('short_text', 'long_text', 'list', 'paired_list')),
  sort_order INTEGER NOT NULL,

  -- short_text / long_text content only.
  text_content TEXT NOT NULL DEFAULT '',

  -- paired_list sublabels only (e.g. "Include" / "Exclude").
  sublabel_a TEXT NOT NULL DEFAULT '',
  sublabel_b TEXT NOT NULL DEFAULT '',

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (storyboard_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_storyboard_reference_fields_board_sort
  ON storyboard_reference_fields(storyboard_id, sort_order);

CREATE TABLE IF NOT EXISTS storyboard_reference_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  field_id UUID NOT NULL REFERENCES storyboard_reference_fields(id) ON DELETE CASCADE,

  -- 'single' for a plain `list` field; 'a'/'b' select a `paired_list`
  -- field's left/right sublist.
  side TEXT NOT NULL DEFAULT 'single' CHECK (side IN ('single', 'a', 'b')),
  sort_order INTEGER NOT NULL,
  content TEXT NOT NULL DEFAULT '',

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_storyboard_reference_items_field_side_sort
  ON storyboard_reference_items(field_id, side, sort_order);

COMMIT;

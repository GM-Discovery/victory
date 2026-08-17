BEGIN;

-- Kernel 89: Director Prepared Play.
--
-- ONE small, typed preparation record -- deliberately NOT an "Encounter"
-- object and deliberately NOT a macro (kernel 89 §6, §26). A row here is
-- one understandable prepared value the Director can recall during live
-- play; it never composes operations, never chains, never runs.
--
-- Why `payload` is JSONB and still not a scripting surface: `kind` is
-- CHECK-constrained to a closed set, and backend/internal/directorprep
-- validates the payload shape per kind before any write. There is no
-- interpreter anywhere that reads this column and performs actions; every
-- consumer reads exactly the two or three scalar fields its own kind
-- defines. Adding a `kind` therefore means adding a Go validator, not
-- adding an opcode.
--
-- Scope is show_id (kernel 89 §27). Shows already own persistent stage
-- state (`actions.show_id`, `shows.current_show_scene_placement_id`), so
-- this introduces no new ownership hierarchy. Merchants are NOT stored
-- here: `merchant_packets` (migration 061) is the canonical model and
-- Kernel 89 extends that table's authoring surface instead of growing a
-- competing one.
--
-- Every row is Director-only by construction: there is no Player-facing
-- read path in directorprep at all (kernel 89 §14/§28). Exposure to a
-- Player is always a separate, explicit act through an already-canonical
-- model (a participant_interactions row for a merchant, a Stage Effect for
-- an announcement).
CREATE TABLE IF NOT EXISTS director_preparations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  label TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT director_preparations_kind_check
    CHECK (kind IN ('target_complexity', 'announcement')),
  CONSTRAINT director_preparations_label_not_blank
    CHECK (length(btrim(label)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_director_preparations_show_kind
  ON director_preparations(show_id, kind, sort_order, created_at);

-- Kernel 89 §9.3: a merchant may be exposed to one Cohort rather than to
-- every eligible Player on the placement. The target lives in the existing
-- participant_interactions.configuration_json (`target_cohort_id`), NOT in
-- a new table -- but the join it implies is worth an index, since
-- ResolveEligibleContext now resolves the actor's cohort on every open.
-- No DDL needed for that (show_cohort_assignments already has its own
-- indexes from migration 096); this comment records the decision so a
-- future reader does not go looking for a merchant-targeting table.

COMMIT;

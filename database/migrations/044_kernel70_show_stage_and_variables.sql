BEGIN;

-- Kernel 70: persistent Show stage. The Show owns its current Scene and
-- runtime state; ending a Session must not reset it. This migration adds:
--   1. shows.current_show_scene_placement_id -- the persistent
--      current-Scene pointer. Ownership ("does this placement belong to
--      this Show", "is it archived") is validated in Go
--      (backend/internal/shows/stage.go), not by a DB constraint --
--      cross-table checks aren't expressible as a plain CHECK.
--   2. actions.show_id -- a nullable column alongside the existing
--      session_id, letting Cue/Scene state-changing actions persist
--      independent of any one session's lifecycle. world.LoadVenueSnapshot
--      folds show_id-scoped actions together with session_id-scoped ones,
--      reusing the existing canonical action-log replay mechanism rather
--      than inventing a second stage-state format.
--   3. shows.variables_json -- a materialized cache of Show variables set
--      by set_show_variable Cue actions. The canonical source of truth is
--      still the show_id-scoped actions row of type
--      'cue/set_show_variable'; this column exists purely so a plain
--      Show read doesn't need to replay the action log.
--   4. venues.config gains scene_rehearsal_enabled=true for first-theater
--      and catharsis (absent/false elsewhere, e.g. middle-school-stage),
--      following the existing venues.config JSONB boolean-flag precedent
--      (index_cards_enabled, actors_can_reveal) rather than extending
--      network/session_control.go's hardcoded venue-slug allowlist.

ALTER TABLE shows ADD COLUMN IF NOT EXISTS current_show_scene_placement_id UUID
  REFERENCES show_scene_placements(id) ON DELETE SET NULL;
ALTER TABLE shows ADD COLUMN IF NOT EXISTS variables_json JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE actions ADD COLUMN IF NOT EXISTS show_id UUID REFERENCES shows(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_actions_show_id ON actions(show_id) WHERE show_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_actions_show_id_ts ON actions(show_id, ts) WHERE show_id IS NOT NULL;

UPDATE venues
SET config = jsonb_set(COALESCE(config, '{}'::jsonb), '{scene_rehearsal_enabled}', 'true'::jsonb, TRUE)
WHERE slug IN ('first-theater', 'catharsis');

COMMIT;

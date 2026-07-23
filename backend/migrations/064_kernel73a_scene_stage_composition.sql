BEGIN;

-- Kernel 73A: Visual Scene Composition. Stores the actual visual
-- composition of a reusable Scene (tokens, map/backdrop, grid config,
-- index cards, and any other placeable element's on-stage state) --
-- deliberately its own scene-scoped snapshot/state table, separate from
-- the existing session-scoped `actions` event log and `venue_layout_
-- elements` (venue-level default placement). Those two remain exactly
-- what they were for live session play; this is additive. A Scene's own
-- composition is what gets loaded into a session when that Scene becomes
-- current (Kernel 70's shows.SetCurrentScenePlacement did not previously
-- touch composition at all -- this is the gap Kernel 73A closes).
--
-- Two-layer model, matching Kernel 69/70's own Scene/Placement precedent
-- (a Scene is reusable across Shows; a Placement is one Show's use of it):
--   - Base layer: show_scene_placement_id IS NULL, scene_id set. The
--     Scene's own default composition, reusable across every Show that
--     stages this Scene.
--   - Show layer: show_scene_placement_id set. A per-placement override/
--     addition scoped to exactly one Show's staging of the Scene, never
--     mutating the Base layer. A viewer resolving "what does this
--     placement look like" reads Base rows first, then Show-layer rows
--     for its own placement_id layered on top (Director authoring
--     concern, not enforced by SQL).
--
-- This follows Kernel 70/73's established precedent of a dedicated table
-- for a concern with its own authority/lifecycle (cues, participant_
-- interactions) rather than repurposing scenes.config_json/show_scene_
-- placements.config_json, which stay unused/unstructured as before.
CREATE TABLE IF NOT EXISTS scene_stage_elements (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  scene_id UUID NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
  show_scene_placement_id UUID REFERENCES show_scene_placements(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  label TEXT,
  data JSONB NOT NULL DEFAULT '{}'::jsonb,
  position JSONB NOT NULL DEFAULT '{}'::jsonb,
  visibility JSONB NOT NULL DEFAULT '{"toRoles":["audience","cast","crew","director","producer"],"privateTo":[]}'::jsonb,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT scene_stage_elements_kind_check CHECK (kind IN (
    'token', 'index_card', 'map_backdrop', 'grid_config'
  ))
);

CREATE INDEX IF NOT EXISTS idx_scene_stage_elements_scene_id
  ON scene_stage_elements(scene_id);
CREATE INDEX IF NOT EXISTS idx_scene_stage_elements_placement_id
  ON scene_stage_elements(show_scene_placement_id);

-- stage_element_bindings: the element-interaction binding schema (Kernel
-- 73A S6). Binds one placed composition element (typically a token, e.g.
-- Kessa's merchant stall) to an existing participant_interactions row, so
-- a Player clicking/tapping that element in the live stage or in Preview
-- as Player opens the bound interaction's Program (Kernel 73's Equip
-- Mode). Deliberately its own table rather than a column on scene_stage_
-- elements: a binding has its own creation/removal lifecycle independent
-- of the element's position/appearance edits, and this leaves room for a
-- second binding_type later (kernel-73A explicitly defers door-style
-- attempt bindings) without a schema migration on scene_stage_elements
-- itself. Exactly one binding type is implemented today, mirroring
-- participant_interactions_type_check's single-value-today precedent.
CREATE TABLE IF NOT EXISTS stage_element_bindings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  scene_stage_element_id UUID NOT NULL REFERENCES scene_stage_elements(id) ON DELETE CASCADE,
  binding_type TEXT NOT NULL DEFAULT 'participant_interaction',
  participant_interaction_id UUID NOT NULL REFERENCES participant_interactions(id) ON DELETE CASCADE,
  created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT stage_element_bindings_type_check CHECK (binding_type IN (
    'participant_interaction'
  )),
  UNIQUE (scene_stage_element_id, binding_type)
);

CREATE INDEX IF NOT EXISTS idx_stage_element_bindings_element_id
  ON stage_element_bindings(scene_stage_element_id);
CREATE INDEX IF NOT EXISTS idx_stage_element_bindings_interaction_id
  ON stage_element_bindings(participant_interaction_id);

-- scene_composer_enabled venue capability flag, following the Kernel 72A/
-- 73 pattern (055/058): a boolean in venues.config, read fail-closed for
-- unknown venues/missing flags, seeded here for existing venue rows AND
-- required in the Kernel 16 fresh-install venue seed (access/
-- kernel16_venue_bootstrap.go) -- miss either and fresh installs lose the
-- Scene Setup composer surface. Separate flag from stage_elements_enabled
-- (the live-session token/index-card action surface) because this gates
-- Scene-scoped authoring (Director/Crew editing a reusable Scene's
-- composition), a different surface with a different blast radius than
-- live in-session element actions.
UPDATE venues
SET config = jsonb_set(COALESCE(config, '{}'::jsonb), '{scene_composer_enabled}', 'true'::jsonb, TRUE)
WHERE slug = 'catharsis'
  AND NOT COALESCE((config ->> 'scene_composer_enabled')::boolean, FALSE);

COMMIT;

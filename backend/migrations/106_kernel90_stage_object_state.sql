BEGIN;

-- Kernel 90: canonical stage-object state, visibility & Cue control.
--
-- ONE table answers "who can currently perceive or interact with this thing
-- on stage?" for every durable Pixi-rendered object Victory has. Before this
-- migration that question had three unrelated answers (kernel 90 §0):
--
--   1. `act/reveal_element` / `act/hide_element` action rows, replayed per
--      session by world/snapshot.go's deriveElementLayerVisibility into an
--      actor/audience *layer* binary. Durable objects, but no Cohort or
--      Character scoping and no state of its own -- visibility was a
--      derived property of an append-only log.
--   2. Cue actions reveal_object/hide_object/enable_interaction/
--      disable_interaction, deferred since Kernel 70 for exactly the reason
--      this table removes: there was no stable cross-kind object identity to
--      target (see the old comment at cues/types.go).
--   3. `participant_interactions.enabled`, a global authoring boolean with
--      no relationship to either.
--
-- This is a reconciliation, not a fourth mechanism. (1) now writes here,
-- (2) now targets rows here, and (3) keeps its own distinct meaning with a
-- documented bridge -- see `interaction_enabled` below.
--
-- Kernel 89's Scene boundary is preserved (§2): NOTHING here is written to
-- scene_stage_elements or scenes. A Scene still describes what is on the
-- stage; this table describes who can currently perceive it. A Scene staged
-- into a second Show therefore starts from its authored default with no
-- inherited hidden state, which is the intended semantics, not an omission.
--
-- No backfill is performed. Any object hidden under mechanism (1) before
-- this migration reads as visible afterward. That is a deliberate operator
-- decision (Grant, 2026-08-17): re-hiding a handful of objects through the
-- new controls is cheaper and far less risky than inferring canonical scoped
-- state from a legacy layer binary that never carried scope information.

-- stage_object_states: one row per (Show, object) that has ever been given
-- non-default state. Absence of a row means "authored default", which for
-- every supported kind is visible-and-enabled (§10) -- so placing an object
-- costs no write here and newly placed objects are never accidentally
-- Director-only.
--
-- Scoped to show_id (§26). The narrowest canonical owner that preserves
-- intended play: Cohort-scoped visibility is inherently a Show-scale
-- concept (show_cohorts is Show-owned), and Kernel 87's drawing_objects
-- already chose show_id for the same reason. Deliberately NOT session-
-- scoped -- §26 forbids reviving Session as the durable owner, and Show
-- ownership is what makes state survive the session replacement that
-- Kernel 70A's Start Show Session performs.
--
-- object_id is a bare UUID with no foreign key, because the four supported
-- kinds live in four different tables. Referential integrity is enforced in
-- Go instead: stageobjects.ResolveRef validates that the target row exists,
-- belongs to this Show, and supports the dimension being set, before any
-- write. Stale rows (object deleted out from under a state row) are treated
-- as absent by the projector rather than erroring a whole snapshot -- the
-- same tolerance world/snapshot.go already applies to a stale current-scene
-- pointer.
CREATE TABLE IF NOT EXISTS stage_object_states (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
  object_kind TEXT NOT NULL,
  object_id UUID NOT NULL,

  -- visibility: 'visible' means every ordinary permitted viewer of this
  -- Show/Scene perceives the object, Audience included. 'hidden' means no
  -- viewer perceives it EXCEPT Director+ (who always work backstage, §14)
  -- and any viewer matched by a stage_object_scope_grants row below.
  --
  -- Hidden is not deleted (§7). Deletion remains each kind's own existing
  -- operation (act/remove_element, drawing_objects.deleted_at, scenes'
  -- stage-element endpoints) and is untouched by this kernel.
  visibility TEXT NOT NULL DEFAULT 'visible',

  -- interaction_enabled is NULL for objects with no interaction dimension
  -- (§8/§12: do not show Interaction controls for objects that cannot be
  -- acted upon). Non-NULL only for kinds that bind an interaction.
  --
  -- The bridge to participant_interactions.enabled (§20), which is the one
  -- boundary in this kernel that genuinely keeps two booleans: that column
  -- means "this interaction exists at all" and is out-of-play AUTHORING
  -- state, global across every Show and the tutorial. This column means
  -- "the Director has switched it off for the rest of this Show", which is
  -- in-play state. They are not competing truths about the same question,
  -- so projection reads BOTH and requires both: globally disabled is never
  -- invokable, and Show-disabled is visible-but-not-invokable. A Director
  -- disabling Kessa mid-scene can therefore never disable Kessa in another
  -- Show or in the tutorial.
  interaction_enabled BOOLEAN,

  updated_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT stage_object_states_kind_check CHECK (object_kind IN (
    -- A live warehouse stage object: elements + venue_layout_elements, what
    -- "Add Token"/"Create Index Card" produce in Catharsis (actions/token.go,
    -- actions/indexcard.go). These are the objects mechanism (1) used to
    -- cover.
    'venue_layout_element',
    -- A Kernel 73A Scene-authored composition element (token/index_card).
    -- These had NO visibility control of any kind before Kernel 90.
    'scene_stage_element',
    -- A Kernel 87 drawing object. Already durable with a stable id and an
    -- existing cohort_id column; §28 asked for the audit and the answer was
    -- yes. No coordinate or Detail View math is touched.
    'drawing_object',
    -- A participant interaction reached through a stage_element_bindings
    -- row. Carries interaction state; its visibility follows its bound
    -- element rather than being set here.
    'participant_interaction'
  )),
  CONSTRAINT stage_object_states_visibility_check CHECK (visibility IN (
    'visible', 'hidden'
  )),
  -- participant_interaction rows exist to carry interaction state, so an
  -- interaction row with a NULL boolean would be a meaningless row.
  CONSTRAINT stage_object_states_interaction_kind_check CHECK (
    object_kind <> 'participant_interaction' OR interaction_enabled IS NOT NULL
  ),
  UNIQUE (show_id, object_kind, object_id)
);

CREATE INDEX IF NOT EXISTS idx_stage_object_states_show
  ON stage_object_states(show_id);

-- stage_object_scope_grants: the exception list that makes a hidden object
-- perceivable by a named audience. Multiple grants per object (Grant,
-- 2026-08-17) -- "revealed to Cohort A AND to one specific Character" is a
-- real theatrical need, and a single scope_kind column on the state row
-- could not express it without hide/reveal churn.
--
-- Grants are meaningful ONLY when the parent row's visibility is 'hidden'.
-- A visible object needs no grants (§10), which is why the common case
-- writes nothing to this table at all.
--
-- This is deliberately not an ACL platform (§34). There are four scope
-- kinds, all theatrical, all resolved server-side from existing authority:
--
--   'cast'      -- every ordinary Show participant, but NOT the Audience.
--                  This is how "visible to the players, hidden from the
--                  house" is expressed; §18 forbids assuming Audience sees
--                  whatever Cast sees, and without this kind that case
--                  would be inexpressible.
--   'cohort'    -- one show_cohorts row (§16). Resolved through the same
--                  rollaudience cohort resolution Kernel 86 already uses,
--                  never from a client-supplied cohort id.
--   'character' -- one character_cards row (§17). Matched against the
--                  viewer's server-resolved CURRENTLY SELECTED Character,
--                  not merely a Character they own, so switching Character
--                  correctly changes what they perceive.
--   'audience'  -- Audience-tier viewers (§18).
--
-- Director+ is absent on purpose: "Director-only" is simply a hidden row
-- with no grants. Adding a 'director' scope kind would give one state two
-- spellings, and §0 says do not add mechanisms.
CREATE TABLE IF NOT EXISTS stage_object_scope_grants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  stage_object_state_id UUID NOT NULL
    REFERENCES stage_object_states(id) ON DELETE CASCADE,
  scope_kind TEXT NOT NULL,
  -- NULL for 'cast' and 'audience', which name a tier rather than a row.
  -- Not a foreign key for the same cross-table reason object_id above is
  -- not; stageobjects validates the cohort/Character belongs to this Show
  -- before writing, and an orphaned grant simply matches nobody.
  scope_id UUID,
  created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT stage_object_scope_grants_kind_check CHECK (scope_kind IN (
    'cast', 'cohort', 'character', 'audience'
  )),
  CONSTRAINT stage_object_scope_grants_id_shape_check CHECK (
    (scope_kind IN ('cohort', 'character') AND scope_id IS NOT NULL)
    OR (scope_kind IN ('cast', 'audience') AND scope_id IS NULL)
  ),
  -- NULLS NOT DISTINCT (Postgres 15+; this install is 16) so a second
  -- 'cast' or 'audience' grant collides rather than silently duplicating.
  UNIQUE NULLS NOT DISTINCT (stage_object_state_id, scope_kind, scope_id)
);

CREATE INDEX IF NOT EXISTS idx_stage_object_scope_grants_state
  ON stage_object_scope_grants(stage_object_state_id);

COMMIT;

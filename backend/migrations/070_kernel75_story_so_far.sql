BEGIN;

-- Kernel 75 S6: Story So Far -- a Character's durable, chronological,
-- private-by-default record of what actually happened in play.
--
-- WHY A DEDICATED TABLE rather than new page_key values in
-- character_workbook_entries (migration 031), which is the repo's existing
-- Character-scoped entry store. Four blockers, each independently
-- disqualifying:
--
--   1. character_workbook_entries has no show_run_id / show_id / session_id
--      / show_scene_placement_id. S6.1 requires all four. Burying them in
--      the payload JSONB would make "every story event in this Show" an
--      unindexed scan and put ON DELETE behavior out of reach.
--
--   2. It has no privacy column, and S6.3 requires private-by-default with
--      a future-compatible reveal state. The nearest existing vocabulary is
--      character_face_overrides.visibility_mode (inferred|shown|hidden),
--      whose 'inferred' value is meaningless for a discrete event -- there
--      is nothing to infer about whether something happened.
--
--   3. Its author_user_id is NOT NULL REFERENCES users(id). These rows are
--      system-generated. Inventing a synthetic user to satisfy the
--      constraint would be a lie in the schema; a nullable
--      authored_by_user_id is the honest column, and NULL is the signal
--      "generated, not written by a person".
--
--   4. buildWorkbookPages loads ALL entries for a card and filters in Go.
--      Adding a growing event stream would slow every workbook page read
--      for every Character, forever.
--
-- S1.8 is also explicit that generated play history must not be merged
-- indistinguishably into Player-authored Face Sheet history. Separate
-- tables make that separation structural rather than conventional.
CREATE TABLE IF NOT EXISTS character_story_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  owner_user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  event_type TEXT NOT NULL,
  title      TEXT NOT NULL DEFAULT '',
  summary    TEXT NOT NULL DEFAULT '',

  show_run_id             UUID REFERENCES show_runs(id) ON DELETE SET NULL,
  show_id                 UUID REFERENCES shows(id) ON DELETE SET NULL,
  session_id              UUID REFERENCES sessions(id) ON DELETE SET NULL,
  show_scene_placement_id UUID REFERENCES show_scene_placements(id) ON DELETE SET NULL,

  -- Polymorphic source reference, deliberately without a foreign key: the
  -- row a clause was derived from lives in one of seven different tables,
  -- and a real FK would mean seven nullable columns and seven joins.
  -- source_kind names which table source_ref points into. These two are
  -- provenance for a human reader and for dedupe -- nothing reads them to
  -- make an authority decision, so the missing referential integrity costs
  -- nothing that matters.
  source_kind TEXT NOT NULL DEFAULT '',
  source_ref  TEXT NOT NULL DEFAULT '',

  -- S6.3: private by default. Only two states exist. There is deliberately
  -- no 'public': publishing a Character's play history to anyone outside
  -- the table is a decision no kernel has made, and an unused enum value is
  -- an invitation to make it accidentally. 'table' means "visible to the
  -- people I play with", which is the only widening S1.8 anticipates.
  visibility_state TEXT NOT NULL DEFAULT 'private',

  -- NULL = system-generated (the overwhelming majority). Set only for
  -- Director-authored moments arriving through /journal (S6.4), where the
  -- authoring Director's identity must be preserved.
  authored_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

  -- Which revision of the template library produced this text. Lets a later
  -- kernel improve the prose without retroactively claiming old entries
  -- were written by the new templates.
  generator_version INT NOT NULL DEFAULT 1,

  occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT character_story_events_visibility_check
    CHECK (visibility_state IN ('private', 'table')),

  CONSTRAINT character_story_events_type_check CHECK (event_type IN (
    'arrival',
    'merchant_met',
    'merchant_stance',
    'merchant_haggle',
    'equipment_acquired',
    'door_intention',
    'dialogue_learned',
    'gate_opened',
    'face_sheet_echo',
    'director_moment',
    'aftercare',
    'tutorial_completed'
  )),

  CONSTRAINT character_story_events_source_kind_check CHECK (source_kind IN (
    '',
    'tutorial_milestone',
    'inventory_item',
    'freeform_submission',
    'interaction_attempt',
    'dialogue_topic',
    'workbook_entry',
    'character_journal',
    'aftercare'
  ))
);

-- The idempotency mechanism, and the reason storysofar.Generate is required
-- to be a pure function of stored inputs. A retried Continue re-derives
-- byte-identical (event_type, source_kind, source_ref) tuples and collides
-- here rather than duplicating the Player's story. If Generate ever became
-- clock- or map-order-dependent, this index would silently stop absorbing
-- retries -- which is why generate_test.go asserts determinism directly.
--
-- The index is TOTAL, not partial. A template that yields at most one clause
-- per Character (arrival, face_sheet_echo, tutorial_completed) uses
-- source_ref = '' and dedupes on (character, type, '', ''). That is what
-- makes "one arrival line per Character" a database fact rather than a
-- convention in Go.
CREATE UNIQUE INDEX IF NOT EXISTS uq_character_story_events_dedupe
  ON character_story_events(character_card_id, event_type, source_kind, source_ref);

CREATE INDEX IF NOT EXISTS idx_character_story_events_character
  ON character_story_events(character_card_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_character_story_events_show
  ON character_story_events(show_id, occurred_at DESC)
  WHERE show_id IS NOT NULL;

-- Kernel 75 S7.1/S7.3: one-time, Player-level recognition.
--
-- Keyed on user_id ALONE -- never on character_card_id. That single-column
-- uniqueness IS the anti-duplication mechanism S7.3 requires: a Player who
-- completes the tutorial again with a second Character hits ON CONFLICT DO
-- NOTHING and is told so, while the Character-level acknowledgement lives
-- in character_story_events (keyed on character_card_id) and is correctly
-- earned again. The split between the two tables is the whole answer to
-- "first-time versus repeat".
--
-- Bounded to ONE key by CHECK constraint. S7.4 forbids a numeric economy,
-- and S14 excludes a full Player-level economy, arbitrary point values, and
-- a general achievement shop. There is no points column, no level, no
-- ordering, and nothing consumable. A second recognition key is a migration,
-- on purpose -- the same friction participant_tutorial_progress uses to
-- keep itself from becoming a quest tracker.
CREATE TABLE IF NOT EXISTS player_recognition_grants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  recognition_key TEXT NOT NULL,
  first_character_card_id UUID REFERENCES character_cards(id) ON DELETE SET NULL,
  first_show_id UUID REFERENCES shows(id) ON DELETE SET NULL,
  granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT player_recognition_grants_key_check
    CHECK (recognition_key IN ('first_tutorial_completed')),
  UNIQUE (user_id, recognition_key)
);

COMMIT;

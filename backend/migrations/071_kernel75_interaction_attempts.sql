BEGIN;

-- Kernel 75 S5.1: Story So Far may draw on "Kessa stance attempts" and
-- "Haggle result". Kernel 73 recorded neither durably.
--
-- merchant.AttemptStance and merchant.AttemptHaggle write only an ephemeral
-- actions row (EventKind 'interaction/stance_attempted' /
-- 'interaction/haggle_attempted'). That log is keyed to actor_id (a user),
-- not to a Character, and it is a Session artifact -- it belongs to Showing
-- Review and does not survive Session archival as Character history. A
-- Character-scoped reflection cannot honestly be built from it.
--
-- So this table is the record of record for "what did this Character
-- actually try at the stall". The actions rows stay exactly as they are:
-- Showing Review reads them, and removing them would be a K73 regression
-- for no gain. This is an ADDITIONAL write, not a replacement.
--
-- No backfill exists and none is possible: the ephemeral rows cannot be
-- attributed to a Character with confidence. Characters who played the
-- tutorial before Kernel 75 simply have no rows here, and the Story So Far
-- template library omits the stance/haggle clauses for them (K75 S5.2,
-- "Omit missing clauses"). That is the honest outcome, not a defect.
CREATE TABLE IF NOT EXISTS character_interaction_attempts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  actor_user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  participant_interaction_id UUID REFERENCES participant_interactions(id) ON DELETE SET NULL,
  show_run_id             UUID REFERENCES show_runs(id) ON DELETE SET NULL,
  show_id                 UUID REFERENCES shows(id) ON DELETE SET NULL,
  session_id              UUID REFERENCES sessions(id) ON DELETE SET NULL,
  show_scene_placement_id UUID REFERENCES show_scene_placements(id) ON DELETE SET NULL,

  packet_slug  TEXT NOT NULL DEFAULT '',
  attempt_kind TEXT NOT NULL,

  -- Stance columns. response_tier is the authored-variant index the die
  -- selected, not a score: AttemptStance's die never changes the
  -- disposition, it only buckets into low/mid/high authored text.
  stance_key   TEXT NOT NULL DEFAULT '',
  disposition  TEXT NOT NULL DEFAULT '',
  response_tier INT,

  -- Haggle columns. success is nullable because a stance row has no notion
  -- of success -- a stance is never pass/fail. NULL here means "not
  -- applicable", never "unknown".
  skill_key    TEXT NOT NULL DEFAULT '',
  has_skill    BOOLEAN,
  die          TEXT NOT NULL DEFAULT '',
  total        INT,
  target_value INT,
  success      BOOLEAN,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT character_interaction_attempts_kind_check
    CHECK (attempt_kind IN ('stance', 'haggle'))
);

-- Deliberately NO unique constraint and no idempotency key, unlike
-- character_inventory_purchase_attempts (migration 059). A purchase must
-- happen once; a stance or a Haggle is genuinely repeatable play, and a
-- Player who tries three stances tried three stances. The Story So Far
-- templates want "the last stance tried" and "the successful Haggle, if
-- any", both of which need the full ordered history rather than a
-- collapsed one. A retried HTTP request producing a second row here is
-- indistinguishable from a second genuine attempt, and that is acceptable:
-- these rows drive reflective prose, never authority or inventory.
CREATE INDEX IF NOT EXISTS idx_character_interaction_attempts_character
  ON character_interaction_attempts(character_card_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_character_interaction_attempts_show
  ON character_interaction_attempts(show_id, created_at DESC)
  WHERE show_id IS NOT NULL;

COMMIT;

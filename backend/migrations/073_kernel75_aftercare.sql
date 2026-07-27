BEGIN;

-- Kernel 75 S8: Aftercare.
--
-- WHAT AFTERCARE IS, stated once so the tables below read as deliberate:
-- a short, optional, qualitative, durable, per-(Player, Character, Show)
-- written reflection, offered once the tutorial is complete. The Player
-- initiates it. It never blocks anything. Directors+ at that Location can
-- read it, and the form says so before the Player types a word.
--
-- S1.12 is explicit that this is QUALITATIVE: there are no numeric ratings,
-- no scores, and no scale columns anywhere below. S1.15 makes the Director's
-- Chair read-only: there is no reply, annotation, score, or thread table,
-- and no route that would write to one.

-- One submission per (Player, Character, Show). UNIQUE rather than
-- append-only because S8.3 says Save and Close "creates or updates one
-- Aftercare submission" -- a Player revising their answer is editing their
-- reflection, not filing a second one.
--
-- responses_json is keyed by prompt key. The prompts themselves live in Go
-- (aftercare.PromptSetV1), not in a table: S1.12 fixes them at three, and a
-- prompt table would be a form builder, which is not in this kernel.
-- prompt_set_version is stored so a later rewrite of the prompts never
-- retroactively mislabels answers written against the old ones.
CREATE TABLE IF NOT EXISTS aftercare_submissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
  show_run_id UUID REFERENCES show_runs(id) ON DELETE SET NULL,
  session_id  UUID REFERENCES sessions(id) ON DELETE SET NULL,
  prompt_set_version INT NOT NULL DEFAULT 1,
  responses_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  -- S8.1: question two may name a person. Stored as an optional structured
  -- reference plus free text, so "who surprised you" can be a real Player
  -- when the roster supports it and prose when it does not.
  surprising_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  surprising_character_card_id UUID REFERENCES character_cards(id) ON DELETE SET NULL,
  idempotency_key TEXT NOT NULL DEFAULT '',
  submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, character_card_id, show_id)
);

-- Drafts live in their OWN table, not in a nullable column on the
-- submission row.
--
-- S8.2 requires that a draft "is not considered submitted" and does not
-- reset the skip count. Keeping drafts out of aftercare_submissions means a
-- reader of that table -- including the Director's Chair -- cannot mistake
-- unfinished thinking for something the Player chose to share. The
-- separation is structural, so no query has to remember a flag.
CREATE TABLE IF NOT EXISTS aftercare_response_drafts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
  responses_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, character_card_id, show_id)
);

-- Explicit skips, append-only.
--
-- Deliberately NO unique constraint: a Player who skips, reopens later, and
-- skips again genuinely skipped twice, and that repetition is the signal
-- S1.14 wants to reflect back to them.
--
-- Deliberately NO counter column anywhere. S8.5's "consecutive" skips are
-- COMPUTED as the skips recorded since that Player's most recent submission.
-- A stored counter would need decrementing somewhere, and a decrement is a
-- route an attacker or a bug can reach; a computed value resets by
-- construction the moment a submission exists (S8.3), and there is no
-- DELETE route for these rows anywhere in the codebase.
--
-- This is also why the skip count cannot be client-forged (S12): the only
-- writer is the skip endpoint, identity comes from the session cookie, and
-- the request body carries nothing but a confirmation flag.
CREATE TABLE IF NOT EXISTS aftercare_skips (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
  skipped_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_aftercare_submissions_show
  ON aftercare_submissions(show_id, submitted_at DESC);
CREATE INDEX IF NOT EXISTS idx_aftercare_skips_user
  ON aftercare_skips(user_id, skipped_at DESC);
CREATE INDEX IF NOT EXISTS idx_aftercare_skips_show
  ON aftercare_skips(show_id, skipped_at DESC);

COMMIT;

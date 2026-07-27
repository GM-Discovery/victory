BEGIN;

-- Kernel 75 S3.1 and the operator's dialogue-editing decision.
--
-- Ra's prose shipped as seed data in migration 066 with no editing surface,
-- so every wording change was a migration. K75 adds a Director+ editor, and
-- also needs to install new closing narration. Shipping both in one kernel
-- raises an obvious hazard: a later content migration silently overwriting
-- an edit a Director made through the editor.
--
-- content_origin is the guard that makes the two safe together:
--
--   EVERY CONTENT MIGRATION TOUCHES A SEEDED FIELD ONLY
--   WHERE content_origin = 'seed'.
--
-- The moment a Director edits a row through the authoring API, its
-- content_origin flips to 'authored' and no future migration will ever
-- clobber it. That turns "append-forward canon; never rewrite an operator's
-- words" from a convention a reviewer has to remember into a predicate the
-- database enforces.
ALTER TABLE dialogue_packets
  ADD COLUMN IF NOT EXISTS updated_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS content_origin TEXT NOT NULL DEFAULT 'seed',
  -- Ordered short beats for the staged lock reveal (S3.1): Ra exposes the
  -- concealed mechanism, works it, the bolt releases, the gate opens. Kept
  -- as an ordered array rather than one paragraph so the Program can reveal
  -- the ending a beat at a time instead of as a wall of prose.
  --
  -- Readers MUST fall back to closing_narration when this is empty. That
  -- fallback is what lets the frontend and backend deploy in either order,
  -- and what keeps an unedited packet working.
  ADD COLUMN IF NOT EXISTS closing_beats JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE dialogue_topics
  ADD COLUMN IF NOT EXISTS updated_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS content_origin TEXT NOT NULL DEFAULT 'seed';

ALTER TABLE dialogue_packets
  DROP CONSTRAINT IF EXISTS dialogue_packets_content_origin_check;
ALTER TABLE dialogue_packets
  ADD CONSTRAINT dialogue_packets_content_origin_check
  CHECK (content_origin IN ('seed', 'authored'));

ALTER TABLE dialogue_topics
  DROP CONSTRAINT IF EXISTS dialogue_topics_content_origin_check;
ALTER TABLE dialogue_topics
  ADD CONSTRAINT dialogue_topics_content_origin_check
  CHECK (content_origin IN ('seed', 'authored'));

-- Who changed what, when. An NPC's authored prose is canon that every Player
-- reads verbatim, so an edit is worth a durable record -- both so a Producer
-- can see what changed before a recording, and so a bad edit can be read
-- back rather than reconstructed from memory.
--
-- Append-only by construction: there is no UPDATE or DELETE route for these
-- rows anywhere in the codebase.
CREATE TABLE IF NOT EXISTS dialogue_packet_revisions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  packet_id UUID NOT NULL REFERENCES dialogue_packets(id) ON DELETE CASCADE,
  topic_id  UUID REFERENCES dialogue_topics(id) ON DELETE SET NULL,
  field_key TEXT NOT NULL,
  previous_text TEXT NOT NULL DEFAULT '',
  next_text     TEXT NOT NULL DEFAULT '',
  edited_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  edited_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dialogue_packet_revisions_packet
  ON dialogue_packet_revisions(packet_id, edited_at DESC);

COMMIT;

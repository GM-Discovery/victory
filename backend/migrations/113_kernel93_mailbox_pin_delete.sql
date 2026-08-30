ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS is_pinned BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_messages_to_user_pinned_created_at
  ON messages(to_user_id, is_pinned DESC, created_at DESC);

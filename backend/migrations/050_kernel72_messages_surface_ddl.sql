-- Kernel 72: DDL extracted verbatim from messages.EnsureKernel11MessagesSurface (function removed).
CREATE TABLE IF NOT EXISTS messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  message_type TEXT NOT NULL DEFAULT 'message',
  to_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  from_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  subject TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  venue_slug TEXT NOT NULL DEFAULT '',
  session_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  is_read BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_messages_to_user_created_at
  ON messages(to_user_id, created_at DESC);

ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS message_type TEXT NOT NULL DEFAULT 'message',
  ADD COLUMN IF NOT EXISTS venue_slug TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS session_id UUID;

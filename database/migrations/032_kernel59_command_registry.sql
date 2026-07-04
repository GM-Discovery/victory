BEGIN;

CREATE TABLE IF NOT EXISTS command_execution_receipts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  command_path TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  result JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_command_execution_receipts_unique
  ON command_execution_receipts(actor_user_id, command_path, idempotency_key);

COMMIT;

BEGIN;

-- Kernel 68: minimal Create Production flow. The `productions` table
-- (migration 000_kernel42_productions_baseline.sql) has never had a
-- created_by_user_id column -- every production row to date was inserted
-- directly by migrations/tests/the fresh-install script, since there was no
-- in-app create route at all. Additive, nullable (existing rows have no
-- known creator), ON DELETE SET NULL so deleting a user never cascades into
-- deleting productions they created.

ALTER TABLE productions
  ADD COLUMN IF NOT EXISTS created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL;

COMMIT;

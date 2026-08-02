BEGIN;

-- Kernel 77 Goal A: deletion receipts. Deliberately minimal -- enough to
-- prove a deletion happened and audit its shape, nothing that could be used
-- to re-identify the deleted account. No email, handle, or Discord id is
-- stored here, and the account reference is a fresh random id unrelated to
-- the original user id (see account-deletion-policy.md).
CREATE TABLE IF NOT EXISTS account_deletion_receipts (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  anonymized_account_ref UUID NOT NULL DEFAULT gen_random_uuid(),
  deleted_private_count  INT NOT NULL DEFAULT 0,
  anonymized_shared_count INT NOT NULL DEFAULT 0,
  transferred_count      INT NOT NULL DEFAULT 0,
  status                TEXT NOT NULL DEFAULT 'completed'
);

-- Kernel 77 Goal B: self-service data export jobs. One row per requested
-- export; the archive lives on local disk (path not exposed to the client)
-- and is fetched via a single-use, hashed, expiring download token, never a
-- predictable URL. Cascades with the user: if the account is later deleted
-- before the export is downloaded, the job and any pending token go with it
-- rather than becoming an orphaned download of a since-deleted account.
CREATE TABLE IF NOT EXISTS account_export_jobs (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  status              TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','running','ready','failed','expired')),
  requested_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at        TIMESTAMPTZ,
  expires_at          TIMESTAMPTZ,
  download_token_hash BYTEA,
  archive_path        TEXT,
  byte_size           BIGINT,
  error               TEXT
);

CREATE INDEX IF NOT EXISTS idx_account_export_jobs_user_id ON account_export_jobs(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_account_export_jobs_download_token
  ON account_export_jobs(download_token_hash) WHERE download_token_hash IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_account_export_jobs_expires_at ON account_export_jobs(expires_at);

-- Kernel 77 Goal D: recovery-email verification. A user's `users.email` may
-- be set (e.g. adopted from Discord) without ever having been proven
-- deliverable; §8.5 requires Victory not advertise password-reset-by-email
-- for an unverified address. Verification is scoped to one specific email
-- string, so changing the address always requires re-verification.
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS auth.email_verification_tokens (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  email       TEXT NOT NULL,
  token_hash  BYTEA NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at  TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  request_ip  INET
);

CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_user_id ON auth.email_verification_tokens(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_email_verification_tokens_hash_unique ON auth.email_verification_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_expires_at ON auth.email_verification_tokens(expires_at);

COMMIT;

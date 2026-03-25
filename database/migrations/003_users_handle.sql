ALTER TABLE users
ADD COLUMN IF NOT EXISTS handle TEXT;

UPDATE users
SET handle = COALESCE(handle, 'user_' || substr(replace(id::text, '-', ''), 1, 8))
WHERE handle IS NULL;

ALTER TABLE users
ALTER COLUMN handle SET NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'users_handle_unique'
  ) THEN
    ALTER TABLE users
    ADD CONSTRAINT users_handle_unique UNIQUE (handle);
  END IF;
END
$$;
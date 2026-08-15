BEGIN;

-- Kernel 88: canonical Social Stance. Confirmed nowhere in the repo before
-- this migration (kernel-85's repo audit S13.9 explicitly declined to
-- invent one). Data-driven registry, same pattern as socio_statuses
-- (migration 097) -- adding a Stance for a later module/expansion is a
-- seed-row INSERT, not a schema change. Vocabulary confirmed by product
-- owner: Insight, Command, Convince, Sympathize, Follow.
CREATE TABLE IF NOT EXISTS socio_stances (
  key TEXT PRIMARY KEY,
  label TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  wheel_order INTEGER NOT NULL DEFAULT 0
);

INSERT INTO socio_stances (key, label, wheel_order) VALUES
  ('insight', 'Insight', 0),
  ('command', 'Command', 1),
  ('convince', 'Convince', 2),
  ('sympathize', 'Sympathize', 3),
  ('follow', 'Follow', 4)
ON CONFLICT (key) DO NOTHING;

-- One active Stance pointer per Character. Nullable stance_key -- "no
-- Stance chosen yet" is a valid state, same lazy-row convention as
-- character_socio_state (row created on first read/write, not backfilled).
CREATE TABLE IF NOT EXISTS character_socio_stance (
  character_card_id UUID PRIMARY KEY REFERENCES character_cards(id) ON DELETE CASCADE,
  stance_key TEXT REFERENCES socio_stances(key) ON DELETE RESTRICT,
  set_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;

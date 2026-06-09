CREATE TABLE IF NOT EXISTS productions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id UUID NOT NULL,

  name TEXT NOT NULL,
  slug TEXT NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE (location_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_productions_location_id
  ON productions(location_id);

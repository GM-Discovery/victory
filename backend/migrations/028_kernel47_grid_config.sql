BEGIN;

CREATE TABLE IF NOT EXISTS venue_grid_configs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  venue_id uuid NOT NULL REFERENCES venues(id) ON DELETE CASCADE,
  grid_type text NOT NULL DEFAULT 'none',
  hex_orientation text NOT NULL DEFAULT 'flat-top',
  cell_size double precision NOT NULL DEFAULT 50,
  offset_x double precision NOT NULL DEFAULT 0,
  offset_y double precision NOT NULL DEFAULT 0,
  line_width double precision NOT NULL DEFAULT 1,
  opacity double precision NOT NULL DEFAULT 0.45,
  line_style text NOT NULL DEFAULT 'neutral',
  visible boolean NOT NULL DEFAULT true,
  updated_by_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (venue_id)
);

COMMIT;

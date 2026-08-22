BEGIN;

-- Kernel 93: the canonical single-Showing Audience admission primitive.
-- Deliberately separate from show_run_tickets (Kernel 71's two-punch
-- Player ticket, Show-Run-scoped) -- see that table's own comment, which
-- already anticipated "a future Audience-ticket kernel widens this CHECK
-- rather than redesigning the table" and was overruled: Kernel 93 needs
-- Showing-scoped admission (one Showing, not a whole Show Run's lifetime),
-- which show_run_tickets cannot express without conflating the two
-- systems. A holder may be admitted to many Showings; each Showing tracks
-- its own admitted users independently.
CREATE TABLE IF NOT EXISTS audience_admissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  showing_id UUID NOT NULL REFERENCES showings(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  issued_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (showing_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_audience_admissions_user_id ON audience_admissions(user_id);

-- Kernel 93 §4/§26: one canonical Showing-level Audience projection config
-- row, not a generic settings engine. show_dice_rolls defaults TRUE to
-- match current public rollaudience.ModeShow behavior (already visible to
-- every viewer); show_presence and show_health_statuses default FALSE per
-- operator decision -- Director opts in per Showing.
CREATE TABLE IF NOT EXISTS audience_projection_configs (
  showing_id UUID PRIMARY KEY REFERENCES showings(id) ON DELETE CASCADE,
  show_dice_rolls BOOLEAN NOT NULL DEFAULT TRUE,
  show_presence BOOLEAN NOT NULL DEFAULT FALSE,
  show_health_statuses BOOLEAN NOT NULL DEFAULT FALSE,
  updated_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;

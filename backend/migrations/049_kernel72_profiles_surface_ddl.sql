-- Kernel 72: DDL extracted verbatim from profiles.EnsureKernel9ProfileSurface.
-- The Go bootstrap now carries only venue seed rows.
CREATE TABLE IF NOT EXISTS performer_profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,

  draft_display_name TEXT NOT NULL DEFAULT '',
  draft_stage_name TEXT NOT NULL DEFAULT '',
  draft_pronouns TEXT NOT NULL DEFAULT '',
  draft_headshot_url TEXT NOT NULL DEFAULT '',
  draft_performance_age_range TEXT NOT NULL DEFAULT '',
  draft_favorite_fun TEXT NOT NULL DEFAULT '',
  draft_most_relaxed TEXT NOT NULL DEFAULT '',
  draft_favorite_color TEXT NOT NULL DEFAULT '',
  draft_favorite_artist TEXT NOT NULL DEFAULT '',
  draft_bio TEXT NOT NULL DEFAULT '',
  draft_credits TEXT NOT NULL DEFAULT '',
  draft_skills TEXT NOT NULL DEFAULT '',
  draft_availability TEXT NOT NULL DEFAULT '',
  draft_public_links TEXT NOT NULL DEFAULT '',
  draft_history TEXT NOT NULL DEFAULT '[]',

  published_display_name TEXT NOT NULL DEFAULT '',
  published_stage_name TEXT NOT NULL DEFAULT '',
  published_pronouns TEXT NOT NULL DEFAULT '',
  published_headshot_url TEXT NOT NULL DEFAULT '',
  published_performance_age_range TEXT NOT NULL DEFAULT '',
  published_favorite_fun TEXT NOT NULL DEFAULT '',
  published_most_relaxed TEXT NOT NULL DEFAULT '',
  published_favorite_color TEXT NOT NULL DEFAULT '',
  published_favorite_artist TEXT NOT NULL DEFAULT '',
  published_bio TEXT NOT NULL DEFAULT '',
  published_credits TEXT NOT NULL DEFAULT '',
  published_skills TEXT NOT NULL DEFAULT '',
  published_availability TEXT NOT NULL DEFAULT '',
  published_public_links TEXT NOT NULL DEFAULT '',
  published_history TEXT NOT NULL DEFAULT '[]',

  is_published BOOLEAN NOT NULL DEFAULT FALSE,
  published_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_performer_profiles_is_published
  ON performer_profiles(is_published);

ALTER TABLE performer_profiles
  ADD COLUMN IF NOT EXISTS draft_favorite_fun TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS draft_most_relaxed TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS draft_favorite_color TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS draft_favorite_artist TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS draft_favorite_food TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS draft_favorite_song TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS draft_favorite_place TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS draft_favorite_movie_or_show TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS draft_hidden_talent TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS draft_ideal_day TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS draft_display_name TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS draft_history TEXT NOT NULL DEFAULT '[]',
  ADD COLUMN IF NOT EXISTS published_favorite_fun TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS published_most_relaxed TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS published_favorite_color TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS published_favorite_artist TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS published_favorite_food TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS published_favorite_song TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS published_favorite_place TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS published_favorite_movie_or_show TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS published_hidden_talent TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS published_ideal_day TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS published_display_name TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS published_history TEXT NOT NULL DEFAULT '[]';

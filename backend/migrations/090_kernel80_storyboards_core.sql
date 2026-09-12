-- Kernel 80: Storyboards Core -- a reusable, server-authoritative,
-- shareable grid board: ordered columns, ordered rows grouped into bands,
-- cells at row x column holding zero or more ordered cards.
--
-- Ownership model (spec 1.7, 4.2): storyboards.owner_user_id is a plain
-- column checked separately from storyboard_grants -- ownership is never a
-- sentinel grant row. The spec is explicit that "the owner is not merely
-- another grant": owner holds capabilities (grant/revoke access, choose
-- granted role, archive, delete) no granted_role value expresses, and a
-- sentinel-row design would make ownership structurally revocable by
-- deleting a grants row. owner_user_id is ON DELETE RESTRICT (not CASCADE)
-- so a board can never silently vanish as a side effect of a user delete --
-- the Kernel 77 account-deletion blocker (backend/internal/identity) is
-- what clears ownership, by requiring every owned board be archived/deleted
-- first; RESTRICT is belt-and-suspenders against a blocker/execute race
-- turning into a silently lost board instead of a loud SQL error.
--
-- granted_role reuses the existing location_role enum (001_init.sql:
-- producer/director/cast/crew/audience) rather than inventing new role
-- names (spec 1.9's explicit instruction) -- this is a DB-enforced reuse,
-- not just a convention. There is no "owner" arm in this enum; owner status
-- lives only in storyboards.owner_user_id.
--
-- Kernel 80 is personal-only scope (confirmed decision, no Production/
-- Show/Module linkage): storyboards has no location_id/production_id/
-- show_id column at all, not even a nullable placeholder, so no later
-- kernel is tempted to half-wire through an unused column.
--
-- Band/row model (spec 1.6, 4.5): row order is two-level, not one flat
-- board-wide sequence -- storyboard_bands.sort_order orders the bands, and
-- storyboard_rows.sort_order_in_band orders a band's own rows within it.
-- Full visual row order for rendering is simply "bands by sort_order, then
-- each band's rows by sort_order_in_band." This makes every one of the
-- spec's band/row invariants true BY CONSTRUCTION, with no invariant-
-- checking Go code required: "every row belongs to exactly one band" is
-- band_id NOT NULL; "bands never overlap or share rows" is structurally
-- impossible since a row's position is only ever expressed relative to its
-- own band, never as a board-wide coordinate a second band could collide
-- with; reordering bands never touches row data; moving a row to another
-- band is just band_id + sort_order_in_band (appended at the target band's
-- end), never a board-wide renumber. This is deliberately simpler than a
-- single global row order with an application-enforced contiguity
-- invariant -- see Construction/Storyboards/storyboards-domain-model.md.
--
-- hidden_from_audience (spec 1.11, 1.12) is a plain BOOLEAN on
-- storyboard_cards, not the {"toRoles":[...],"privateTo":[...]} JSONB
-- shape used by scene_stage_elements.visibility. That JSONB shape exists to
-- express per-role visibility across a 5-role stage; Kernel 80's rule is
-- strictly binary (audience+cast vs. crew+), so a boolean is the more
-- honest match for what the spec actually asks for -- a deliberate,
-- explained deviation from the established convention, not an oversight.
--
-- storyboard_cards.author_user_id is nullable / ON DELETE SET NULL, unlike
-- board ownership -- cards are expected to outlive their author (spec
-- 12.2: authored cards remain with anonymized authorship on account
-- deletion). Account deletion reassigns this to the tombstone user rather
-- than leaving it NULL, matching how every other author-attribution field
-- in backend/internal/identity/account_deletion.go already behaves.
--
-- The 200-column technical safety limit (spec 1.4) is enforced in Go
-- (storyboards.AddColumn counts siblings and rejects with a clear
-- column_limit_exceeded error before insert), not a DB constraint -- it's
-- framed as a soft implementation safeguard with a specific client-facing
-- error, not a hard schema rule.
--
-- No optional activity/version table: storyboard_cards.version plus
-- updated_at/author_user_id already satisfies the spec's "does not require
-- full revision history" / optional-versioning clause (spec 4.7), and
-- adding one would widen this migration beyond the "one coherent migration
-- family" guardrail (spec 21).
--
-- Venue seed at the bottom follows migration 085's exact
-- WITH location_row/lot_row ... INSERT ... ON CONFLICT (lot_id, slug) DO
-- UPDATE shape. Map-tile visibility itself is granted separately in Go
-- (access/visibility.go's authenticated_surface UNION arm) -- the venues
-- row here is not what drives who sees the tile.

BEGIN;

CREATE TABLE IF NOT EXISTS storyboards (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_storyboards_owner ON storyboards(owner_user_id);

CREATE TABLE IF NOT EXISTS storyboard_grants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  storyboard_id UUID NOT NULL REFERENCES storyboards(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  granted_role location_role NOT NULL,
  granted_by UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (storyboard_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_storyboard_grants_storyboard ON storyboard_grants(storyboard_id);
CREATE INDEX IF NOT EXISTS idx_storyboard_grants_user ON storyboard_grants(user_id);

CREATE TABLE IF NOT EXISTS storyboard_columns (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  storyboard_id UUID NOT NULL REFERENCES storyboards(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  sort_order INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (storyboard_id, sort_order)
);

CREATE INDEX IF NOT EXISTS idx_storyboard_columns_storyboard_sort ON storyboard_columns(storyboard_id, sort_order);

CREATE TABLE IF NOT EXISTS storyboard_bands (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  storyboard_id UUID NOT NULL REFERENCES storyboards(id) ON DELETE CASCADE,
  label TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  sort_order INTEGER NOT NULL,
  is_collapsed BOOLEAN NOT NULL DEFAULT FALSE,
  is_locked BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (storyboard_id, sort_order)
);

CREATE INDEX IF NOT EXISTS idx_storyboard_bands_storyboard_sort ON storyboard_bands(storyboard_id, sort_order);

CREATE TABLE IF NOT EXISTS storyboard_rows (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  storyboard_id UUID NOT NULL REFERENCES storyboards(id) ON DELETE CASCADE,
  band_id UUID NOT NULL REFERENCES storyboard_bands(id) ON DELETE CASCADE,
  label TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  sort_order_in_band INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (band_id, sort_order_in_band)
);

CREATE INDEX IF NOT EXISTS idx_storyboard_rows_storyboard ON storyboard_rows(storyboard_id);
CREATE INDEX IF NOT EXISTS idx_storyboard_rows_band_sort ON storyboard_rows(band_id, sort_order_in_band);
CREATE INDEX IF NOT EXISTS idx_storyboard_rows_band ON storyboard_rows(band_id);

CREATE TABLE IF NOT EXISTS storyboard_cards (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  storyboard_id UUID NOT NULL REFERENCES storyboards(id) ON DELETE CASCADE,
  row_id UUID NOT NULL REFERENCES storyboard_rows(id) ON DELETE CASCADE,
  column_id UUID NOT NULL REFERENCES storyboard_columns(id) ON DELETE CASCADE,
  sort_order_in_cell INTEGER NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  front_text TEXT NOT NULL DEFAULT '',
  back_text TEXT NOT NULL DEFAULT '',
  category TEXT NOT NULL DEFAULT '',
  color_token TEXT NOT NULL DEFAULT '',
  hidden_from_audience BOOLEAN NOT NULL DEFAULT FALSE,
  is_locked BOOLEAN NOT NULL DEFAULT FALSE,
  author_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  version INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_storyboard_cards_cell ON storyboard_cards(row_id, column_id, sort_order_in_cell);
CREATE INDEX IF NOT EXISTS idx_storyboard_cards_storyboard ON storyboard_cards(storyboard_id);
CREATE INDEX IF NOT EXISTS idx_storyboard_cards_author ON storyboard_cards(author_user_id);

WITH location_row AS (
  SELECT id FROM locations WHERE is_default
),
lot_row AS (
  SELECT id FROM lots
  WHERE slug = 'main-lot'
    AND location_id = (SELECT id FROM location_row)
)
INSERT INTO venues (lot_id, name, slug, kind, config, is_public, is_workshop)
SELECT
  id,
  'Storyboards',
  'storyboards',
  'commons',
  '{
    "surface": "storyboards"
  }'::jsonb,
  FALSE,
  FALSE
FROM lot_row
ON CONFLICT (lot_id, slug) DO UPDATE
  SET name = EXCLUDED.name,
      kind = EXCLUDED.kind,
      is_public = EXCLUDED.is_public,
      is_workshop = EXCLUDED.is_workshop;

COMMIT;

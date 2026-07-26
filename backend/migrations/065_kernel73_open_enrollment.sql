BEGIN;

-- Real-world opening night (Socio at Catharsis) proved the Kernel 71
-- two-punch ticket flow, correct for scheduled/consent-based casting, is
-- unworkable for live walk-up attendance: a Director cannot manually punch
-- a ticket for every Player who shows up wanting to select a Character and
-- see the opening Scene. Audience already has a frictionless self-join
-- path (`audience_self_join_enabled`); Player never did. This adds the
-- equivalent per-Show-Run flag for Player self-join, deliberately named
-- distinctly from `audience_self_join_enabled` since it gates a different
-- role and a different authority path (SelfJoinAsPlayer, not
-- SelfJoinAsAudience). The ticket system itself is untouched and remains
-- the default everywhere this flag is off.
ALTER TABLE show_runs
  ADD COLUMN IF NOT EXISTS open_enrollment BOOLEAN NOT NULL DEFAULT FALSE;

COMMIT;

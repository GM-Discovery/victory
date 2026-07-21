-- Kernel 73 follow-up: reference-only catalog fields for equipment_items.
-- Purely informational -- nothing server-side computes, deducts, or gates
-- against cost_credits/stats_json; this is display data for a narrator or
-- player to read, the same status as short_description. Does not reopen
-- Kernel 73's locked "no automatic prices/currency" exclusion (spec S2.8/
-- S16): no wallet, no ledger, no purchase-time price enforcement exists or
-- is added here.
ALTER TABLE equipment_items
  ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT '';

ALTER TABLE equipment_items
  ADD COLUMN IF NOT EXISTS cost_credits NUMERIC;

-- stats_json is a flexible per-category bag (damage, damage_reduction,
-- tags, special, effect, and for weapons counters/vulnerable_to/
-- status_effect/key_skills from the Weapon Effectiveness Matrix) rather
-- than a rigid column per possible stat -- category shapes vary too much
-- (a weapon's damage die has nothing in common with a mount's travel
-- bonus) to force one schema. Rendered as-is; never parsed for branching
-- logic server-side.
ALTER TABLE equipment_items
  ADD COLUMN IF NOT EXISTS stats_json JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_equipment_items_category
  ON equipment_items(location_id, category);

-- Kernel 73: minimal reusable equipment/inventory model. No slots,
-- encumbrance, durability, resale, crafting, weight, price enforcement, or
-- vendor stock depletion (spec S5.1). Confirmed absent from the codebase by
-- pre-implementation audit -- assets/warehouse.go is image/token storage
-- keyed by producer/location with no quantity or mechanical fields, and no
-- inventory/possession concept exists anywhere else.
CREATE TABLE IF NOT EXISTS equipment_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE RESTRICT,
  production_id UUID REFERENCES productions(id) ON DELETE SET NULL,
  name TEXT NOT NULL,
  slug TEXT NOT NULL,
  image_asset_id UUID REFERENCES assets(id) ON DELETE SET NULL,
  short_description TEXT NOT NULL DEFAULT '',
  descriptors_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  quantity_mode TEXT NOT NULL DEFAULT 'stackable',
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT equipment_items_quantity_mode_check CHECK (quantity_mode IN ('stackable', 'unique')),
  UNIQUE (location_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_equipment_items_location_active
  ON equipment_items(location_id, active);

-- One row per (character, item): the character's current holding.
-- "unique" quantity_mode items are clamped to quantity=1 by application
-- logic (a repeat purchase is a no-op, not a second unit or an error).
-- source_* reflects the most recent acquisition context -- switching which
-- Character is currently selected for a Show Run never touches these rows,
-- since they are already keyed to the specific character_card_id an item
-- was purchased onto (spec S2.10's "historical acquisition metadata is not
-- rewritten" guarantee).
CREATE TABLE IF NOT EXISTS character_inventory_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  equipment_item_id UUID NOT NULL REFERENCES equipment_items(id) ON DELETE RESTRICT,
  quantity INTEGER NOT NULL DEFAULT 1,
  acquired_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  source_show_run_id UUID REFERENCES show_runs(id) ON DELETE SET NULL,
  source_show_id UUID REFERENCES shows(id) ON DELETE SET NULL,
  source_session_id UUID REFERENCES sessions(id) ON DELETE SET NULL,
  source_scene_placement_id UUID REFERENCES show_scene_placements(id) ON DELETE SET NULL,
  source_interaction_key TEXT NOT NULL DEFAULT '',
  first_acquired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT character_inventory_items_quantity_positive CHECK (quantity > 0),
  UNIQUE (character_card_id, equipment_item_id)
);

CREATE INDEX IF NOT EXISTS idx_character_inventory_items_character
  ON character_inventory_items(character_card_id, updated_at DESC);

-- Purchase-retry safety (spec S5.3), following the established idempotency
-- pattern in backend/internal/cues/execute.go's cue_executions table: a
-- dedicated attempt ledger, not a field on the holdings row, so a retried
-- request (same idempotency_key) can be detected and answered from the
-- ledger without double-applying the quantity change. A deliberate second
-- purchase (new idempotency_key) is free to increase quantity again.
CREATE TABLE IF NOT EXISTS character_inventory_purchase_attempts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  equipment_item_id UUID NOT NULL REFERENCES equipment_items(id) ON DELETE RESTRICT,
  idempotency_key TEXT NOT NULL,
  quantity_delta INTEGER NOT NULL DEFAULT 1,
  resulting_inventory_item_id UUID REFERENCES character_inventory_items(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (character_card_id, equipment_item_id, idempotency_key)
);

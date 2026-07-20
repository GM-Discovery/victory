-- Kernel 73: bounded merchant/program packet model. Reusable for a future
-- second merchant, not Kessa-specific at the schema level (spec S9.1: "The
-- structure should be reusable for another bounded merchant/program packet
-- without pretending to be a general dialogue graph"). Deliberately NOT a
-- node-graph dialogue system -- five fixed stance slots plus one Haggle
-- slot, authored disposition/response text, no branching.
CREATE TABLE IF NOT EXISTS merchant_packets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE RESTRICT,
  slug TEXT NOT NULL,
  display_name TEXT NOT NULL,
  portrait_asset_id UUID REFERENCES assets(id) ON DELETE SET NULL,
  intro_text TEXT NOT NULL DEFAULT '',
  -- stance_dispositions_json: {"command": {"disposition":"reject", "responses":[...]}, ...}
  -- one entry per of the five fixed stances (command/convince/insight/
  -- follow/sympathize); "responses" is an ordered list of response-text
  -- variants selected by roll-result tier (low/mid/high), never by
  -- pass/fail -- the disposition itself is fixed per spec S2.6.
  stance_dispositions_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  haggle_skill_key TEXT NOT NULL DEFAULT 'haggle',
  haggle_target_value INTEGER NOT NULL DEFAULT 5,
  haggle_skilled_die TEXT NOT NULL DEFAULT 'd6',
  haggle_unskilled_die TEXT NOT NULL DEFAULT 'd4',
  haggle_success_text TEXT NOT NULL DEFAULT '',
  haggle_failure_text TEXT NOT NULL DEFAULT '',
  return_label TEXT NOT NULL DEFAULT 'Return to Conversation',
  close_label TEXT NOT NULL DEFAULT 'Leave the Shop',
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (location_id, slug)
);

CREATE TABLE IF NOT EXISTS merchant_packet_equipment_items (
  packet_id UUID NOT NULL REFERENCES merchant_packets(id) ON DELETE CASCADE,
  equipment_item_id UUID NOT NULL REFERENCES equipment_items(id) ON DELETE CASCADE,
  sort_order INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (packet_id, equipment_item_id)
);

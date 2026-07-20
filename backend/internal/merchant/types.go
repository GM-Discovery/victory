// Package merchant is Kernel 73's participant-local Equip Mode: a
// Director-authored interaction attached to one Show Scene Placement that
// opens a private Program Panel for the triggering Player only, where they
// speak with a bounded merchant packet (Kessa) through five authored
// stances plus a Haggle skill check, and purchase durable Character
// inventory. Not a general dialogue engine, NPC AI, or economy.
package merchant

import "time"

// EquipmentItem is the smallest reusable purchasable definition: no slots,
// encumbrance, durability, resale, crafting, weight, or price enforcement
// (kernel-73 spec S5.1).
type EquipmentItem struct {
	ID               string    `json:"id"`
	LocationID       string    `json:"location_id"`
	ProductionID     string    `json:"production_id,omitempty"`
	Name             string    `json:"name"`
	Slug             string    `json:"slug"`
	ImageAssetID     string    `json:"image_asset_id,omitempty"`
	ShortDescription string    `json:"short_description"`
	DescriptorsJSON  []string  `json:"descriptors"`
	QuantityMode     string    `json:"quantity_mode"`
	Active           bool      `json:"active"`
	CreatedByUserID  string    `json:"created_by_user_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

const (
	QuantityModeStackable = "stackable"
	QuantityModeUnique    = "unique"
)

// InventoryEntry is a Character's current holding of one EquipmentItem --
// one row per (character, item); quantity increments on repeat stackable
// purchase. Never rewritten when the Player later switches which Character
// is selected for the Show Run (kernel-73 spec S2.10) -- it stays keyed to
// the specific character_card_id it was purchased onto.
type InventoryEntry struct {
	ID                     string    `json:"id"`
	CharacterCardID        string    `json:"character_card_id"`
	EquipmentItemID        string    `json:"equipment_item_id"`
	Quantity               int       `json:"quantity"`
	AcquiredByUserID       string    `json:"acquired_by_user_id"`
	SourceShowRunID        string    `json:"source_show_run_id,omitempty"`
	SourceShowID           string    `json:"source_show_id,omitempty"`
	SourceSessionID        string    `json:"source_session_id,omitempty"`
	SourceScenePlacementID string    `json:"source_scene_placement_id,omitempty"`
	SourceInteractionKey   string    `json:"source_interaction_key,omitempty"`
	FirstAcquiredAt        time.Time `json:"first_acquired_at"`
	UpdatedAt              time.Time `json:"updated_at"`

	// Item is populated by read paths that join equipment_items for display
	// (the Inventory page, Equip Mode's confirmation) -- never set on write.
	Item *EquipmentItem `json:"item,omitempty"`
}

// StanceDisposition is one of the five fixed, authored merchant stances.
// Disposition never flips on roll result (kernel-73 spec S2.6) -- only
// which authored response variant is shown may vary.
type StanceDisposition struct {
	Disposition string   `json:"disposition"` // "reject" | "positive" | "neutral"
	Responses   []string `json:"responses"`
}

// MerchantPacket is the bounded, reusable merchant/program packet model
// (kernel-73 spec S9.1) -- five fixed stance slots plus one Haggle slot,
// authored text, no branching dialogue graph.
type MerchantPacket struct {
	ID                 string                       `json:"id"`
	LocationID         string                       `json:"location_id"`
	Slug               string                       `json:"slug"`
	DisplayName        string                       `json:"display_name"`
	PortraitAssetID    string                       `json:"portrait_asset_id,omitempty"`
	IntroText          string                       `json:"intro_text"`
	StanceDispositions map[string]StanceDisposition `json:"stance_dispositions"`
	HaggleSkillKey     string                       `json:"haggle_skill_key"`
	HaggleTargetValue  int                          `json:"haggle_target_value"`
	HaggleSkilledDie   string                       `json:"haggle_skilled_die"`
	HaggleUnskilledDie string                       `json:"haggle_unskilled_die"`
	HaggleSuccessText  string                       `json:"haggle_success_text"`
	HaggleFailureText  string                       `json:"haggle_failure_text"`
	ReturnLabel        string                       `json:"return_label"`
	CloseLabel         string                       `json:"close_label"`
	Active             bool                         `json:"active"`
	CreatedAt          time.Time                    `json:"created_at"`
	UpdatedAt          time.Time                    `json:"updated_at"`

	// Stock is populated by read paths that join merchant_packet_equipment_items.
	Stock []EquipmentItem `json:"stock,omitempty"`
}

// FixedStanceKeys is the exhaustive, ordered list of stance option keys
// (kernel-73 spec S2.5) -- exactly five, never a branching tree.
var FixedStanceKeys = []string{"command", "convince", "insight", "follow", "sympathize"}

// ParticipantInteraction is a Director-authored, participant-local entry
// point attached to one Show Scene Placement. Deliberately its own table
// rather than folded into show_scene_placements.config_json, mirroring how
// Cues (migration 045) got their own table for the same reason: a
// participant interaction has its own authority/enablement/ordering
// lifecycle distinct from placement config. The structural difference from
// a Cue: a Cue is shared/role-scoped (every eligible viewer sees and can
// press the SAME button); a participant interaction opens a Program for
// ONLY the triggering Player.
type ParticipantInteraction struct {
	ID                   string         `json:"id"`
	ShowScenePlacementID string         `json:"show_scene_placement_id"`
	InternalName         string         `json:"internal_name"`
	StageButtonLabel     string         `json:"stage_button_label"`
	InteractionType      string         `json:"interaction_type"`
	ConfigurationJSON    map[string]any `json:"configuration_json"`
	Enabled              bool           `json:"enabled"`
	SortOrder            int            `json:"sort_order"`
	CreatedByUserID      string         `json:"created_by_user_id,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

const InteractionTypeOpenEquipMode = "open_equip_mode"

// PlayerVisibleInteraction is the curated, player-facing stage-button
// listing -- mirrors cues.PlayerVisibleCue's shape.
type PlayerVisibleInteraction struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

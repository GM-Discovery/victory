package scenes

import (
	"encoding/json"
	"time"
)

// StageElement is one row of scene_stage_elements -- a single placed piece
// of a Scene's visual composition (a token, an index card, the map/
// backdrop, or the grid config). ShowScenePlacementID is nil for a
// Base-layer element (the Scene's own reusable default, visible in every
// Show that stages this Scene) and set for a Show-layer element (an
// addition/override scoped to exactly one Show's placement of the Scene).
// Kernel 73A S4.
type StageElement struct {
	ID                   string          `json:"id"`
	SceneID              string          `json:"scene_id"`
	ShowScenePlacementID *string         `json:"show_scene_placement_id,omitempty"`
	Kind                 string          `json:"kind"`
	Label                string          `json:"label,omitempty"`
	Data                 json.RawMessage `json:"data,omitempty"`
	Position             json.RawMessage `json:"position,omitempty"`
	Visibility           json.RawMessage `json:"visibility,omitempty"`
	SortOrder            int             `json:"sort_order"`
	CreatedByUserID      *string         `json:"created_by_user_id,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`

	// Layer is a derived, read-only convenience ("base" or "show") --
	// never stored, always computed from ShowScenePlacementID.
	Layer string `json:"layer"`

	// Binding is populated by read paths that resolve this element's
	// stage_element_bindings row, if any (nil when unbound).
	Binding *StageElementBinding `json:"binding,omitempty"`
}

const (
	StageElementKindToken       = "token"
	StageElementKindIndexCard   = "index_card"
	StageElementKindMapBackdrop = "map_backdrop"
	StageElementKindGridConfig  = "grid_config"
)

// ValidStageElementKinds is the exhaustive list enforced by the DB's CHECK
// constraint, mirrored here so a bad kind is rejected before ever hitting
// the database.
var ValidStageElementKinds = map[string]bool{
	StageElementKindToken:       true,
	StageElementKindIndexCard:   true,
	StageElementKindMapBackdrop: true,
	StageElementKindGridConfig:  true,
}

// CreateStageElementInput is the caller-supplied subset of a new element.
type CreateStageElementInput struct {
	Kind       string
	Label      string
	Data       map[string]any
	Position   map[string]any
	Visibility map[string]any
	SortOrder  int
}

// UpdateStageElementPatch carries only the fields being changed.
type UpdateStageElementPatch struct {
	Label      *string
	Data       *map[string]any
	Position   *map[string]any
	Visibility *map[string]any
	SortOrder  *int
}

// StageElementBinding is one row of stage_element_bindings -- binds a
// placed composition element to an existing participant_interactions row
// (Kernel 73A S6). Exactly one binding_type exists today.
type StageElementBinding struct {
	ID                       string    `json:"id"`
	SceneStageElementID      string    `json:"scene_stage_element_id"`
	BindingType              string    `json:"binding_type"`
	ParticipantInteractionID string    `json:"participant_interaction_id"`
	CreatedByUserID          *string   `json:"created_by_user_id,omitempty"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

const BindingTypeParticipantInteraction = "participant_interaction"

// ResolvedComposition is the merged Base+Show layer view for one
// placement -- what Preview as Player renders and what gets projected
// into a live session when its Scene becomes current (Kernel 73A S4, S7,
// the SetCurrentScenePlacement gap).
type ResolvedComposition struct {
	SceneID     string         `json:"scene_id"`
	PlacementID string         `json:"placement_id"`
	Elements    []StageElement `json:"elements"`
}

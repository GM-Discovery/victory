package cues

import (
	"encoding/json"
	"time"
)

// Cue is one row of cues -- a stable, Show-Scene-Placement-scoped backstage
// control (Kernel 70 §6.1). The visible stage-button label is never the
// Cue's identity; the button always triggers the underlying Cue by id.
type Cue struct {
	ID                   string          `json:"id"`
	ShowScenePlacementID string          `json:"show_scene_placement_id"`
	InternalName         string          `json:"internal_name"`
	StageButtonLabel     string          `json:"stage_button_label,omitempty"`
	TriggerScope         string          `json:"trigger_scope"`
	SortOrder            int             `json:"sort_order"`
	Enabled              bool            `json:"enabled"`
	Actions              json.RawMessage `json:"actions"`
	CreatedByUserID      *string         `json:"created_by_user_id,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

// Trigger scopes (Kernel 70 §1.7). Audience may never trigger a Cue under
// any scope -- CanTriggerCue enforces this as a hard floor regardless of
// which of these three values is stored.
const (
	TriggerScopeDirectorCrewOnly  = "director_crew_only"
	TriggerScopePlayersMayTrigger = "players_may_trigger"
	TriggerScopeAnyRosterMember   = "any_roster_member"
)

func isValidTriggerScope(scope string) bool {
	switch scope {
	case TriggerScopeDirectorCrewOnly, TriggerScopePlayersMayTrigger, TriggerScopeAnyRosterMember:
		return true
	default:
		return false
	}
}

// Cue action types (Kernel 70 §6.3).
//
// The first three shipped with Kernel 70. The four visibility actions were
// deferred there for a stated reason -- there was no canonical object
// identity to target, so implementing them would have meant extending
// world/snapshot.go's per-session visibility-layer derivation into a second
// show-scoped source, i.e. building the second object model that kernel doc
// told us to avoid.
//
// Kernel 90 removed that blocker rather than working around it. There is now
// one canonical object identity (stageobjects.Ref) and one canonical state
// path (stageobjects.ApplyMutation), and these four actions call exactly the
// mutation the manual Director controls call. That shared call is what makes
// Kernel 90 §24's manual/Cue parity structural instead of a coincidence
// maintained by tests.
const (
	ActionTypeGoToScene       = "go_to_scene"
	ActionTypeEmitGameEvent   = "emit_game_event"
	ActionTypeSetShowVariable = "set_show_variable"
	ActionTypeRevealObject       = "reveal_object"
	ActionTypeHideObject         = "hide_object"
	ActionTypeEnableInteraction  = "enable_interaction"
	ActionTypeDisableInteraction = "disable_interaction"
)

// CueAction is one entry in a Cue's ordered actions array. Exactly one of
// the typed fields is set, matching Type. Decoded lazily via
// encoding/json's tolerant unmarshal (unset fields simply stay nil),
// matching this codebase's config_json json.RawMessage passthrough
// precedent rather than a stricter discriminated-union decode.
type CueAction struct {
	Type            string                 `json:"type"`
	GoToScene       *GoToSceneAction       `json:"go_to_scene,omitempty"`
	EmitGameEvent   *EmitGameEventAction   `json:"emit_game_event,omitempty"`
	SetShowVariable *SetShowVariableAction `json:"set_show_variable,omitempty"`
	// StageObject carries the target for all four Kernel 90 visibility
	// actions. One field rather than four, because the four actions differ
	// only in which canonical operation they request -- the target shape is
	// identical, and four identical structs would invite them to drift.
	StageObject *StageObjectAction `json:"stage_object,omitempty"`
}

// StageObjectAction targets one durable stage object by canonical identity
// (Kernel 90 §22).
//
// ObjectKind + ObjectID only. Deliberately no selector, no label, no
// coordinate: §54 makes a DOM selector or screen coordinate as Cue identity
// a FAIL condition, and the reason is practical rather than stylistic -- a
// Cue authored today must still resolve after the stage is re-laid out, the
// object is moved, or its label is edited.
type StageObjectAction struct {
	ObjectKind string `json:"object_kind"`
	ObjectID   string `json:"object_id"`
}

type GoToSceneAction struct {
	ShowScenePlacementID string `json:"show_scene_placement_id"`
}

type EmitGameEventAction struct {
	EventKind string         `json:"event_kind"`
	Detail    map[string]any `json:"detail,omitempty"`
}

type SetShowVariableAction struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// CreateCueInput is the caller-supplied subset of a new Cue.
// ShowScenePlacementID is a separate function argument, matching this
// codebase's convention elsewhere (scenes.CreateSceneInput, etc.).
type CreateCueInput struct {
	InternalName     string
	StageButtonLabel string
	TriggerScope     string
	SortOrder        int
	Enabled          *bool // nil defaults to true
	Actions          []CueAction
}

// UpdateCuePatch carries only the fields being changed.
type UpdateCuePatch struct {
	InternalName     *string
	StageButtonLabel *string
	TriggerScope     *string
	SortOrder        *int
	Enabled          *bool
	Actions          *[]CueAction
}

// ActionResult is the per-action outcome recorded in a Cue execution.
type ActionResult struct {
	ActionIndex int    `json:"action_index"`
	Type        string `json:"type"`
	Status      string `json:"status"` // succeeded | failed
	Error       string `json:"error,omitempty"`
}

// PlayerVisibleCue is the curated shape for the player-facing stage
// button listing (Kernel 70 §6.2, §8.2) -- only enabled Cues the viewer
// can actually trigger, with only the button label (falling back to
// internal_name only when no stage_button_label is set -- the internal
// name is meant to be readable if it leaks, but is never shown to a
// viewer who wouldn't otherwise see backstage Cue detail). No
// trigger_scope, actions, created_by_user_id, or internal timestamps ever
// appear here.
type PlayerVisibleCue struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// CueExecutionResult is what a GO press returns to the caller and what is
// replayed verbatim on an idempotent repeat request. ShowID is included so
// HTTP callers can broadcast a show-scoped invalidation without a second
// lookup.
type CueExecutionResult struct {
	ExecutionID   string         `json:"execution_id"`
	CueID         string         `json:"cue_id"`
	ShowID        string         `json:"-"`
	Status        string         `json:"status"`
	ActionResults []ActionResult `json:"action_results"`
}

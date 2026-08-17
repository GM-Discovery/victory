// Package stageobjects is Victory's single answer to "who can currently
// perceive or interact with this thing on stage?" (Kernel 90).
//
// It owns one canonical object-state model over the durable Pixi-rendered
// objects Victory already has, and exactly one server-side mutation path
// through it. Manual Director controls (HandleObjectState) and Cue actions
// (cues.executeOneAction) both call ApplyMutation -- that shared call is
// what Kernel 90 §24's manual/Cue parity requirement reduces to in code,
// rather than two implementations kept in agreement by discipline.
//
// What this package deliberately is not:
//
//   - Not fog-of-war (§46). There is no geometry here, no line of sight, no
//     lighting, no occlusion. Perception is per-object and boolean.
//   - Not a rule engine (§48). Nothing in this package evaluates a
//     condition, watches for a trigger, or decides on its own to change
//     state. Every mutation has a named actor who asked for it.
//   - Not an ACL system (§34). There are four theatrical scope kinds and
//     they resolve through existing Show/Cohort/Character authority.
//   - Not a Scene concept (§2). This package never writes to scenes or
//     scene_stage_elements. A Scene describes the stage; this describes who
//     perceives it.
//
// The map is out of scope by product direction (§3): map_backdrop elements
// are refused by ResolveRef, because the map already has its own
// disappear/remove behavior and "fly to another Scene" is the intended
// theatrical operation for radically changing the stage.
package stageobjects

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier is satisfied by both *pgxpool.Pool and pgx.Tx, matching
// rollaudience.Querier so a caller already holding either can reuse it.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Execer adds the write half. Mutations need it; projection does not, which
// is why they are separate interfaces -- a read path physically cannot
// change state through this package.
type Execer interface {
	Querier
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Object kinds. These are the four durable object families Kernel 90 §5
// asked to be discovered from the repository, and they exhaust what Victory
// renders durably onto the Pixi stage today.
const (
	// KindVenueLayoutElement is a live warehouse stage object: an `elements`
	// row placed through `venue_layout_elements`. What Add Token / Create
	// Index Card produce during a live session.
	KindVenueLayoutElement = "venue_layout_element"
	// KindSceneStageElement is a Kernel 73A Scene-authored composition
	// element (token or index_card).
	KindSceneStageElement = "scene_stage_element"
	// KindDrawingObject is a Kernel 87 drawing object.
	KindDrawingObject = "drawing_object"
	// KindParticipantInteraction carries interaction state for an
	// interaction reached through a stage_element_bindings row. Its
	// visibility is not settable -- it follows the element it is bound to.
	KindParticipantInteraction = "participant_interaction"
)

// Visibility states (§7). Two states, and hidden is emphatically not
// deleted: the row keeps existing, keeps its identity, and can be revealed
// again. Deletion stays each kind's own separate operation.
const (
	VisibilityVisible = "visible"
	VisibilityHidden  = "hidden"
)

// Scope kinds (§9). Director+ is absent on purpose: Director-only is a
// hidden object with no grants, and giving one state two spellings is
// exactly the kind of drift §0 forbids.
const (
	// ScopeCast is every ordinary Show participant but NOT the Audience.
	// This is what makes "the players can see it, the house cannot"
	// expressible, which §18 requires and a base visibility flag alone
	// cannot say.
	ScopeCast = "cast"
	// ScopeCohort is one show_cohorts row (§16).
	ScopeCohort = "cohort"
	// ScopeCharacter is one character_cards row, matched against the
	// viewer's currently SELECTED Character (§17).
	ScopeCharacter = "character"
	// ScopeAudience is the Audience tier (§18).
	ScopeAudience = "audience"
)

// Ref is the canonical stage-object reference (§5/§22). Kind plus a stable
// database id -- never a DOM selector, never a canvas coordinate, never a
// display label, all three of which §54 makes a FAIL condition for Cue
// targeting.
type Ref struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func (r Ref) normalized() Ref {
	return Ref{
		Kind: strings.ToLower(strings.TrimSpace(r.Kind)),
		ID:   strings.TrimSpace(r.ID),
	}
}

// IsSupportedKind reports whether kind is one of the four canonical durable
// object kinds. Exported so Cue authoring can refuse an unsupported target
// before the Cue is ever saved, rather than only at fire time.
func IsSupportedKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case KindVenueLayoutElement, KindSceneStageElement, KindDrawingObject, KindParticipantInteraction:
		return true
	default:
		return false
	}
}

func (r Ref) validate() error {
	n := r.normalized()
	if n.ID == "" {
		return errors.New("object_id_required")
	}
	switch n.Kind {
	case KindVenueLayoutElement, KindSceneStageElement, KindDrawingObject, KindParticipantInteraction:
		return nil
	case "":
		return errors.New("object_kind_required")
	default:
		return errors.New("unsupported_object_kind")
	}
}

// SupportsVisibility reports whether this kind's visibility can be set
// directly. A participant interaction's visibility follows the element it
// is bound to, so setting it here would create a second, competing answer
// for the same question.
func (r Ref) SupportsVisibility() bool {
	return r.normalized().Kind != KindParticipantInteraction
}

// SupportsInteraction reports whether this kind has an interaction
// dimension at all. §12 requires the Director menu not to offer Interaction
// controls for objects that cannot be acted upon; this is the predicate
// behind that.
//
// Only participant interactions carry it. A token is not itself
// interactable -- it becomes interactable by being bound to a participant
// interaction, and that binding is the thing that gets enabled or disabled.
func (r Ref) SupportsInteraction() bool {
	return r.normalized().Kind == KindParticipantInteraction
}

// State is one canonical object-state row plus its scope grants.
//
// An absent State means "authored default", which for every kind is
// visible and enabled (§10). Callers must therefore treat "no row" and
// "visible with no grants" identically -- Projector does, and that is why
// placing an object costs no write here.
type State struct {
	ID       string `json:"id"`
	ShowID   string `json:"show_id"`
	Ref      Ref    `json:"object_ref"`
	Visibility string `json:"visibility"`
	// InteractionEnabled is nil for kinds with no interaction dimension.
	InteractionEnabled *bool   `json:"interaction_enabled,omitempty"`
	Scopes             []Scope `json:"scopes"`
	UpdatedByUserID    string  `json:"updated_by_user_id,omitempty"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Hidden reports whether this state hides the object from ordinary viewers.
func (s State) Hidden() bool {
	return strings.EqualFold(strings.TrimSpace(s.Visibility), VisibilityHidden)
}

// Scope is one grant: an audience that may perceive an otherwise-hidden
// object. ScopeID is empty for ScopeCast and ScopeAudience, which name a
// tier rather than a row.
type Scope struct {
	Kind string `json:"scope_kind"`
	ID   string `json:"scope_id,omitempty"`
}

func (s Scope) normalized() Scope {
	return Scope{
		Kind: strings.ToLower(strings.TrimSpace(s.Kind)),
		ID:   strings.TrimSpace(s.ID),
	}
}

func (s Scope) validate() error {
	n := s.normalized()
	switch n.Kind {
	case ScopeCast, ScopeAudience:
		if n.ID != "" {
			return errors.New("scope_id_not_allowed_for_kind")
		}
		return nil
	case ScopeCohort, ScopeCharacter:
		if n.ID == "" {
			return errors.New("scope_id_required_for_kind")
		}
		return nil
	case "":
		return errors.New("scope_kind_required")
	default:
		return errors.New("unsupported_scope_kind")
	}
}

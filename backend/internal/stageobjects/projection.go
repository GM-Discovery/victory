package stageobjects

import (
	"context"
	"strings"

	"victory/backend/internal/rollaudience"
)

// Viewer is one fully-resolved perceiver. Every field is derived
// server-side; nothing here may come from a client claim (§13: "the client
// must not decide hidden-state authority", §17: "do not trust a
// client-supplied arbitrary Character id as authority").
type Viewer struct {
	UserID string
	// Backstage is the Director/Producer/Operator/Crew tier. A backstage
	// viewer perceives hidden objects (§14) so a Director can direct; the
	// caller is responsible for marking them visually distinct, which
	// HiddenFor exists to support.
	Backstage bool
	// Audience is the audience tier -- watching, not playing.
	Audience bool
	// CohortID is the viewer's show_cohorts assignment, or "" if Ungrouped.
	CohortID string
	// SelectedCharacterID is the Character the viewer currently has
	// selected, not merely one they own. Switching Character therefore
	// correctly changes what they perceive, matching how Kernel 74's local
	// projection is already scoped.
	SelectedCharacterID string
}

// Projector answers perception questions for one viewer against one Show's
// canonical state. Built once per snapshot read and then queried per object,
// so a stage with fifty objects costs the same two queries as an empty one.
type Projector struct {
	viewer Viewer
	states map[Ref]State
}

// ProjectorFor loads a Show's canonical state and binds it to a viewer.
func ProjectorFor(ctx context.Context, q Querier, showID string, viewer Viewer) (*Projector, error) {
	states, err := LoadShowStates(ctx, q, showID)
	if err != nil {
		return nil, err
	}
	return &Projector{viewer: viewer, states: states}, nil
}

// NewProjector builds a Projector from already-loaded state, for callers
// that resolve several viewers against one Show (the browser proof's
// multi-viewer assertions, and any future batch projection) without
// re-querying per viewer.
func NewProjector(viewer Viewer, states map[Ref]State) *Projector {
	if states == nil {
		states = map[Ref]State{}
	}
	return &Projector{viewer: viewer, states: states}
}

// Viewer returns the bound viewer.
func (p *Projector) Viewer() Viewer { return p.viewer }

// CanPerceive reports whether this viewer may receive the object at all.
//
// A false answer means the object must be OMITTED from the payload, not
// shipped with a flag the client is trusted to honour. §36 makes that
// explicit and §54 makes the alternative a FAIL condition; the same
// omit-don't-flag discipline Kernel 74 already applies to milestone-gated
// elements in world/snapshot.go.
func (p *Projector) CanPerceive(ref Ref) bool {
	if p == nil {
		return true
	}
	st, ok := p.states[ref.normalized()]
	// No row means authored default, which is visible (§10). This is why
	// placing an object requires no state write and why newly placed objects
	// are never accidentally Director-only.
	if !ok || !st.Hidden() {
		return true
	}
	// §14: the Director's working stage keeps hidden objects. A dedicated
	// "preview as audience" path gets truthful projection by constructing a
	// Viewer with Backstage false rather than by a cosmetic client filter,
	// which is what §15 means by "the same canonical projection path".
	if p.viewer.Backstage {
		return true
	}
	return p.matchesAnyScope(st.Scopes)
}

// HiddenFor reports whether the object is hidden from ordinary viewers,
// regardless of whether THIS viewer may perceive it.
//
// This is the Director-facing signal behind §14's visual distinction: the
// Director both receives the object (CanPerceive true) and knows to render
// it as backstage-only (HiddenFor true). It is only ever meaningful to send
// to a viewer who is already permitted to perceive the object, so it leaks
// nothing.
func (p *Projector) HiddenFor(ref Ref) bool {
	if p == nil {
		return false
	}
	st, ok := p.states[ref.normalized()]
	return ok && st.Hidden()
}

// ScopesFor returns the grant set for an object, for Director UI only.
//
// Callers MUST gate this on the viewer being backstage. The grant set is
// itself sensitive: telling Cohort B that an object is revealed to Cohort A
// is the §35 "Cohort A cannot infer Cohort B-only object metadata" leak,
// even when the object itself is correctly withheld.
func (p *Projector) ScopesFor(ref Ref) []Scope {
	if p == nil || !p.viewer.Backstage {
		return nil
	}
	st, ok := p.states[ref.normalized()]
	if !ok {
		return nil
	}
	return st.Scopes
}

// InteractionEnabled reports the SHOW-SCOPED half of whether an interaction
// may be invoked. Absent state means enabled (§10).
//
// Callers must AND this with the interaction's own global
// participant_interactions.enabled authoring flag. Those two booleans answer
// different questions -- "does this interaction exist at all" versus "has
// the Director switched it off for this Show" -- and the migration comment on
// interaction_enabled records why collapsing them would let a Director
// disabling Kessa mid-scene disable Kessa in the tutorial for everyone.
func (p *Projector) InteractionEnabled(ref Ref) bool {
	if p == nil {
		return true
	}
	st, ok := p.states[ref.normalized()]
	if !ok || st.InteractionEnabled == nil {
		return true
	}
	return *st.InteractionEnabled
}

// matchesAnyScope resolves the grant set against this viewer.
//
// Grants are exceptions to hidden, never restrictions on visible: a visible
// object ignores grants entirely (CanPerceive returns before reaching here).
// That is what keeps §10's default cheap and keeps "visible" from quietly
// meaning "visible to some people".
func (p *Projector) matchesAnyScope(scopes []Scope) bool {
	for _, raw := range scopes {
		s := raw.normalized()
		switch s.Kind {
		case ScopeCast:
			// Ordinary Show participants, explicitly not the Audience --
			// this is the grant that expresses "the players see it, the
			// house does not" (§18).
			if !p.viewer.Audience && p.viewer.UserID != "" {
				return true
			}
		case ScopeAudience:
			if p.viewer.Audience {
				return true
			}
		case ScopeCohort:
			// An Ungrouped viewer has no CohortID and must not match a
			// cohort grant; the empty-string guard is what prevents
			// "" == "" from silently revealing every cohort-scoped object
			// to everyone outside a cohort.
			if p.viewer.CohortID != "" && p.viewer.CohortID == s.ID {
				return true
			}
		case ScopeCharacter:
			if p.viewer.SelectedCharacterID != "" && p.viewer.SelectedCharacterID == s.ID {
				return true
			}
		}
	}
	return false
}

// ResolveViewer derives a Viewer from a live session, Show and user id.
//
// Cast-vs-Audience is resolved from SHOW PARTICIPATION, not from the passed-in
// role string, and that is deliberate rather than defensive. Kernel 90 §13
// lists "Show participation" as a projection input in its own right, and §32
// says to use current Show/Show Run authority rather than legacy session
// identity. Both point at the Show Run roster.
//
// It is also load-bearing here, because the role string is not trustworthy for
// this question on the live venue path. cmd/victory's lookupVenueRole calls
// participation.ResolveParticipationContext with an EMPTY showRunID hint, so
// that resolver's step 3 (show_run_roster_members) never runs and every roster
// Player falls through to its step 4 fallback and resolves to "audience". The
// pre-existing symptom of the same gap is recorded in loadCompositionRows'
// comment about Kessa's token vanishing "for any viewer whose session role
// resolved to audience".
//
// Deriving Cast from the roster instead means Kernel 90's cast-vs-audience
// scoping is correct today without changing that shared resolver -- which is
// used by many venues and is squarely outside this kernel. The upstream gap is
// recorded as a found-not-fixed finding rather than silently worked around: a
// Director scoping to Cast gets the right answer here regardless.
//
// selectedCharacterID is passed in rather than looked up because callers on the
// snapshot path (world.LoadVenueSnapshot) have already resolved the viewer's
// selected Character through resolveTheaterContext, and re-querying it here
// would be a second answer to a question already answered.
func ResolveViewer(ctx context.Context, q Querier, sessionID, showID, userID, role, selectedCharacterID string) (Viewer, error) {
	v := Viewer{
		UserID:              strings.TrimSpace(userID),
		SelectedCharacterID: strings.TrimSpace(selectedCharacterID),
		Backstage:           IsBackstageRole(role),
	}
	if v.Backstage {
		// A backstage viewer perceives everything, so neither their cohort nor
		// their cast membership can change the outcome. Skip both lookups.
		return v, nil
	}
	if v.UserID == "" {
		// An anonymous onlooker is neither Cast nor a known Audience member. It
		// matches no grant at all, which is the safe direction.
		return v, nil
	}

	cast, err := isShowCast(ctx, q, showID, v.UserID)
	if err != nil {
		return Viewer{}, err
	}
	// Audience is the complement of Cast among non-backstage viewers: someone
	// watching this Show without being in its cast. A viewer with no Show at
	// all (a venue outside a Show) is treated as Audience for scoping purposes,
	// since there is no roster that could make them Cast.
	v.Audience = !cast

	if sessionID == "" {
		return v, nil
	}
	decision, err := rollaudience.Resolve(ctx, q, sessionID, v.UserID, rollaudience.ModeCohort)
	if err != nil {
		return Viewer{}, err
	}
	v.CohortID = decision.CohortID
	return v, nil
}

// isShowCast reports whether the user holds an active Show Run roster row for
// this Show -- the Kernel 83+ authority, reached through shows.show_run_id.
//
// Any active roster role counts as Cast, including crew: the distinction this
// answers is "in the company" versus "in the house", not which job they do.
// Backstage roles never reach here (ResolveViewer returns first), so a Director
// is not being classified as Cast by this query.
func isShowCast(ctx context.Context, q Querier, showID, userID string) (bool, error) {
	showID = strings.TrimSpace(showID)
	if showID == "" {
		return false, nil
	}
	var ok bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM show_run_roster_members srrm
			JOIN shows sh ON sh.show_run_id = srrm.show_run_id
			WHERE sh.id = $1::uuid
			  AND srrm.user_id = $2::uuid
			  AND srrm.removed_at IS NULL
		)
	`, showID, userID).Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}

// IsBackstageRole reports the Kernel 70A backstage tier. Duplicated from
// world/snapshot.go's isBackstageRole rather than imported because world
// imports this package, and the set is a three-line closed list rather than
// logic that could drift meaningfully.
func IsBackstageRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "producer", "director", "operator", "crew":
		return true
	default:
		return false
	}
}

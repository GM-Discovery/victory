package stageobjects

import "testing"

// These are pure tests over the projection decision -- no database, no
// fixtures. Kernel 90 §38's list is mostly a list of SEMANTIC claims
// ("hidden != deleted", "Cohort A sees it, Cohort B does not"), and semantics
// are best pinned where they cannot be confounded by fixture drift. The
// dbtests alongside prove the storage and the manual/Cue parity that these
// cannot.

func ref(id string) Ref {
	return Ref{Kind: KindSceneStageElement, ID: id}
}

func states(entries ...State) map[Ref]State {
	out := map[Ref]State{}
	for _, e := range entries {
		out[e.Ref.normalized()] = e
	}
	return out
}

func hiddenWith(id string, scopes ...Scope) State {
	return State{Ref: ref(id), Visibility: VisibilityHidden, Scopes: scopes}
}

func TestAbsentStateIsVisibleToEveryone(t *testing.T) {
	// §10: placing an object must not make it Director-only. An object with
	// no state row at all is the overwhelmingly common case and must be
	// perceivable by every tier without any write having happened.
	empty := map[Ref]State{}
	for name, v := range map[string]Viewer{
		"player":    {UserID: "u1"},
		"audience":  {UserID: "u2", Audience: true},
		"director":  {UserID: "u3", Backstage: true},
		"anonymous": {},
	} {
		if !NewProjector(v, empty).CanPerceive(ref("obj")) {
			t.Errorf("%s could not perceive an object with no state row", name)
		}
	}
}

func TestHiddenWithNoGrantsIsDirectorOnly(t *testing.T) {
	// §41's "Director-only object" case, and the reason there is no separate
	// 'director' scope kind: hidden-with-no-grants already IS that state.
	s := states(hiddenWith("obj"))

	if NewProjector(Viewer{UserID: "player"}, s).CanPerceive(ref("obj")) {
		t.Error("a Player perceived a hidden object with no grants")
	}
	if NewProjector(Viewer{UserID: "aud", Audience: true}, s).CanPerceive(ref("obj")) {
		t.Error("the Audience perceived a hidden object with no grants")
	}
	// §14: the Director keeps it on their working stage.
	if !NewProjector(Viewer{UserID: "dir", Backstage: true}, s).CanPerceive(ref("obj")) {
		t.Error("the Director lost a hidden object from their working stage")
	}
}

func TestHiddenIsNotDeleted(t *testing.T) {
	// §7/§54: hidden must be a state the object HAS, not an absence. The
	// observable form of that claim: the very same ref becomes perceivable
	// again on reveal, with no re-creation step, and the Director could see
	// it the whole time.
	hidden := states(hiddenWith("obj"))
	player := Viewer{UserID: "player"}

	if NewProjector(player, hidden).CanPerceive(ref("obj")) {
		t.Fatal("hidden object was perceivable")
	}
	if !NewProjector(Viewer{UserID: "d", Backstage: true}, hidden).CanPerceive(ref("obj")) {
		t.Fatal("hidden object was gone even for the Director -- that is deletion, not hiding")
	}

	revealed := states(State{Ref: ref("obj"), Visibility: VisibilityVisible})
	if !NewProjector(player, revealed).CanPerceive(ref("obj")) {
		t.Fatal("revealing did not restore the same object reference")
	}
}

func TestCohortScopedVisibilityDoesNotLeakAcrossCohorts(t *testing.T) {
	// §16's boat example. One object, one Scene, two Cohorts, no Scene fork.
	s := states(hiddenWith("boat", Scope{Kind: ScopeCohort, ID: "cohort-a"}))

	if !NewProjector(Viewer{UserID: "a", CohortID: "cohort-a"}, s).CanPerceive(ref("boat")) {
		t.Error("Cohort A could not see its own object")
	}
	if NewProjector(Viewer{UserID: "b", CohortID: "cohort-b"}, s).CanPerceive(ref("boat")) {
		t.Error("Cohort B saw a Cohort-A-scoped object")
	}
	// An Ungrouped participant has no cohort and must not match. This is the
	// empty-string trap: without the guard in matchesAnyScope, "" == "" would
	// have revealed every cohort-scoped object to everyone outside a cohort.
	if NewProjector(Viewer{UserID: "u"}, s).CanPerceive(ref("boat")) {
		t.Error("an Ungrouped viewer matched a cohort grant")
	}
	// And a grant with an empty id must not act as a wildcard either.
	broken := states(hiddenWith("boat", Scope{Kind: ScopeCohort}))
	if NewProjector(Viewer{UserID: "a", CohortID: "cohort-a"}, broken).CanPerceive(ref("boat")) {
		t.Error("a cohort grant with no id behaved as a wildcard")
	}
}

func TestCharacterScopedVisibilityTracksSelectedCharacter(t *testing.T) {
	// §17. Matched against the SELECTED Character, so switching Character
	// changes what the same user perceives.
	s := states(hiddenWith("letter", Scope{Kind: ScopeCharacter, ID: "char-kessa"}))

	if !NewProjector(Viewer{UserID: "u", SelectedCharacterID: "char-kessa"}, s).CanPerceive(ref("letter")) {
		t.Error("the targeted Character could not see the object")
	}
	if NewProjector(Viewer{UserID: "u", SelectedCharacterID: "char-other"}, s).CanPerceive(ref("letter")) {
		t.Error("the same user saw a Character-scoped object while playing a different Character")
	}
	if NewProjector(Viewer{UserID: "u"}, s).CanPerceive(ref("letter")) {
		t.Error("a viewer with no selected Character matched a character grant")
	}
}

func TestAudienceScopeIsIndependentOfCast(t *testing.T) {
	// §18 explicitly forbids assuming "Audience sees whatever Cast sees".
	// Both directions have to work, which is precisely why ScopeCast exists
	// as well as a base visible state.
	audienceOnly := states(hiddenWith("chandelier", Scope{Kind: ScopeAudience}))
	if !NewProjector(Viewer{UserID: "a", Audience: true}, audienceOnly).CanPerceive(ref("chandelier")) {
		t.Error("the Audience could not see an audience-scoped object")
	}
	if NewProjector(Viewer{UserID: "p"}, audienceOnly).CanPerceive(ref("chandelier")) {
		t.Error("a Player saw an audience-only object")
	}

	castOnly := states(hiddenWith("map-pin", Scope{Kind: ScopeCast}))
	if !NewProjector(Viewer{UserID: "p"}, castOnly).CanPerceive(ref("map-pin")) {
		t.Error("Cast could not see a cast-scoped object")
	}
	if NewProjector(Viewer{UserID: "a", Audience: true}, castOnly).CanPerceive(ref("map-pin")) {
		t.Error("the Audience saw a cast-only object -- the house should not see the players' pin")
	}
	// An anonymous onlooker is not Cast.
	if NewProjector(Viewer{}, castOnly).CanPerceive(ref("map-pin")) {
		t.Error("an unauthenticated viewer matched the cast grant")
	}
}

func TestMultipleGrantsAreAdditive(t *testing.T) {
	// Grant's chosen scope model: a hidden object may be revealed to several
	// audiences at once, which a single scope_kind column could not express.
	s := states(hiddenWith("relic",
		Scope{Kind: ScopeCohort, ID: "cohort-a"},
		Scope{Kind: ScopeCharacter, ID: "char-solo"},
	))
	if !NewProjector(Viewer{UserID: "a", CohortID: "cohort-a"}, s).CanPerceive(ref("relic")) {
		t.Error("the granted cohort could not see it")
	}
	if !NewProjector(Viewer{UserID: "s", SelectedCharacterID: "char-solo"}, s).CanPerceive(ref("relic")) {
		t.Error("the granted Character could not see it")
	}
	if NewProjector(Viewer{UserID: "z", CohortID: "cohort-z"}, s).CanPerceive(ref("relic")) {
		t.Error("an ungranted viewer saw it")
	}
}

func TestGrantsDoNotRestrictAVisibleObject(t *testing.T) {
	// Grants are exceptions to hidden, never restrictions on visible. If this
	// inverted, "visible" would quietly come to mean "visible to some people"
	// and §10's cheap default would stop being a default at all.
	s := states(State{
		Ref:        ref("lamp"),
		Visibility: VisibilityVisible,
		Scopes:     []Scope{{Kind: ScopeCohort, ID: "cohort-a"}},
	})
	if !NewProjector(Viewer{UserID: "b", CohortID: "cohort-b"}, s).CanPerceive(ref("lamp")) {
		t.Error("a cohort grant restricted an object whose state is visible")
	}
}

func TestVisibleAndDisabledAreSeparateDimensions(t *testing.T) {
	// §8: disabled is not hidden and hidden is not disabled. The two failure
	// modes this guards are both in §53's PARTIAL list.
	enabled := false
	interactionRef := Ref{Kind: KindParticipantInteraction, ID: "int-1"}

	// A visible object with a disabled interaction stays visible.
	visibleDisabled := map[Ref]State{
		ref("stall"):                {Ref: ref("stall"), Visibility: VisibilityVisible},
		interactionRef.normalized(): {Ref: interactionRef, Visibility: VisibilityVisible, InteractionEnabled: &enabled},
	}
	p := NewProjector(Viewer{UserID: "p"}, visibleDisabled)
	if !p.CanPerceive(ref("stall")) {
		t.Error("disabling an interaction hid the object")
	}
	if p.InteractionEnabled(interactionRef) {
		t.Error("a disabled interaction reported as enabled")
	}

	// A hidden object whose interaction is enabled is still not projected.
	on := true
	hiddenEnabled := map[Ref]State{
		ref("stall"):                hiddenWith("stall"),
		interactionRef.normalized(): {Ref: interactionRef, InteractionEnabled: &on},
	}
	p2 := NewProjector(Viewer{UserID: "p"}, hiddenEnabled)
	if p2.CanPerceive(ref("stall")) {
		t.Error("an enabled interaction made a hidden object perceivable")
	}
	if !p2.InteractionEnabled(interactionRef) {
		t.Error("interaction state was mutated by the object's visibility")
	}
}

func TestInteractionDefaultsToEnabled(t *testing.T) {
	// §10 applied to the interaction dimension: absent state means enabled,
	// so binding an interaction requires no state write to be usable.
	r := Ref{Kind: KindParticipantInteraction, ID: "int-x"}
	if !NewProjector(Viewer{UserID: "p"}, map[Ref]State{}).InteractionEnabled(r) {
		t.Error("an interaction with no state row defaulted to disabled")
	}
	// A visibility-only row (no interaction dimension) must not read as
	// disabled either -- a nil InteractionEnabled means "not applicable".
	s := states(hiddenWith("obj"))
	if !NewProjector(Viewer{UserID: "p"}, s).InteractionEnabled(ref("obj")) {
		t.Error("a nil interaction_enabled read as disabled")
	}
}

func TestScopeMetadataIsBackstageOnly(t *testing.T) {
	// §35: "Cohort A cannot infer Cohort B-only object metadata". Withholding
	// the OBJECT is not enough -- the grant list itself names cohorts and
	// Characters, so it must never reach a non-backstage viewer even for an
	// object they can perceive.
	s := states(hiddenWith("obj", Scope{Kind: ScopeCohort, ID: "cohort-a"}))

	if got := NewProjector(Viewer{UserID: "a", CohortID: "cohort-a"}, s).ScopesFor(ref("obj")); got != nil {
		t.Errorf("scope metadata leaked to a Player who can perceive the object: %v", got)
	}
	if got := NewProjector(Viewer{UserID: "d", Backstage: true}, s).ScopesFor(ref("obj")); len(got) != 1 {
		t.Errorf("the Director could not read scope metadata, got %v", got)
	}
}

func TestHiddenForReportsOrdinaryVisibilityNotViewerOutcome(t *testing.T) {
	// §14's Director marker. HiddenFor must describe the OBJECT ("this is
	// backstage-only"), not the viewer's own outcome -- otherwise the
	// Director's stage could not distinguish a hidden object it can see from
	// an ordinary one.
	s := states(hiddenWith("obj", Scope{Kind: ScopeCohort, ID: "cohort-a"}))
	if !NewProjector(Viewer{UserID: "d", Backstage: true}, s).HiddenFor(ref("obj")) {
		t.Error("the Director was not told the object is hidden from ordinary viewers")
	}
	// Granted Cohort A perceives it, and it is still genuinely hidden.
	if !NewProjector(Viewer{UserID: "a", CohortID: "cohort-a"}, s).HiddenFor(ref("obj")) {
		t.Error("HiddenFor described the viewer's outcome instead of the object's state")
	}
	if NewProjector(Viewer{UserID: "d", Backstage: true}, map[Ref]State{}).HiddenFor(ref("obj")) {
		t.Error("an object with no state row reported as hidden")
	}
}

func TestRefKindsAreClosedAndMapRefused(t *testing.T) {
	// §3/§46: the map must not enter the generic object system, and an
	// arbitrary kind must not be invented by a client. resolveSceneStageElement
	// refuses map_backdrop by stored kind (covered in the dbtests); this
	// covers the reference-level closed set.
	for _, kind := range []string{KindVenueLayoutElement, KindSceneStageElement, KindDrawingObject, KindParticipantInteraction} {
		if !IsSupportedKind(kind) {
			t.Errorf("%q should be a supported kind", kind)
		}
	}
	for _, kind := range []string{"map_backdrop", "grid_config", "stage_effect", "pinned_die", "announcement", "", "  "} {
		if IsSupportedKind(kind) {
			t.Errorf("%q must not be a supported canonical object kind", kind)
		}
	}
}

func TestOnlyInteractionsCarryInteractionDimension(t *testing.T) {
	// §12: the Director menu must not offer Interaction controls for objects
	// that cannot be acted upon, and this predicate is what it asks.
	if (Ref{Kind: KindSceneStageElement, ID: "x"}).SupportsInteraction() {
		t.Error("a Scene element claimed an interaction dimension")
	}
	if !(Ref{Kind: KindParticipantInteraction, ID: "x"}).SupportsInteraction() {
		t.Error("a participant interaction lacked an interaction dimension")
	}
	// An interaction's visibility follows its bound element, so it must not
	// be independently settable -- two switches for the same pixels is the
	// competing-truth failure §20 forbids.
	if (Ref{Kind: KindParticipantInteraction, ID: "x"}).SupportsVisibility() {
		t.Error("a participant interaction offered its own visibility switch")
	}
	if !(Ref{Kind: KindDrawingObject, ID: "x"}).SupportsVisibility() {
		t.Error("a drawing object could not be hidden")
	}
}

func TestScopeValidationRejectsMalformedGrants(t *testing.T) {
	valid := []Scope{
		{Kind: ScopeCast}, {Kind: ScopeAudience},
		{Kind: ScopeCohort, ID: "c"}, {Kind: ScopeCharacter, ID: "ch"},
	}
	for _, s := range valid {
		if err := s.validate(); err != nil {
			t.Errorf("scope %+v should be valid, got %v", s, err)
		}
	}
	invalid := []Scope{
		{Kind: ScopeCohort},            // tier-less kind with no id
		{Kind: ScopeCharacter},         // same
		{Kind: ScopeCast, ID: "c"},     // id on a tier scope
		{Kind: ScopeAudience, ID: "c"}, // same
		{Kind: "director"},             // hidden-with-no-grants is that state
		{Kind: "everyone"},             // not a theatrical scope
		{Kind: ""},                     // no kind
	}
	for _, s := range invalid {
		if err := s.validate(); err == nil {
			t.Errorf("scope %+v should have been rejected", s)
		}
	}
}

func TestNilProjectorFailsOpenOnlyForAbsentState(t *testing.T) {
	// A nil projector is what a caller with no Show has. It must behave like
	// "no state exists" (everything visible and enabled) rather than panicking
	// -- a venue with no Show has no canonical state and must still render.
	var p *Projector
	if !p.CanPerceive(ref("obj")) {
		t.Error("nil projector hid an object")
	}
	if p.HiddenFor(ref("obj")) {
		t.Error("nil projector reported an object as hidden")
	}
	if !p.InteractionEnabled(ref("obj")) {
		t.Error("nil projector disabled an interaction")
	}
	if p.ScopesFor(ref("obj")) != nil {
		t.Error("nil projector returned scope metadata")
	}
}

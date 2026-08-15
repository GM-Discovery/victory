// Package storysofar builds and stores a Character's Story So Far: the
// durable, chronological, private-by-default record of what actually
// happened in play (kernel-75 S5, S6).
//
// Two boundaries define this package.
//
// First, the prose is DETERMINISTIC (S1.6). There is no AI author inside
// Victory. Every sentence comes from a fixed template applied to recorded
// data, and a template whose inputs are missing omits its clause rather than
// guessing (S5.2). This package must never call an external service and
// must never assert a motivation, feeling, or trait the data does not
// literally record.
//
// Second, this package is a LEAF. It imports nothing else from this repo --
// not characters, not merchant, not tutorial. That is deliberate and
// load-bearing: characters.buildWorkbookPages needs to render StoryEvent to
// show the Story So Far page, so characters imports storysofar. If
// storysofar imported characters back (for the archetype catalog, the
// obvious temptation) the two would form an import cycle. Instead the
// canonical rules arrive as Rules, filled in by a caller that may import
// characters. There is exactly one archetype-to-attribute mapping in this
// codebase and it stays in characters.Chapter3Archetypes.
package storysofar

import "time"

// Visibility states. S6.3: private by default, with one future-compatible
// widening. There is deliberately no "public" -- see migration 070.
const (
	VisibilityPrivate = "private"
	VisibilityTable   = "table"
)

// Event types, mirroring character_story_events_type_check in migration 070.
const (
	EventArrival           = "arrival"
	EventMerchantMet       = "merchant_met"
	EventMerchantStance    = "merchant_stance"
	EventMerchantHaggle    = "merchant_haggle"
	EventEquipmentAcquired = "equipment_acquired"
	EventDoorIntention     = "door_intention"
	EventDialogueLearned   = "dialogue_learned"
	EventGateOpened        = "gate_opened"
	EventFaceSheetEcho     = "face_sheet_echo"
	EventDirectorMoment    = "director_moment"
	EventAftercare         = "aftercare"
	EventTutorialCompleted = "tutorial_completed"
	// Kernel 88: Director-awarded Fate and resolved Help/interrupt outcomes.
	// Deliberately narrow -- only Director-award-reason ledger entries and
	// resolved (not every opened) interrupts qualify, matching kernel-88
	// spec §19's "do not turn every button click into Story So Far noise."
	EventFatePointsAwarded = "fate_points_awarded"
	EventHelpResolved      = "help_resolved"
)

// Source kinds, mirroring character_story_events_source_kind_check.
const (
	SourceNone               = ""
	SourceTutorialMilestone  = "tutorial_milestone"
	SourceInventoryItem      = "inventory_item"
	SourceFreeformSubmission = "freeform_submission"
	SourceInteractionAttempt = "interaction_attempt"
	SourceDialogueTopic      = "dialogue_topic"
	SourceWorkbookEntry      = "workbook_entry"
	SourceCharacterJournal   = "character_journal"
	SourceAftercare          = "aftercare"
	SourceSocioFateLedger    = "socio_fate_ledger"
	SourceSocioPendingAction = "socio_pending_action"
)

// GeneratorVersion is the revision of the template library below. Bump it
// when the prose changes so old entries are not retroactively attributed to
// templates that did not write them.
const GeneratorVersion = 1

// StoryEvent is a stored row.
type StoryEvent struct {
	ID               string    `json:"id"`
	CharacterCardID  string    `json:"character_card_id"`
	OwnerUserID      string    `json:"owner_user_id"`
	EventType        string    `json:"event_type"`
	Title            string    `json:"title"`
	Summary          string    `json:"summary"`
	ShowRunID        string    `json:"show_run_id,omitempty"`
	ShowID           string    `json:"show_id,omitempty"`
	SessionID        string    `json:"session_id,omitempty"`
	PlacementID      string    `json:"show_scene_placement_id,omitempty"`
	SourceKind       string    `json:"source_kind,omitempty"`
	SourceRef        string    `json:"source_ref,omitempty"`
	VisibilityState  string    `json:"visibility_state"`
	AuthoredByUserID string    `json:"authored_by_user_id,omitempty"`
	AuthoredByName   string    `json:"authored_by_name,omitempty"`
	GeneratorVersion int       `json:"generator_version"`
	OccurredAt       time.Time `json:"occurred_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// Generated reports whether this entry was written by the template library
// rather than by a person. S5.5: the Player may not edit generated text.
func (e StoryEvent) Generated() bool { return e.AuthoredByUserID == "" }

// DraftEvent is one clause the template library produced, before storage.
// The (EventType, SourceKind, SourceRef) triple is the dedupe key -- see
// uq_character_story_events_dedupe.
type DraftEvent struct {
	EventType  string
	Title      string
	Summary    string
	SourceKind string
	SourceRef  string
	OccurredAt time.Time
}

// ArchetypeRule is the slice of characters.Chapter3Archetype this package
// needs. Injected rather than imported -- see the package doc.
type ArchetypeRule struct {
	Key                string
	Title              string
	PrimaryAttribute   string
	SecondaryAttribute string
	KeySkill           string
}

// Rules carries the canonical Socio definitions this package must not own a
// second copy of. AttributeOrder must be the full canonical attribute list
// in its canonical order: SelectFaceSheetLine uses that order as a
// tie-breaker, so a caller passing a map-derived slice would reintroduce
// exactly the nondeterminism the ordering exists to remove.
type Rules struct {
	AttributeOrder []string
	Archetype      func(key string) (ArchetypeRule, bool)
}

// StageLine is one life-stage Face Sheet history line
// (character_workbook_entries, page_key='history', entry_type='chapter2_stage').
type StageLine struct {
	ID          string
	Title       string
	Body        string
	StageNumber int
	SortOrder   int
}

// Attempt is one durable stance or Haggle record
// (character_interaction_attempts, migration 071).
type Attempt struct {
	Kind         string
	PacketSlug   string
	StanceKey    string
	Disposition  string
	ResponseTier int
	SkillKey     string
	HasSkill     bool
	Die          string
	Total        int
	TargetValue  int
	Success      bool
	CreatedAt    time.Time
}

// InventoryLine is one acquired item. Declared here rather than imported
// from merchant, which would create the cycle described in the package doc.
type InventoryLine struct {
	InventoryItemID string
	ItemName        string
	Quantity        int
	AcquiredAt      time.Time
}

// DialogueTopicSeen is one Ra topic the Player actually read.
type DialogueTopicSeen struct {
	TopicID  string
	TopicKey string
	Label    string
	Required bool
	ViewedAt time.Time
}

// Inputs is everything the template library may draw on. The field set is
// exactly S5.1's permitted source list -- nothing else is available to the
// templates, which is how "never invent unsupported traits" is enforced
// structurally rather than by reviewer discipline.
type Inputs struct {
	CharacterCardID string
	OwnerUserID     string
	CharacterName   string
	Pronouns        string

	ArchetypeKey       string
	ArchetypeTitle     string
	PrimaryAttribute   string
	SecondaryAttribute string
	KeySkill           string
	ArchetypeConfirmed bool

	Attributes map[string]int
	StageLines []StageLine

	Milestones map[string]bool

	Inventory     []InventoryLine
	DoorIntention string
	Attempts      []Attempt
	Topics        []DialogueTopicSeen

	// FateAwards and HelpResolutions are Kernel 88's Socio play-surface
	// sources. Already pre-filtered by the caller to the "meaningful"
	// subset (Director-award-reason ledger rows; resolved, not merely
	// opened, interrupts) -- this package only renders, it does not decide
	// what counts as meaningful (kernel-88 spec §19).
	FateAwards      []FateAwardLine
	HelpResolutions []HelpResolutionLine

	ShowRunID   string
	ShowID      string
	SessionID   string
	PlacementID string

	// CompletedAt anchors every generated entry's occurred_at. Passed in
	// rather than read from the clock so Generate stays pure and testable.
	CompletedAt time.Time
}

// FateAwardLine is one Director-awarded (not player-spent) Fate ledger
// entry (character_socio_fate_ledger, backend/internal/socio).
type FateAwardLine struct {
	LedgerID  string
	Delta     int
	Reason    string
	CreatedAt time.Time
}

// HelpResolutionLine is one resolved Help/interrupt outcome
// (socio_pending_actions, backend/internal/socio).
type HelpResolutionLine struct {
	PendingActionID  string
	HelperName       string
	PrimaryActorName string
	Overage          int
	Succeeded        bool
	ResolvedAt       time.Time
}

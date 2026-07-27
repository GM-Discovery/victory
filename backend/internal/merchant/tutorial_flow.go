package merchant

// Kernel 74's Locked Courtyard tutorial flow: Kessa completion, the freeform
// door intention, and Ra's guided dialogue.
//
// Why this lives in package merchant rather than its own package: every one
// of these actions must resolve the exact same participation context --
// authenticated, active Player roster member, Show's current Scene matches,
// Character selected/active/owned, active session, venue capability on --
// and that gate is ResolveEligibleContext, which lives here. A sibling
// package would have to either import merchant (fine) or duplicate the gate
// (not fine). Importing was the choice everywhere it was possible;
// dialogue, projection, and tutorial are all leaf packages this file calls
// into. What could not be moved without a large refactor is
// ResolveEligibleContext itself, since participant_interactions CRUD,
// authoring, and preview all sit beside it.
//
// The package name is now narrower than its contents. That is a naming debt
// worth one comment rather than a rename touching every call site.

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dialogue"
	"victory/backend/internal/messages"
	"victory/backend/internal/projection"
	"victory/backend/internal/tutorial"
)

const (
	InteractionTypeFreeformSubmission = "freeform_submission"
	InteractionTypeGuidedDialogue     = "guided_dialogue"
)

// defaultFreeformMaxLength bounds the door intention. Generous enough for
// the "original full-sentence intention" the manual golden path asks for,
// small enough that the Directors+ note stays readable in a chat tray.
const defaultFreeformMaxLength = 1000

// participationFor projects an already-authorized EligibleContext into the
// identity shape the leaf packages take. Never constructed from a client
// payload.
func participationFor(eligible EligibleContext) tutorial.Participation {
	return tutorial.Participation{
		UserID:          eligible.ActorUserID,
		CharacterCardID: eligible.CharacterCardID,
		ShowID:          eligible.ShowID,
		ShowRunID:       eligible.ShowRunID,
		PlacementID:     eligible.PlacementID,
	}
}

// requireBindingMilestone enforces a gated interaction's milestone on
// INVOCATION, not merely on discovery.
//
// The snapshot gate in world/snapshot.go removes a gated element from a
// Player's payload, which is what makes the door undiscoverable. That alone
// is not sufficient: a Player who learns the interaction id another way (a
// second Character who already unlocked it, a shared screen, a replayed
// request) could otherwise POST straight to it. S13 requires both -- "client-
// forged progress does not reveal the door" AND "Player cannot unlock the
// door before Kessa completion" -- so the same milestone is checked again
// here, against the same rows, before any gated action proceeds.
//
// An ungated interaction (no binding, or a binding with no milestone) is
// unaffected, which is every Kernel 73/73A interaction.
func requireBindingMilestone(ctx context.Context, pool *pgxpool.Pool, eligible EligibleContext) error {
	var milestone string
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(requires_milestone, '')
		FROM stage_element_bindings
		WHERE participant_interaction_id = $1 AND COALESCE(requires_milestone, '') <> ''
		LIMIT 1
	`, eligible.Interaction.ID).Scan(&milestone)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	has, err := tutorial.HasMilestone(ctx, pool, participationFor(eligible), milestone)
	if err != nil {
		return err
	}
	if !has {
		return errors.New("milestone_required")
	}
	return nil
}

// --- Kessa completion ------------------------------------------------------

// CompleteKessaIntro records that this Player is done with Kessa (S6.1).
//
// Completion requires no purchase, no Haggle, and no successful stance --
// the Player may simply leave. It is scoped to this Player, Character,
// Show, and interaction, and persists through refresh/reconnect because it
// is a row, not panel state. The Player may return to Kessa afterward; this
// records a milestone, it does not close the shop.
func CompleteKessaIntro(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID string) ([]string, error) {
	eligible, err := ResolveEligibleContext(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return nil, err
	}
	if eligible.Interaction.InteractionType != InteractionTypeOpenEquipMode {
		return nil, errors.New("interaction_type_mismatch")
	}
	p := participationFor(eligible)
	payload, _ := json.Marshal(map[string]any{"interaction_id": interactionID})
	if err := tutorial.RecordMilestone(ctx, pool, p, tutorial.MilestoneKessaIntroCompleted, payload); err != nil {
		return nil, err
	}
	return tutorial.LoadProgress(ctx, pool, p)
}

// --- Freeform door intention ------------------------------------------------

// FreeformConfig is the bounded configuration a freeform_submission
// interaction carries. Exactly these keys -- S7.1 excludes a general
// form-builder, so there is no field list, no validation DSL, and no
// conditional logic.
type FreeformConfig struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	Prompt          string `json:"prompt"`
	SubmitLabel     string `json:"submit_label"`
	MaxLength       int    `json:"max_length"`
	NextInteraction string `json:"next_interaction_slug"`
	ProgressKey     string `json:"progress_key"`
	NoteSubject     string `json:"note_subject"`
	NoteSuffix      string `json:"note_suffix"`
}

func freeformConfigFrom(config map[string]any) FreeformConfig {
	raw, _ := json.Marshal(config)
	var out FreeformConfig
	_ = json.Unmarshal(raw, &out)
	if out.MaxLength <= 0 || out.MaxLength > defaultFreeformMaxLength {
		out.MaxLength = defaultFreeformMaxLength
	}
	if strings.TrimSpace(out.SubmitLabel) == "" {
		out.SubmitLabel = "Make the Attempt"
	}
	return out
}

// FreeformSubmissionResult is what the Player sees immediately after
// committing their intention.
//
// SubmittedText is echoed back as the exact stored plain text so the client
// can render it with textContent. Victory never wraps it in authored
// narration server-side (S1.4) -- Narration and Interruption are separate
// authored strings the client places around the quoted text, precisely so a
// Player who typed a complete sentence is not mangled into "You start to I
// kick the door in."
type FreeformSubmissionResult struct {
	SubmissionID  string `json:"submission_id"`
	SubmittedText string `json:"submitted_text"`
	Narration     string `json:"narration"`
	Interruption  string `json:"interruption"`
	// NextInteractionID is the guided-dialogue interaction Ra's Program
	// should open immediately (S7.3: "Then automatically begin Ra"). Empty
	// if the packet does not chain one.
	NextInteractionID string `json:"next_interaction_id,omitempty"`
	// AlreadySubmitted is true when this was a retry that resolved to the
	// original row rather than creating a second intention.
	AlreadySubmitted bool `json:"already_submitted"`
}

const (
	freeformNarrationLead = "You make your move:"
	freeformInterruption  = "Before you can carry it through, a voice cuts across the courtyard."
)

// resolveNextInteractionID finds the sibling interaction on the same
// placement whose internal_name matches the configured next slug. Resolved
// server-side by name rather than accepting an ID from the client, so a
// Player cannot chain the door into an arbitrary interaction.
func resolveNextInteractionID(ctx context.Context, pool *pgxpool.Pool, placementID, slug string) (string, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "", nil
	}
	var id string
	err := pool.QueryRow(ctx, `
		SELECT id::text FROM participant_interactions
		WHERE show_scene_placement_id = $1 AND internal_name = $2 AND enabled
		LIMIT 1
	`, placementID, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// SubmitFreeform records the Player's door intention and creates the
// Directors+ note.
//
// Deliberately absent: any adjudication. There is no roll, no stance, no
// skill check, and no Director resolution (S1.3, S7.3). The submission's
// only mechanical effect is to begin Ra's interruption.
func SubmitFreeform(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID, text, idempotencyKey string) (FreeformSubmissionResult, []string, error) {
	eligible, err := ResolveEligibleContext(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return FreeformSubmissionResult{}, nil, err
	}
	if eligible.Interaction.InteractionType != InteractionTypeFreeformSubmission {
		return FreeformSubmissionResult{}, nil, errors.New("interaction_type_mismatch")
	}
	if err := requireBindingMilestone(ctx, pool, eligible); err != nil {
		return FreeformSubmissionResult{}, nil, err
	}
	cfg := freeformConfigFrom(eligible.Interaction.ConfigurationJSON)

	// Trim first, then reject: a whitespace-only submission is empty
	// (S7.2), and the stored text should not carry incidental padding.
	text = strings.TrimSpace(text)
	if text == "" {
		return FreeformSubmissionResult{}, nil, errors.New("submission_text_required")
	}
	if utf8.RuneCountInString(text) > cfg.MaxLength {
		return FreeformSubmissionResult{}, nil, errors.New("submission_text_too_long")
	}

	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return FreeformSubmissionResult{}, nil, errors.New("idempotency_key_required")
	}

	// Two-layer idempotency, matching the two constraints in migration 066:
	// the (interaction, user, character, key) UNIQUE catches an exact retry,
	// and the key-independent partial UNIQUE catches a client that generated
	// a fresh key. Either way, DO NOTHING then read back -- so the Directors+
	// note below is only created on a genuinely new row.
	var submissionID string
	err = pool.QueryRow(ctx, `
		INSERT INTO participant_freeform_submissions (
			participant_interaction_id, user_id, character_card_id, show_id,
			session_id, show_scene_placement_id, submitted_text, idempotency_key
		)
		VALUES ($1, $2, $3, $4, $5::uuid, $6::uuid, $7, $8)
		ON CONFLICT DO NOTHING
		RETURNING id::text
	`, interactionID, actorUserID, eligible.CharacterCardID, eligible.ShowID,
		nullableUUID(eligible.SessionID), nullableUUID(eligible.PlacementID), text, idempotencyKey).Scan(&submissionID)

	alreadySubmitted := false
	if errors.Is(err, pgx.ErrNoRows) {
		alreadySubmitted = true
		var storedText string
		if err := pool.QueryRow(ctx, `
			SELECT id::text, submitted_text FROM participant_freeform_submissions
			WHERE participant_interaction_id = $1 AND user_id = $2 AND character_card_id = $3
			LIMIT 1
		`, interactionID, actorUserID, eligible.CharacterCardID).Scan(&submissionID, &storedText); err != nil {
			return FreeformSubmissionResult{}, nil, err
		}
		// Return the ORIGINAL words, not the retry's. The intention cannot be
		// edited after Ra's interruption begins (S7.2).
		text = storedText
	} else if err != nil {
		return FreeformSubmissionResult{}, nil, err
	}

	p := participationFor(eligible)
	payload, _ := json.Marshal(map[string]any{"submission_id": submissionID})
	if err := tutorial.RecordMilestone(ctx, pool, p, tutorial.MilestoneDoorIntentionSubmitted, payload); err != nil {
		return FreeformSubmissionResult{}, nil, err
	}

	var noteRecipients []string
	if !alreadySubmitted {
		noteRecipients, err = createDoorIntentionNote(ctx, pool, eligible, cfg, text)
		if err != nil {
			return FreeformSubmissionResult{}, nil, err
		}
	}

	nextID, err := resolveNextInteractionID(ctx, pool, eligible.PlacementID, cfg.NextInteraction)
	if err != nil {
		return FreeformSubmissionResult{}, nil, err
	}

	return FreeformSubmissionResult{
		SubmissionID:      submissionID,
		SubmittedText:     text,
		Narration:         freeformNarrationLead,
		Interruption:      freeformInterruption,
		NextInteractionID: nextID,
		AlreadySubmitted:  alreadySubmitted,
	}, noteRecipients, nil
}

// createDoorIntentionNote writes the Directors+ backstage note (S8.1).
//
// Informational only. No GO control, no cue, no readiness signal, nothing
// the Show waits on (S8.4) -- the Player has already moved on to Ra by the
// time any Director reads this.
func createDoorIntentionNote(ctx context.Context, pool *pgxpool.Pool, eligible EligibleContext, cfg FreeformConfig, text string) ([]string, error) {
	characterName := "Unnamed Character"
	_ = pool.QueryRow(ctx, `
		SELECT COALESCE(NULLIF(name, ''), 'Unnamed Character') FROM character_cards WHERE id = $1
	`, eligible.CharacterCardID).Scan(&characterName)

	subject := strings.TrimSpace(cfg.NoteSubject)
	if subject == "" {
		subject = "Unresolved Door Intention"
	}
	suffix := strings.TrimSpace(cfg.NoteSuffix)
	if suffix == "" {
		suffix = "Interrupted by Ra before completion."
	}

	// The Player's words are embedded as plain text in a plain-text body.
	// Nothing downstream renders this as HTML -- the Catharsis backstage tab
	// uses textContent, and the mailbox already treats bodies as text.
	body := characterName + "\n\"" + text + "\"\n" + suffix

	venueSlug, err := resolvePlacementVenueSlug(ctx, pool, eligible.PlacementID)
	if err != nil {
		return nil, err
	}

	return messages.InsertBackstageNote(ctx, pool, messages.BackstageNoteInput{
		SessionID: eligible.SessionID,
		VenueSlug: venueSlug,
		Subject:   subject + " — " + characterName,
		Body:      body,
	})
}

// OpenFreeform returns the authored prompt configuration for a freeform
// interaction, plus any submission this participation has already made.
//
// Opening records NO milestone and creates nothing. The Player has not
// committed to anything by looking at the door -- only SubmitFreeform does,
// which is also what makes re-opening after submitting safe: it returns the
// original words rather than an empty field to overwrite.
type FreeformOpenState struct {
	Config    FreeformConfig            `json:"config"`
	Submitted *FreeformSubmissionResult `json:"submitted,omitempty"`
}

func OpenFreeform(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID string) (FreeformOpenState, error) {
	eligible, err := ResolveEligibleContext(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return FreeformOpenState{}, err
	}
	if eligible.Interaction.InteractionType != InteractionTypeFreeformSubmission {
		return FreeformOpenState{}, errors.New("interaction_type_mismatch")
	}
	if err := requireBindingMilestone(ctx, pool, eligible); err != nil {
		return FreeformOpenState{}, err
	}
	out := FreeformOpenState{Config: freeformConfigFrom(eligible.Interaction.ConfigurationJSON)}

	var submissionID, storedText string
	err = pool.QueryRow(ctx, `
		SELECT id::text, submitted_text FROM participant_freeform_submissions
		WHERE participant_interaction_id = $1 AND user_id = $2 AND character_card_id = $3
		LIMIT 1
	`, interactionID, actorUserID, eligible.CharacterCardID).Scan(&submissionID, &storedText)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return FreeformOpenState{}, err
	}
	nextID, err := resolveNextInteractionID(ctx, pool, eligible.PlacementID, out.Config.NextInteraction)
	if err != nil {
		return FreeformOpenState{}, err
	}
	out.Submitted = &FreeformSubmissionResult{
		SubmissionID:      submissionID,
		SubmittedText:     storedText,
		Narration:         freeformNarrationLead,
		Interruption:      freeformInterruption,
		NextInteractionID: nextID,
		AlreadySubmitted:  true,
	}
	return out, nil
}

// --- Ra's guided dialogue ---------------------------------------------------

func dialoguePacketSlug(config map[string]any) string {
	if v, ok := config["packet_slug"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// resolveDialoguePacket is the shared prelude of all three dialogue actions.
func resolveDialoguePacket(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID string) (EligibleContext, dialogue.Packet, []dialogue.Topic, error) {
	eligible, err := ResolveEligibleContext(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return EligibleContext{}, dialogue.Packet{}, nil, err
	}
	if eligible.Interaction.InteractionType != InteractionTypeGuidedDialogue {
		return EligibleContext{}, dialogue.Packet{}, nil, errors.New("interaction_type_mismatch")
	}
	if err := requireBindingMilestone(ctx, pool, eligible); err != nil {
		return EligibleContext{}, dialogue.Packet{}, nil, err
	}
	slug := dialoguePacketSlug(eligible.Interaction.ConfigurationJSON)
	if slug == "" {
		return EligibleContext{}, dialogue.Packet{}, nil, errors.New("interaction_missing_packet")
	}
	packet, err := dialogue.LoadPacketBySlug(ctx, pool, eligible.LocationID, slug)
	if err != nil {
		return EligibleContext{}, dialogue.Packet{}, nil, err
	}
	topics, err := dialogue.LoadTopics(ctx, pool, packet.ID)
	if err != nil {
		return EligibleContext{}, dialogue.Packet{}, nil, err
	}
	return eligible, packet, topics, nil
}

// OpenDialogue starts or resumes Ra's Program (S9.1, S9.5).
//
// Resuming is the same call: seen topics stay marked, unlocked topics stay
// unlocked, and Leave stays available once earned, because all of that is
// recomputed from rows rather than restored from client state.
func OpenDialogue(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID string) (dialogue.State, error) {
	eligible, packet, topics, err := resolveDialoguePacket(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return dialogue.State{}, err
	}
	p := participationFor(eligible)
	payload, _ := json.Marshal(map[string]any{"packet_slug": packet.Slug})
	if err := tutorial.RecordMilestone(ctx, pool, p, tutorial.MilestoneRaIntroStarted, payload); err != nil {
		return dialogue.State{}, err
	}
	return dialogue.BuildOpenState(ctx, pool, p, packet, topics)
}

// ReadTopic returns one authored response and marks it seen.
//
// The locked-topic refusal here is the real gate, not the disabled button in
// the UI (S13): a Player posting a topic_key whose prerequisites are unmet
// gets topic_locked, and the authored response never leaves the server.
func ReadTopic(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID, topicKey string) (dialogue.State, error) {
	eligible, packet, topics, err := resolveDialoguePacket(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return dialogue.State{}, err
	}
	p := participationFor(eligible)
	return dialogue.ReadTopic(ctx, pool, p, packet, topics, topicKey)
}

// LeaveDialogueResult is what Leave Ra returns: the lock-reveal narration
// plus the destination the Player has just transitioned to.
type LeaveDialogueResult struct {
	State      dialogue.State    `json:"state"`
	Projection projection.Active `json:"projection"`
}

// LeaveDialogue is the Player-controlled end of the tutorial (S11.1).
//
// It records ra_intro_completed and tutorial_handoff_entered, opens THIS
// Player's local projection, and changes nothing shared: no write to
// shows.current_show_scene_placement_id, no other Player moved, no Director
// approval required (S16.13-16.16).
func LeaveDialogue(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID string) (LeaveDialogueResult, error) {
	eligible, packet, topics, err := resolveDialoguePacket(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return LeaveDialogueResult{}, err
	}
	p := participationFor(eligible)

	// Server-authorized, not UI-gated: posting Leave directly without having
	// read the required topics is refused (S13).
	state, err := dialogue.LeaveState(ctx, pool, p, packet, topics)
	if err != nil {
		return LeaveDialogueResult{}, err
	}

	venueSlug, err := resolvePlacementVenueSlug(ctx, pool, eligible.PlacementID)
	if err != nil {
		return LeaveDialogueResult{}, err
	}
	enabled, err := projection.VenueLocalProjectionEnabled(ctx, pool, venueSlug)
	if err != nil {
		return LeaveDialogueResult{}, err
	}
	if !enabled {
		return LeaveDialogueResult{}, errors.New("unknown_target")
	}

	sceneID, err := projection.ResolveDestinationScene(ctx, pool, eligible.LocationID, packet.DestinationSceneSlug)
	if err != nil {
		return LeaveDialogueResult{}, err
	}
	active, err := projection.Open(ctx, pool, actorUserID, eligible.CharacterCardID, eligible.ShowID, sceneID, eligible.PlacementID)
	if err != nil {
		return LeaveDialogueResult{}, err
	}

	completedPayload, _ := json.Marshal(map[string]any{"packet_slug": packet.Slug})
	if err := tutorial.RecordMilestone(ctx, pool, p, tutorial.MilestoneRaIntroCompleted, completedPayload); err != nil {
		return LeaveDialogueResult{}, err
	}
	handoffPayload, _ := json.Marshal(map[string]any{"scene_slug": active.SceneSlug})
	if err := tutorial.RecordMilestone(ctx, pool, p, tutorial.MilestoneTutorialHandoffEntered, handoffPayload); err != nil {
		return LeaveDialogueResult{}, err
	}

	return LeaveDialogueResult{State: state, Projection: active}, nil
}

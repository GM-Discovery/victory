package merchant

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/characters"
	"victory/backend/internal/recognition"
	"victory/backend/internal/storysofar"
	"victory/backend/internal/tutorial"
)

// Kernel 75 S3.2/S4: the Continue verb and the completion Program payload.

// TutorialCompletion is everything the full-screen completion Program needs
// in one round trip. Every string in it is server-authored: the client must
// not become a second place where this copy is decided, because then the
// wording would live in two files that drift.
type TutorialCompletion struct {
	Headline    string   `json:"headline"`
	Body        []string `json:"body"`
	RecapLines  []string `json:"recap_lines"`
	WaitingCopy []string `json:"waiting_copy"`

	StoryEvents []storysofar.StoryEvent `json:"story_events"`
	Inventory   []InventoryEntry        `json:"inventory"`
	Recognition recognition.State       `json:"recognition"`
	Milestones  []string                `json:"milestones"`

	CharacterCardID string `json:"character_card_id"`
	CharacterName   string `json:"character_name"`
	ShowID          string `json:"show_id"`
}

// completionHeadline / completionBody / waitingCopy are S4.2 and S10.1's
// required wording. Reflective, not celebratory -- the kernel is explicit
// that this is not a confetti moment.
//
// Setting neutrality matters here (S1.5): the tutorial introduced Niava and
// the Crown Bet, but the next shared Scene belongs to whatever setting the
// Narrator chooses, and this copy must not presume otherwise.
const completionHeadline = "You completed Victory's automated Socio tutorial."

var completionBody = []string{
	"You created and equipped a Character, entered a shared social space, made meaningful choices, and carried those choices into a continuing history.",
	"Your Character, equipment, and progress are saved.",
	"The next part of the story is played with real people during a scheduled Session. Your Narrator may continue into Niava or any other setting the table chooses.",
}

// waitingCopy is S10.1 verbatim. It must never acquire a countdown, a
// participant count, or an "N of M ready" phrasing -- S1.11's rule that the
// system must not depend on a declared or inferred group size applies to
// this copy as much as to the backstage list.
var waitingCopy = []string{
	"The automated beginning is complete.",
	"Your Character and progress are saved.",
	"The story continues with real people during a scheduled Session.",
}

// storyRules injects the canonical Socio definitions into the storysofar
// leaf package, which owns no copy of them. characters.Chapter3Archetypes
// stays the single source of the archetype-to-attribute mapping.
func storyRules() storysofar.Rules {
	return storysofar.Rules{
		AttributeOrder: characters.AllChapter2Attributes,
		Archetype: func(key string) (storysofar.ArchetypeRule, bool) {
			a, ok := characters.Chapter3ArchetypeByKey(key)
			if !ok {
				return storysofar.ArchetypeRule{}, false
			}
			return storysofar.ArchetypeRule{
				Key:                a.Key,
				Title:              a.Title,
				PrimaryAttribute:   a.PrimaryAttribute,
				SecondaryAttribute: a.SecondaryAttribute,
				KeySkill:           a.KeySkill,
			}, true
		},
	}
}

func storyRefFor(eligible EligibleContext) storysofar.Ref {
	return storysofar.Ref{
		CharacterCardID: eligible.CharacterCardID,
		OwnerUserID:     eligible.ActorUserID,
		ShowRunID:       eligible.ShowRunID,
		ShowID:          eligible.ShowID,
		SessionID:       eligible.SessionID,
		PlacementID:     eligible.PlacementID,
	}
}

// CompleteTutorial is the Continue press (S3.2).
//
// It records the completion milestone, generates and stores Story So Far,
// grants one-time Player recognition, and returns the completion Program
// payload. It does NOT alter the shared current Scene and does not move
// anyone -- LeaveDialogue already opened this Player's local projection.
//
// RETRY-SAFE AND IDEMPOTENT, via three independent database-level
// mechanisms rather than an application-level guard:
//
//	1. participant_tutorial_progress UNIQUE(user, character, show, key)
//	2. uq_character_story_events_dedupe, which absorbs a retry only because
//	   storysofar.Generate is a pure function of stored inputs
//	3. player_recognition_grants UNIQUE(user_id, recognition_key)
//
// The three writes are deliberately NOT wrapped in one transaction. Each is
// independently idempotent, so a partial failure self-heals on the next
// press -- the same property LeaveDialogue's milestone sequence already has.
// A cross-package pgx.Tx would mean threading a tx through three leaf
// packages that all take *pgxpool.Pool today, for no correctness gain.
func CompleteTutorial(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID string) (TutorialCompletion, error) {
	eligible, err := ResolveEligibleContext(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return TutorialCompletion{}, err
	}
	p := participationFor(eligible)

	// The gate must actually be open. A client posting Continue without
	// having left Ra is refused here, not merely hidden in the UI -- the
	// same server-authorized posture as dialogue.LeaveState's
	// required_topics_unseen. writeError maps milestone_required to 403.
	entered, err := tutorial.HasMilestone(ctx, pool, p, tutorial.MilestoneTutorialHandoffEntered)
	if err != nil {
		return TutorialCompletion{}, err
	}
	if !entered {
		return TutorialCompletion{}, errors.New("milestone_required")
	}

	payload, _ := json.Marshal(map[string]any{"interaction_id": interactionID})
	if err := tutorial.RecordMilestone(ctx, pool, p, tutorial.MilestoneTutorialCompleted, payload); err != nil {
		return TutorialCompletion{}, err
	}

	if err := generateStoryForCompletion(ctx, pool, eligible); err != nil {
		return TutorialCompletion{}, err
	}

	granted, err := recognition.GrantFirstTutorialCompleted(ctx, pool,
		eligible.ActorUserID, eligible.CharacterCardID, eligible.ShowID)
	if err != nil {
		return TutorialCompletion{}, err
	}

	completion, err := LoadTutorialCompletion(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return TutorialCompletion{}, err
	}
	// NewlyGranted is knowable only HERE, at the moment of the insert.
	// LoadTutorialCompletion is a pure read and deliberately reports
	// NewlyGranted=false -- reopening the Program a week later must not
	// claim the Player just earned something. Carrying the grant result
	// across is what lets the Program show the recognition beat exactly
	// once, on the press that actually earned it (S7.3).
	completion.Recognition = granted
	return completion, nil
}

// generateStoryForCompletion loads the permitted source data, runs the
// deterministic template library, and stores the result.
//
// completedAt is read from the recorded milestone rather than from the clock
// so that a reopened Program and a retried Continue anchor every entry to
// the same instant. Reading the clock here would make Generate's output
// differ between the first press and a retry -- which would not duplicate
// rows (the dedupe key excludes timestamps) but would make two identical
// stories claim two different completion times.
func generateStoryForCompletion(ctx context.Context, pool *pgxpool.Pool, eligible EligibleContext) error {
	completedAt := time.Now().UTC()
	var recorded time.Time
	err := pool.QueryRow(ctx, `
		SELECT created_at FROM participant_tutorial_progress
		WHERE user_id = $1 AND character_card_id = $2 AND show_id = $3 AND milestone_key = $4
	`, eligible.ActorUserID, eligible.CharacterCardID, eligible.ShowID,
		tutorial.MilestoneTutorialCompleted).Scan(&recorded)
	if err == nil {
		completedAt = recorded
	}

	ref := storyRefFor(eligible)
	inputs, err := storysofar.LoadInputs(ctx, pool, ref, storyRules(), completedAt)
	if err != nil {
		return err
	}
	return storysofar.Persist(ctx, pool, ref, storysofar.Generate(inputs, storyRules()))
}

// LoadTutorialCompletion is the reopen path (S4.3).
//
// A pure read: it records no milestone, generates no story, and grants no
// recognition. Reopening the completion Program a week later must show the
// same ending, not re-award anything. The Recognition it reports is
// therefore always NewlyGranted=false, because the grant happened at
// Continue.
func LoadTutorialCompletion(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID string) (TutorialCompletion, error) {
	eligible, err := ResolveEligibleContext(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return TutorialCompletion{}, err
	}
	p := participationFor(eligible)

	milestones, err := tutorial.LoadProgress(ctx, pool, p)
	if err != nil {
		return TutorialCompletion{}, err
	}

	events, err := storysofar.ListForCharacter(ctx, pool, eligible.CharacterCardID, true)
	if err != nil {
		return TutorialCompletion{}, err
	}

	// Passing ActorUserID re-runs ListInventoryForCharacter's own
	// ownership check. Redundant with ResolveEligibleContext above, and kept
	// that way on purpose: the inventory read owns its own gate, and
	// bypassing it here would create a second definition of who may see a
	// Character's equipment.
	inventory, err := ListInventoryForCharacter(ctx, pool, eligible.ActorUserID, eligible.CharacterCardID)
	if err != nil {
		return TutorialCompletion{}, err
	}

	grants, err := recognition.LoadForUser(ctx, pool, eligible.ActorUserID)
	if err != nil {
		return TutorialCompletion{}, err
	}
	recogState := recognition.State{}
	for i := range grants {
		if grants[i].Key == recognition.KeyFirstTutorialCompleted {
			recogState.Grant = &grants[i]
			break
		}
	}

	var characterName string
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(NULLIF(name, ''), 'Your Character')
		FROM character_cards WHERE id = $1
	`, eligible.CharacterCardID).Scan(&characterName); err != nil {
		return TutorialCompletion{}, err
	}

	recap := make([]string, 0, len(events))
	for _, e := range events {
		recap = append(recap, e.Summary)
	}

	return TutorialCompletion{
		Headline:        completionHeadline,
		Body:            completionBody,
		RecapLines:      recap,
		WaitingCopy:     waitingCopy,
		StoryEvents:     events,
		Inventory:       inventory,
		Recognition:     recogState,
		Milestones:      milestones,
		CharacterCardID: eligible.CharacterCardID,
		CharacterName:   characterName,
		ShowID:          eligible.ShowID,
	}, nil
}

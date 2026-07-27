// Package aftercare is Kernel 75's optional post-Show reflection (S8).
//
// Three qualitative prompts, every field optional, with Save and Close or an
// explicitly confirmed Skip. It never blocks play, and it is delivered
// READ-ONLY to Directors+ (S1.15) -- this package exposes no reply,
// annotation, score, or comment write path, and adding one would be a
// product decision, not a refactor.
//
// Qualitative means qualitative (S1.12): there are no numeric ratings here,
// no scale, and no score. Do not add one.
//
// The prompts live in Go rather than in a table. S1.12 fixes them at three,
// and a prompts table would be a form builder -- the same boundary Kernel
// 74 drew when it gave freeform submissions a fixed config key set instead
// of an authoring surface.
package aftercare

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PromptSetVersion identifies the prompt wording answers were written
// against. Stored on every submission so a later rewrite never retroactively
// mislabels an old answer.
const PromptSetVersion = 1

// Prompt is one authored question.
type Prompt struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	MaxLength   int    `json:"max_length"`
	AllowPerson bool   `json:"allow_person"`
}

// PromptSetV1 is S1.12's three locked prompts, verbatim.
var PromptSetV1 = []Prompt{
	{Key: "favorite_moments", Label: "What were your favorite moments from the Show?", MaxLength: 2000},
	{Key: "who_surprised", Label: "Who surprised you the most?", MaxLength: 2000, AllowPerson: true},
	{Key: "next_session", Label: "What do you hope to see in the next Session?", MaxLength: 2000},
}

func promptByKey(key string) (Prompt, bool) {
	for _, p := range PromptSetV1 {
		if p.Key == key {
			return p, true
		}
	}
	return Prompt{}, false
}

// ValidateResponses accepts only known prompt keys and enforces each
// prompt's own length cap.
//
// Unknown keys are an ERROR rather than being silently dropped: a client
// sending a key this server does not know is either a version skew or a
// probe, and both deserve a visible failure rather than quiet data loss.
//
// Every field is optional (S8.1), so an entirely empty map is valid and
// submits successfully -- that is what "Save and Close may submit partial
// responses" means (S1.13).
func ValidateResponses(in map[string]string) (map[string]string, error) {
	out := map[string]string{}
	for key, value := range in {
		prompt, ok := promptByKey(key)
		if !ok {
			return nil, errors.New("unknown_prompt_key")
		}
		value = strings.TrimSpace(value)
		if len([]rune(value)) > prompt.MaxLength {
			return nil, errors.New("response_too_long")
		}
		if value != "" {
			out[key] = value
		}
	}
	return out, nil
}

// Participation is the already-authorized identity an Aftercare record
// belongs to. This package never derives identity from a client payload and
// never re-checks authority -- the HTTP layer resolves it through
// merchant.ResolveShowParticipation, the same single gate every other
// participant action uses.
type Participation struct {
	UserID          string
	CharacterCardID string
	ShowID          string
	ShowRunID       string
	SessionID       string
}

func (p Participation) valid() error {
	if strings.TrimSpace(p.UserID) == "" {
		return errors.New("not_authenticated")
	}
	if strings.TrimSpace(p.CharacterCardID) == "" {
		return errors.New("no_character_selected")
	}
	if strings.TrimSpace(p.ShowID) == "" {
		return errors.New("no_active_show")
	}
	return nil
}

func nullableID(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

// Draft is unsubmitted work in progress.
type Draft struct {
	Responses map[string]string `json:"responses"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Submission is a Player's finished (possibly partial) Aftercare.
type Submission struct {
	Responses        map[string]string `json:"responses"`
	PromptSetVersion int               `json:"prompt_set_version"`
	SubmittedAt      time.Time         `json:"submitted_at"`
}

// State is what the Aftercare form renders from.
type State struct {
	Prompts    []Prompt    `json:"prompts"`
	Draft      *Draft      `json:"draft,omitempty"`
	Submission *Submission `json:"submission,omitempty"`
	// ConsecutiveSkips is computed, never stored. See migration 073.
	ConsecutiveSkips int `json:"consecutive_skips"`
	// Resolved reports whether this Player has either submitted or skipped
	// for this Show -- used to decide whether to offer the form again.
	Resolved bool `json:"resolved"`
}

// Offer loads everything the form needs.
func Offer(ctx context.Context, pool *pgxpool.Pool, p Participation) (State, error) {
	if err := p.valid(); err != nil {
		return State{}, err
	}
	state := State{Prompts: PromptSetV1}

	var draftJSON []byte
	var draftUpdated time.Time
	err := pool.QueryRow(ctx, `
		SELECT responses_json, updated_at FROM aftercare_response_drafts
		WHERE user_id = $1 AND character_card_id = $2 AND show_id = $3
	`, p.UserID, p.CharacterCardID, p.ShowID).Scan(&draftJSON, &draftUpdated)
	if err == nil {
		d := Draft{Responses: map[string]string{}, UpdatedAt: draftUpdated}
		_ = json.Unmarshal(draftJSON, &d.Responses)
		state.Draft = &d
	}

	var subJSON []byte
	var submittedAt time.Time
	var version int
	err = pool.QueryRow(ctx, `
		SELECT responses_json, prompt_set_version, submitted_at FROM aftercare_submissions
		WHERE user_id = $1 AND character_card_id = $2 AND show_id = $3
	`, p.UserID, p.CharacterCardID, p.ShowID).Scan(&subJSON, &version, &submittedAt)
	if err == nil {
		s := Submission{Responses: map[string]string{}, PromptSetVersion: version, SubmittedAt: submittedAt}
		_ = json.Unmarshal(subJSON, &s.Responses)
		state.Submission = &s
		state.Resolved = true
	}

	skips, err := ConsecutiveSkips(ctx, pool, p.UserID)
	if err != nil {
		return State{}, err
	}
	state.ConsecutiveSkips = skips
	if skips > 0 && state.Submission == nil {
		// A skip for THIS Show also counts as resolved, so the form is not
		// pushed at the Player again on the same visit.
		var skippedHere bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM aftercare_skips
			WHERE user_id = $1 AND character_card_id = $2 AND show_id = $3)
		`, p.UserID, p.CharacterCardID, p.ShowID).Scan(&skippedHere); err != nil {
			return State{}, err
		}
		state.Resolved = skippedHere
	}
	return state, nil
}

// ConsecutiveSkips counts explicit skips since this Player's most recent
// submission (S8.5).
//
// COMPUTED, never stored. Two consequences, both requirements:
//
//   - Submitting resets the count to zero by construction (S8.3). There is
//     no reset code to forget to call and no counter to decrement.
//   - Closing the panel, refreshing, abandoning a draft, or never opening
//     Aftercare cannot possibly affect it, because none of them writes an
//     aftercare_skips row. S8.5's list of things that must not count is
//     satisfied by there being no route that could.
//
// Player-level, not per-Character: S8.5 says "at the Player level", and the
// question the warning asks -- "are you drifting away from reflecting?" --
// is about the person, not the Character.
func ConsecutiveSkips(ctx context.Context, pool *pgxpool.Pool, userID string) (int, error) {
	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM aftercare_skips
		WHERE user_id = $1
		  AND skipped_at > COALESCE(
		        (SELECT MAX(submitted_at) FROM aftercare_submissions WHERE user_id = $1),
		        '-infinity'::timestamptz)
	`, userID).Scan(&count)
	return count, err
}

// SaveDraft upserts work in progress. Never counts as a submission and
// never touches the skip count (S8.2).
func SaveDraft(ctx context.Context, pool *pgxpool.Pool, p Participation, responses map[string]string) (Draft, error) {
	if err := p.valid(); err != nil {
		return Draft{}, err
	}
	clean, err := ValidateResponses(responses)
	if err != nil {
		return Draft{}, err
	}
	encoded, err := json.Marshal(clean)
	if err != nil {
		return Draft{}, err
	}
	var updatedAt time.Time
	if err := pool.QueryRow(ctx, `
		INSERT INTO aftercare_response_drafts (user_id, character_card_id, show_id, responses_json, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, NOW())
		ON CONFLICT (user_id, character_card_id, show_id)
		DO UPDATE SET responses_json = EXCLUDED.responses_json, updated_at = NOW()
		RETURNING updated_at
	`, p.UserID, p.CharacterCardID, p.ShowID, encoded).Scan(&updatedAt); err != nil {
		return Draft{}, err
	}
	return Draft{Responses: clean, UpdatedAt: updatedAt}, nil
}

// Submit records a (possibly partial) Aftercare submission (S8.3).
//
// Accepts partial answers, resets the consecutive-skip count to zero by
// construction, and clears the draft so a stale draft cannot later be
// mistaken for newer thinking than the submission.
func Submit(ctx context.Context, pool *pgxpool.Pool, p Participation, responses map[string]string) (Submission, error) {
	if err := p.valid(); err != nil {
		return Submission{}, err
	}
	clean, err := ValidateResponses(responses)
	if err != nil {
		return Submission{}, err
	}
	encoded, err := json.Marshal(clean)
	if err != nil {
		return Submission{}, err
	}
	var submittedAt time.Time
	if err := pool.QueryRow(ctx, `
		INSERT INTO aftercare_submissions (
			user_id, character_card_id, show_id, show_run_id, session_id,
			prompt_set_version, responses_json, submitted_at, updated_at
		)
		VALUES ($1, $2, $3, $4::uuid, $5::uuid, $6, $7::jsonb, NOW(), NOW())
		ON CONFLICT (user_id, character_card_id, show_id)
		DO UPDATE SET responses_json = EXCLUDED.responses_json,
		              prompt_set_version = EXCLUDED.prompt_set_version,
		              updated_at = NOW()
		RETURNING submitted_at
	`, p.UserID, p.CharacterCardID, p.ShowID, nullableID(p.ShowRunID), nullableID(p.SessionID),
		PromptSetVersion, encoded).Scan(&submittedAt); err != nil {
		return Submission{}, err
	}

	if _, err := pool.Exec(ctx, `
		DELETE FROM aftercare_response_drafts
		WHERE user_id = $1 AND character_card_id = $2 AND show_id = $3
	`, p.UserID, p.CharacterCardID, p.ShowID); err != nil {
		return Submission{}, err
	}

	return Submission{Responses: clean, PromptSetVersion: PromptSetVersion, SubmittedAt: submittedAt}, nil
}

// Skip records an explicitly confirmed skip (S8.4).
//
// confirmed is required. S1.14 makes the skip a two-step act: the warning
// panel with its factual consecutive-skip context, then Continue Without
// Aftercare. A client posting without confirmation is refused here rather
// than being trusted to have shown the panel -- the count must reflect a
// deliberate choice, not an accidental one.
func Skip(ctx context.Context, pool *pgxpool.Pool, p Participation, confirmed bool) error {
	if err := p.valid(); err != nil {
		return err
	}
	if !confirmed {
		return errors.New("confirmation_required")
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO aftercare_skips (user_id, character_card_id, show_id)
		VALUES ($1, $2, $3)
	`, p.UserID, p.CharacterCardID, p.ShowID)
	return err
}

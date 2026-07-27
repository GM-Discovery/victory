// Package tutorial is Kernel 74's bounded participant tutorial-progress
// record: five enumerated milestones for one Player's (user, Character,
// Show) participation context, written idempotently and read back to decide
// what that Player may reach next.
//
// This is deliberately NOT a quest tracker (kernel-74 S5.1, S15). There is
// no milestone registry, no authoring surface, no ordering engine, and no
// generic key/value store -- the milestone set is a CHECK constraint in
// migration 066 and the mirrored constant list below. Adding a sixth
// milestone is a migration, on purpose: that friction is what keeps this
// from growing into the general progression system the kernel excludes.
//
// This package performs no authorization of its own. Every caller must have
// already resolved a server-authoritative participation context (in
// practice merchant.ResolveEligibleContext) -- see Participation.
package tutorial

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The milestones of the Locked Courtyard tutorial (kernel-74 S5.1,
// kernel-75 S3.1/S3.2). Kept in sync by hand with
// participant_tutorial_progress_milestone_check in migrations 066 and 069;
// IsMilestone below is the Go-side gate so a typo fails as a clean error
// rather than a constraint violation surfacing as a 500.
//
// MilestoneTutorialGateOpened and MilestoneTutorialCompleted are Kernel 75's
// two additions. They bracket the Player's Continue press: the gate opening
// is a narrative fact recorded when Ra's closing beats are generated, and
// tutorial_completed is the Player's own acknowledgement. The latter is the
// idempotency anchor for Story So Far generation and one-time Player
// recognition -- both ask "was this already recorded" rather than
// read-then-write, so a retried Continue converges.
const (
	MilestoneKessaIntroCompleted    = "kessa_intro_completed"
	MilestoneDoorIntentionSubmitted = "door_intention_submitted"
	MilestoneRaIntroStarted         = "ra_intro_started"
	MilestoneRaIntroCompleted       = "ra_intro_completed"
	MilestoneTutorialGateOpened     = "tutorial_gate_opened"
	MilestoneTutorialHandoffEntered = "tutorial_handoff_entered"
	MilestoneTutorialCompleted      = "tutorial_completed"
)

var allMilestones = []string{
	MilestoneKessaIntroCompleted,
	MilestoneDoorIntentionSubmitted,
	MilestoneRaIntroStarted,
	MilestoneRaIntroCompleted,
	MilestoneTutorialGateOpened,
	MilestoneTutorialHandoffEntered,
	MilestoneTutorialCompleted,
}

func IsMilestone(key string) bool {
	key = strings.TrimSpace(key)
	for _, m := range allMilestones {
		if m == key {
			return true
		}
	}
	return false
}

// Participation is the already-authorized identity a milestone belongs to.
// Callers construct it from a resolved eligibility context; this package
// never derives identity from a client payload and never re-checks
// authority, because doing so here would duplicate (and eventually drift
// from) merchant.ResolveEligibleContext's single gate.
//
// CharacterCardID is part of the key, not decoration: kernel-74 S5.2
// requires that switching the selected Character not transfer the previous
// Character's completion, and that prior records stay associated with the
// Character who performed them.
type Participation struct {
	UserID          string
	CharacterCardID string
	ShowID          string
	ShowRunID       string
	PlacementID     string
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

// RecordMilestone writes a milestone idempotently (kernel-74 S5.3). A
// repeated call -- double-click, retry, reconnect, replayed request -- is a
// no-op rather than a duplicate row, enforced by the UNIQUE constraint
// rather than a read-then-write race.
//
// payload is small authored context for backstage display (e.g. which
// dialogue packet was completed); it is never read to make a decision, so a
// second call losing its payload to ON CONFLICT DO NOTHING is harmless.
func RecordMilestone(ctx context.Context, pool *pgxpool.Pool, p Participation, milestoneKey string, payload []byte) error {
	if err := p.valid(); err != nil {
		return err
	}
	milestoneKey = strings.TrimSpace(milestoneKey)
	if !IsMilestone(milestoneKey) {
		return errors.New("invalid_milestone")
	}
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO participant_tutorial_progress (
			user_id, character_card_id, show_id, show_run_id,
			show_scene_placement_id, milestone_key, payload_json
		)
		VALUES ($1, $2, $3, $4::uuid, $5::uuid, $6, $7::jsonb)
		ON CONFLICT (user_id, character_card_id, show_id, milestone_key) DO NOTHING
	`, p.UserID, p.CharacterCardID, p.ShowID, nullableID(p.ShowRunID),
		nullableID(p.PlacementID), milestoneKey, payload)
	return err
}

// HasMilestone is the read half of the reveal gate. Note the (user,
// character, show) key: Character A completing Kessa must not unlock
// anything for Character B, even for the same user in the same Show.
func HasMilestone(ctx context.Context, pool *pgxpool.Pool, p Participation, milestoneKey string) (bool, error) {
	if err := p.valid(); err != nil {
		return false, err
	}
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM participant_tutorial_progress
			WHERE user_id = $1 AND character_card_id = $2 AND show_id = $3 AND milestone_key = $4
		)
	`, p.UserID, p.CharacterCardID, p.ShowID, strings.TrimSpace(milestoneKey)).Scan(&exists)
	return exists, err
}

// LoadProgress returns the milestone keys this participation has recorded,
// in the order they were reached.
func LoadProgress(ctx context.Context, pool *pgxpool.Pool, p Participation) ([]string, error) {
	if err := p.valid(); err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT milestone_key FROM participant_tutorial_progress
		WHERE user_id = $1 AND character_card_id = $2 AND show_id = $3
		ORDER BY created_at ASC
	`, p.UserID, p.CharacterCardID, p.ShowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	return out, rows.Err()
}

// CharacterProgress is one Character's furthest tutorial position, for the
// optional Directors+ status list (kernel-74 S11.5). Deliberately a flat
// list with no denominator: S1.11 forbids "2 of 4 Players ready" framing
// because the system must not depend on a declared or inferred group size.
type CharacterProgress struct {
	UserID          string `json:"user_id"`
	CharacterCardID string `json:"character_card_id"`
	CharacterName   string `json:"character_name"`
	Status          string `json:"status"`
}

// statusForMilestones maps a milestone set to the single human phrase the
// backstage list shows. Ordered most-advanced-first.
func statusForMilestones(seen map[string]bool) string {
	switch {
	case seen[MilestoneTutorialCompleted]:
		return "Tutorial complete"
	case seen[MilestoneTutorialHandoffEntered]:
		return "Tutorial handoff entered"
	case seen[MilestoneTutorialGateOpened]:
		return "Gate opened"
	case seen[MilestoneRaIntroCompleted]:
		return "Finished with Ra"
	case seen[MilestoneRaIntroStarted]:
		return "Speaking with Ra"
	case seen[MilestoneDoorIntentionSubmitted]:
		return "At the locked door"
	case seen[MilestoneKessaIntroCompleted]:
		return "Left Kessa's stall"
	default:
		return "At Kessa's stall"
	}
}

// ListShowProgress is the Directors+ status list for one Show. No authority
// check here -- the HTTP layer gates it, exactly as with the rest of this
// package.
func ListShowProgress(ctx context.Context, pool *pgxpool.Pool, showID string) ([]CharacterProgress, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			p.user_id::text,
			p.character_card_id::text,
			COALESCE(NULLIF(cc.name, ''), 'Unnamed Character'),
			p.milestone_key
		FROM participant_tutorial_progress p
		LEFT JOIN character_cards cc ON cc.id = p.character_card_id
		WHERE p.show_id = $1
		ORDER BY p.created_at ASC
	`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	order := []string{}
	byCharacter := map[string]*CharacterProgress{}
	seenByCharacter := map[string]map[string]bool{}

	for rows.Next() {
		var userID, characterID, characterName, milestone string
		if err := rows.Scan(&userID, &characterID, &characterName, &milestone); err != nil {
			return nil, err
		}
		if _, ok := byCharacter[characterID]; !ok {
			order = append(order, characterID)
			byCharacter[characterID] = &CharacterProgress{
				UserID: userID, CharacterCardID: characterID, CharacterName: characterName,
			}
			seenByCharacter[characterID] = map[string]bool{}
		}
		seenByCharacter[characterID][milestone] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]CharacterProgress, 0, len(order))
	for _, id := range order {
		entry := byCharacter[id]
		entry.Status = statusForMilestones(seenByCharacter[id])
		out = append(out, *entry)
	}
	return out, nil
}

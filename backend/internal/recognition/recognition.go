// Package recognition is Kernel 75's one-time, Player-level acknowledgement
// that someone finished the guided tutorial for the first time (S7.1, S7.3).
//
// WHAT THIS IS DELIBERATELY NOT (S7.4, S14): experience points, levels, a
// point economy, a level curve, a badge catalog, a rewards shop, streaks,
// leaderboards, or anything consumable, ordered, or spendable. There is no
// numeric column here and no ordering between keys, because K75 explicitly
// does not define progression balance and an unused numeric field is an
// invitation to invent one.
//
// The extension points S1.10 asks for are left as a SHAPE, not as a system:
// player_recognition_grants can carry more keys, and a future kernel that
// genuinely needs points or titles adds the column it needs then, with the
// balance decisions made deliberately. Adding a second recognition key today
// is a migration -- the same friction tutorial/progress.go uses to keep
// itself from drifting into a quest tracker.
//
// The anti-duplication rule S7.3 requires is enforced by the schema, not by
// Go: player_recognition_grants is UNIQUE on (user_id, recognition_key),
// keyed on the user ALONE and never on the Character. A Player completing
// the tutorial again with a second Character collides and is told so, while
// the Character-level acknowledgement lives in character_story_events and is
// correctly earned again. That split between two tables is the entire answer
// to "first-time versus repeat".
package recognition

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// KeyFirstTutorialCompleted is the only recognition key that exists.
// Mirrors player_recognition_grants_key_check in migration 070.
const KeyFirstTutorialCompleted = "first_tutorial_completed"

// Label is the operator-facing name. S7.2 notes the operator may rename the
// badge/title before deployment; this is the one place to change it.
const LabelFirstTutorialCompleted = "First Curtain"

// Grant is one recorded recognition.
type Grant struct {
	Key                  string    `json:"key"`
	Label                string    `json:"label"`
	FirstCharacterCardID string    `json:"first_character_card_id,omitempty"`
	FirstShowID          string    `json:"first_show_id,omitempty"`
	GrantedAt            time.Time `json:"granted_at"`
}

// State is what the completion Program renders.
type State struct {
	// NewlyGranted is true only on the very first tutorial completion by
	// this Player, ever. On a replay with a second Character it is false and
	// the Program shows the Character-level line instead -- the visible half
	// of S7.3.
	NewlyGranted bool    `json:"newly_granted"`
	Grant        *Grant  `json:"grant,omitempty"`
}

func nullableID(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

// GrantFirstTutorialCompleted records the one-time Player recognition.
//
// Returns newlyGranted=false when this Player already had it. That answer
// comes from INSERT ... ON CONFLICT DO NOTHING RETURNING id finding no row,
// not from a preceding SELECT: a read-then-write would race two concurrent
// Continue presses into two grants, which is exactly the duplication S7.3
// forbids.
//
// No authority check here. The caller has already resolved a
// server-authoritative participation context, and a Player cannot reach this
// function without having actually recorded the completion milestone -- so
// "Player cannot award themself milestones" (S12) holds because there is no
// route that takes a recognition key from a client.
func GrantFirstTutorialCompleted(ctx context.Context, pool *pgxpool.Pool, userID, characterCardID, showID string) (State, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return State{}, errors.New("not_authenticated")
	}

	var grantedAt time.Time
	err := pool.QueryRow(ctx, `
		INSERT INTO player_recognition_grants (
			user_id, recognition_key, first_character_card_id, first_show_id
		)
		VALUES ($1, $2, $3::uuid, $4::uuid)
		ON CONFLICT (user_id, recognition_key) DO NOTHING
		RETURNING granted_at
	`, userID, KeyFirstTutorialCompleted, nullableID(characterCardID), nullableID(showID)).Scan(&grantedAt)

	switch {
	case err == nil:
		return State{
			NewlyGranted: true,
			Grant: &Grant{
				Key:                  KeyFirstTutorialCompleted,
				Label:                LabelFirstTutorialCompleted,
				FirstCharacterCardID: characterCardID,
				FirstShowID:          showID,
				GrantedAt:            grantedAt,
			},
		}, nil
	case errors.Is(err, pgx.ErrNoRows):
		// Already held. Load the original so the Program can say when.
		existing, loadErr := LoadForUser(ctx, pool, userID)
		if loadErr != nil {
			return State{}, loadErr
		}
		for i := range existing {
			if existing[i].Key == KeyFirstTutorialCompleted {
				return State{NewlyGranted: false, Grant: &existing[i]}, nil
			}
		}
		return State{NewlyGranted: false}, nil
	default:
		return State{}, err
	}
}

// LoadForUser lists a Player's recognitions. Scoped to the authenticated
// user by the caller; there is no route that reads another Player's.
func LoadForUser(ctx context.Context, pool *pgxpool.Pool, userID string) ([]Grant, error) {
	rows, err := pool.Query(ctx, `
		SELECT recognition_key,
		       COALESCE(first_character_card_id::text, ''),
		       COALESCE(first_show_id::text, ''),
		       granted_at
		FROM player_recognition_grants
		WHERE user_id = $1
		ORDER BY granted_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Grant{}
	for rows.Next() {
		var g Grant
		if err := rows.Scan(&g.Key, &g.FirstCharacterCardID, &g.FirstShowID, &g.GrantedAt); err != nil {
			return nil, err
		}
		g.Label = labelFor(g.Key)
		out = append(out, g)
	}
	return out, rows.Err()
}

func labelFor(key string) string {
	if key == KeyFirstTutorialCompleted {
		return LabelFirstTutorialCompleted
	}
	return key
}

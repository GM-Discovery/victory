package dialogue

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/tutorial"
)

// BuildOpenState renders the Program on open or resume (S9.1, S9.5).
//
// CurrentResponse is the packet's opening line on a first open. On a resume
// it is still the opening line rather than the last topic read: the last
// response is not persisted, and inventing a "you were here" replay would
// mean storing conversation transcript state this kernel does not need. The
// Player's actual progress -- which topics are seen and unlocked, and
// whether Leave is earned -- is fully restored, which is what S9.5 asks for.
func BuildOpenState(ctx context.Context, pool *pgxpool.Pool, p tutorial.Participation, packet Packet, topics []Topic) (State, error) {
	seen, err := loadSeenTopicKeys(ctx, pool, p, packet.ID)
	if err != nil {
		return State{}, err
	}
	return buildState(packet, topics, seen, packet.OpeningLine), nil
}

// ReadTopic authorizes, records, and returns one authored response.
//
// Order matters here: the unlock check runs against the seen-set as it was
// BEFORE this topic is marked, so a topic can never satisfy its own
// prerequisite. The returned state is rebuilt from the post-write seen-set
// so newly unlocked topics appear in the same response.
func ReadTopic(ctx context.Context, pool *pgxpool.Pool, p tutorial.Participation, packet Packet, topics []Topic, topicKey string) (State, error) {
	topicKey = strings.TrimSpace(topicKey)
	if topicKey == "" {
		return State{}, errors.New("topic_key_required")
	}

	var target *Topic
	for i := range topics {
		if topics[i].TopicKey == topicKey {
			target = &topics[i]
			break
		}
	}
	if target == nil {
		return State{}, errors.New("topic_not_found")
	}

	seen, err := loadSeenTopicKeys(ctx, pool, p, packet.ID)
	if err != nil {
		return State{}, err
	}
	if !topicUnlocked(*target, seen) {
		return State{}, errors.New("topic_locked")
	}

	// Idempotent: re-reading an already-seen topic returns the same response
	// and does not create a duplicate view row (S5.3).
	if _, err := pool.Exec(ctx, `
		INSERT INTO participant_dialogue_topic_views (user_id, character_card_id, show_id, topic_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, character_card_id, show_id, topic_id) DO NOTHING
	`, p.UserID, p.CharacterCardID, p.ShowID, target.ID); err != nil {
		return State{}, err
	}
	seen[topicKey] = true

	return buildState(packet, topics, seen, target.ResponseText), nil
}

// LeaveState authorizes Leave and returns the closing/unlock narration.
//
// This is the server-side half of S13's "Player cannot skip required Ra
// topics by posting completion directly": the CanLeave the client renders
// and the refusal here are computed by the same canLeave function over the
// same rows, so a forged Leave with unmet required topics returns
// required_topics_unseen rather than the closing narration.
func LeaveState(ctx context.Context, pool *pgxpool.Pool, p tutorial.Participation, packet Packet, topics []Topic) (State, error) {
	seen, err := loadSeenTopicKeys(ctx, pool, p, packet.ID)
	if err != nil {
		return State{}, err
	}
	if !canLeave(topics, seen) {
		return State{}, errors.New("required_topics_unseen")
	}
	state := buildState(packet, topics, seen, packet.OpeningLine)
	// The lock reveal (S10.1): Ra operates a concealed courtyard-side
	// mechanism. Only Leave carries this -- it is the beat that makes the
	// door's "no obvious lock or keyhole" opening description honest rather
	// than contradicted.
	state.ClosingNarration = packet.ClosingNarration
	state.CurrentResponse = ""
	return state, nil
}

package dialogue

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/tutorial"
)

const packetColumns = `
	id::text, location_id::text, slug, npc_name,
	COALESCE(portrait_asset_id::text, ''), portrait_url,
	opening_narration, opening_line, closing_narration, leave_label,
	destination_scene_slug, active, created_at, updated_at,
	closing_beats, content_origin
`

func scanPacket(row pgx.Row) (Packet, error) {
	var p Packet
	var closingBeats []byte
	if err := row.Scan(
		&p.ID, &p.LocationID, &p.Slug, &p.NPCName,
		&p.PortraitAssetID, &p.PortraitURL,
		&p.OpeningNarration, &p.OpeningLine, &p.ClosingNarration, &p.LeaveLabel,
		&p.DestinationSceneSlug, &p.Active, &p.CreatedAt, &p.UpdatedAt,
		&closingBeats, &p.ContentOrigin,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Packet{}, errors.New("dialogue_packet_not_found")
		}
		return Packet{}, err
	}
	if len(closingBeats) > 0 {
		_ = json.Unmarshal(closingBeats, &p.ClosingBeats)
	}
	return p, nil
}

// ClosingBeatsOrNarration is the one place the empty-array fallback lives
// (kernel-75 S3.1).
//
// closing_beats arrived in migration 072; a packet seeded before it, or one
// a Director has deliberately left as a single paragraph, has an empty
// array. Callers must never read ClosingBeats directly -- going through this
// helper is what lets an unedited packet keep working and what lets the
// frontend and backend deploy in either order.
func (p Packet) ClosingBeatsOrNarration() []string {
	out := []string{}
	for _, beat := range p.ClosingBeats {
		if strings.TrimSpace(beat) != "" {
			out = append(out, beat)
		}
	}
	if len(out) > 0 {
		return out
	}
	if strings.TrimSpace(p.ClosingNarration) != "" {
		return []string{p.ClosingNarration}
	}
	return []string{}
}

// LoadPacketBySlug mirrors merchant.LoadPacketBySlug exactly: packets are
// location-scoped, so the same slug in two Locations is two packets.
func LoadPacketBySlug(ctx context.Context, pool *pgxpool.Pool, locationID, slug string) (Packet, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return Packet{}, errors.New("dialogue_packet_not_found")
	}
	row := pool.QueryRow(ctx, `SELECT `+packetColumns+`
		FROM dialogue_packets WHERE location_id = $1 AND slug = $2 AND active`, locationID, slug)
	return scanPacket(row)
}

// LoadTopics returns the packet's active topics in author order, each with
// its prerequisite topic keys resolved.
func LoadTopics(ctx context.Context, pool *pgxpool.Pool, packetID string) ([]Topic, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, packet_id::text, topic_key, label, response_text,
		       required_for_completion, sort_order, active
		FROM dialogue_topics
		WHERE packet_id = $1 AND active
		ORDER BY sort_order ASC, topic_key ASC
	`, packetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	topics := []Topic{}
	byID := map[string]int{}
	for rows.Next() {
		var t Topic
		if err := rows.Scan(&t.ID, &t.PacketID, &t.TopicKey, &t.Label, &t.ResponseText,
			&t.RequiredForCompletion, &t.SortOrder, &t.Active); err != nil {
			return nil, err
		}
		byID[t.ID] = len(topics)
		topics = append(topics, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	preqRows, err := pool.Query(ctx, `
		SELECT p.topic_id::text, req.topic_key
		FROM dialogue_topic_prerequisites p
		JOIN dialogue_topics req ON req.id = p.requires_topic_id
		JOIN dialogue_topics t ON t.id = p.topic_id
		WHERE t.packet_id = $1
	`, packetID)
	if err != nil {
		return nil, err
	}
	defer preqRows.Close()
	for preqRows.Next() {
		var topicID, requiresKey string
		if err := preqRows.Scan(&topicID, &requiresKey); err != nil {
			return nil, err
		}
		if idx, ok := byID[topicID]; ok {
			topics[idx].RequiresTopicKeys = append(topics[idx].RequiresTopicKeys, requiresKey)
		}
	}
	if err := preqRows.Err(); err != nil {
		return nil, err
	}
	for i := range topics {
		sort.Strings(topics[i].RequiresTopicKeys)
	}

	if err := validateAcyclic(topics); err != nil {
		return nil, err
	}
	return topics, nil
}

// validateAcyclic refuses a packet whose prerequisites form a cycle. S9.2
// forbids arbitrary cyclic graphs; without this check a bad seed would not
// error, it would simply leave every topic in the cycle permanently locked
// and the Player permanently unable to Leave -- a far worse failure than a
// loud one. Reported as a server error, not a Player-facing state.
func validateAcyclic(topics []Topic) error {
	byKey := map[string]Topic{}
	for _, t := range topics {
		byKey[t.TopicKey] = t
	}
	const (
		unvisited = 0
		visiting  = 1
		done      = 2
	)
	state := map[string]int{}

	var walk func(key string) error
	walk = func(key string) error {
		switch state[key] {
		case visiting:
			return errors.New("dialogue_packet_cyclic")
		case done:
			return nil
		}
		state[key] = visiting
		for _, req := range byKey[key].RequiresTopicKeys {
			if _, ok := byKey[req]; !ok {
				// A prerequisite pointing at a missing/inactive topic would
				// lock its dependant forever. Same reasoning as a cycle.
				return errors.New("dialogue_packet_unreachable_topic")
			}
			if err := walk(req); err != nil {
				return err
			}
		}
		state[key] = done
		return nil
	}

	for _, t := range topics {
		if err := walk(t.TopicKey); err != nil {
			return err
		}
	}
	return nil
}

// loadSeenTopicKeys reads this participation's viewed topics. Keyed on
// (user, character, show) so Character B never inherits Character A's
// conversation (S5.2).
func loadSeenTopicKeys(ctx context.Context, pool *pgxpool.Pool, p tutorial.Participation, packetID string) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `
		SELECT t.topic_key
		FROM participant_dialogue_topic_views v
		JOIN dialogue_topics t ON t.id = v.topic_id
		WHERE v.user_id = $1 AND v.character_card_id = $2 AND v.show_id = $3
		  AND t.packet_id = $4
	`, p.UserID, p.CharacterCardID, p.ShowID, packetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := map[string]bool{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		seen[key] = true
	}
	return seen, rows.Err()
}

// topicUnlocked is the single definition of "may this Player ask this
// topic". Both the rendered state and the server-side refusal in ReadTopic
// call it, so the button the Player sees and the gate the server enforces
// can never disagree.
func topicUnlocked(t Topic, seen map[string]bool) bool {
	for _, req := range t.RequiresTopicKeys {
		if !seen[req] {
			return false
		}
	}
	return true
}

// canLeave reports whether every required topic has been viewed (S9.3).
// Optional topics never block.
func canLeave(topics []Topic, seen map[string]bool) bool {
	for _, t := range topics {
		if t.RequiredForCompletion && !seen[t.TopicKey] {
			return false
		}
	}
	return true
}

// buildState assembles the Program Panel payload. currentResponse is the
// caller's choice of what line is showing right now.
func buildState(packet Packet, topics []Topic, seen map[string]bool, currentResponse string) State {
	views := make([]TopicView, 0, len(topics))
	for _, t := range topics {
		views = append(views, TopicView{
			TopicKey:  t.TopicKey,
			Label:     t.Label,
			Seen:      seen[t.TopicKey],
			Unlocked:  topicUnlocked(t, seen),
			Required:  t.RequiredForCompletion,
			SortOrder: t.SortOrder,
		})
	}
	return State{
		PacketSlug:       packet.Slug,
		NPCName:          packet.NPCName,
		PortraitAssetID:  packet.PortraitAssetID,
		PortraitURL:      packet.PortraitURL,
		OpeningNarration: packet.OpeningNarration,
		CurrentResponse:  currentResponse,
		Topics:           views,
		LeaveLabel:       packet.LeaveLabel,
		CanLeave:         canLeave(topics, seen),
	}
}

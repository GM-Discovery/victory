package dialogue

// Director+ authoring for guided-dialogue packets (kernel-75, operator
// decision 3).
//
// Ra's prose shipped as seed data in migration 066 with no editing surface,
// so every wording change was a migration. This is that surface.
//
// BOUNDED ON PURPOSE. This edits TEXT AND ORDERING ONLY. It cannot create or
// delete topics, cannot edit prerequisite edges, and cannot change
// destination_scene_slug. Those are structural facts a Player's reachability
// depends on -- a deleted topic can strand someone mid-conversation, and a
// changed destination silently reroutes where the tutorial ends. They remain
// migrations, which is what keeps this from becoming the arbitrary
// dialogue-graph engine S14 excludes.
//
// Every write flips the edited row's content_origin from 'seed' to
// 'authored', which is what stops a future content migration from clobbering
// an operator's words -- see migration 072.

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Revision is one recorded edit.
type Revision struct {
	ID           string    `json:"id"`
	PacketID     string    `json:"packet_id"`
	TopicID      string    `json:"topic_id,omitempty"`
	FieldKey     string    `json:"field_key"`
	PreviousText string    `json:"previous_text"`
	NextText     string    `json:"next_text"`
	EditedByName string    `json:"edited_by_name,omitempty"`
	EditedAt     time.Time `json:"edited_at"`
}

// PacketPatch carries only the fields an editor may change. Pointers so
// "absent" and "set to empty" are distinguishable -- a Director clearing the
// opening narration is a real edit, not a no-op.
type PacketPatch struct {
	NPCName          *string   `json:"npc_name"`
	PortraitURL      *string   `json:"portrait_url"`
	OpeningNarration *string   `json:"opening_narration"`
	OpeningLine      *string   `json:"opening_line"`
	ClosingNarration *string   `json:"closing_narration"`
	ClosingBeats     *[]string `json:"closing_beats"`
	LeaveLabel       *string   `json:"leave_label"`
}

// TopicPatch is the same idea for one topic. Note the absence of TopicKey
// and of any prerequisite field: keys are referenced by seeds and code, and
// prerequisites decide reachability.
type TopicPatch struct {
	Label                 *string `json:"label"`
	ResponseText          *string `json:"response_text"`
	SortOrder             *int    `json:"sort_order"`
	RequiredForCompletion *bool   `json:"required_for_completion"`
}

// ListPacketsForLocation lists the editable packets at a Location.
func ListPacketsForLocation(ctx context.Context, pool *pgxpool.Pool, locationID string) ([]Packet, error) {
	rows, err := pool.Query(ctx, `SELECT `+packetColumns+`
		FROM dialogue_packets WHERE location_id = $1 ORDER BY slug ASC`, locationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Packet{}
	for rows.Next() {
		// pgx.Rows satisfies pgx.Row, so the single-row scanner works here
		// too and there is only one packet-scanning function to keep in
		// sync with packetColumns.
		p, err := scanPacket(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// LoadPacketForEditing returns a packet with its topics, including each
// topic's response_text -- which the Player-facing TopicView deliberately
// withholds until a topic is unlocked. This is a backstage read, gated by
// the HTTP layer.
func LoadPacketForEditing(ctx context.Context, pool *pgxpool.Pool, packetID string) (Packet, []Topic, error) {
	row := pool.QueryRow(ctx, `SELECT `+packetColumns+` FROM dialogue_packets WHERE id = $1`, packetID)
	packet, err := scanPacket(row)
	if err != nil {
		return Packet{}, nil, err
	}
	topics, err := LoadTopics(ctx, pool, packet.ID)
	if err != nil {
		return Packet{}, nil, err
	}
	return packet, topics, nil
}

func recordRevision(ctx context.Context, pool *pgxpool.Pool, packetID, topicID, field, prev, next, editorUserID string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO dialogue_packet_revisions (packet_id, topic_id, field_key, previous_text, next_text, edited_by_user_id)
		VALUES ($1, $2::uuid, $3, $4, $5, $6::uuid)
	`, packetID, nullableID(topicID), field, prev, next, nullableID(editorUserID))
	return err
}

func nullableID(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

// UpdatePacket applies a patch, recording one revision row per changed
// field and flipping content_origin to 'authored'.
func UpdatePacket(ctx context.Context, pool *pgxpool.Pool, editorUserID, packetID string, patch PacketPatch) (Packet, error) {
	current, _, err := LoadPacketForEditing(ctx, pool, packetID)
	if err != nil {
		return Packet{}, err
	}

	changed := false
	apply := func(field string, next *string, currentValue string, column string) error {
		if next == nil || *next == currentValue {
			return nil
		}
		if _, err := pool.Exec(ctx,
			`UPDATE dialogue_packets SET `+column+` = $2, updated_at = NOW(),
			 updated_by_user_id = $3::uuid, content_origin = 'authored' WHERE id = $1`,
			packetID, *next, nullableID(editorUserID)); err != nil {
			return err
		}
		changed = true
		return recordRevision(ctx, pool, packetID, "", field, currentValue, *next, editorUserID)
	}

	// Column names are literals here, never interpolated from input.
	if err := apply("npc_name", patch.NPCName, current.NPCName, "npc_name"); err != nil {
		return Packet{}, err
	}
	if err := apply("portrait_url", patch.PortraitURL, current.PortraitURL, "portrait_url"); err != nil {
		return Packet{}, err
	}
	if err := apply("opening_narration", patch.OpeningNarration, current.OpeningNarration, "opening_narration"); err != nil {
		return Packet{}, err
	}
	if err := apply("opening_line", patch.OpeningLine, current.OpeningLine, "opening_line"); err != nil {
		return Packet{}, err
	}
	if err := apply("closing_narration", patch.ClosingNarration, current.ClosingNarration, "closing_narration"); err != nil {
		return Packet{}, err
	}
	if err := apply("leave_label", patch.LeaveLabel, current.LeaveLabel, "leave_label"); err != nil {
		return Packet{}, err
	}

	if patch.ClosingBeats != nil {
		beats := []string{}
		for _, b := range *patch.ClosingBeats {
			if strings.TrimSpace(b) != "" {
				beats = append(beats, b)
			}
		}
		prev, _ := json.Marshal(current.ClosingBeats)
		next, err := json.Marshal(beats)
		if err != nil {
			return Packet{}, err
		}
		if string(prev) != string(next) {
			if _, err := pool.Exec(ctx, `
				UPDATE dialogue_packets SET closing_beats = $2::jsonb, updated_at = NOW(),
				updated_by_user_id = $3::uuid, content_origin = 'authored' WHERE id = $1
			`, packetID, next, nullableID(editorUserID)); err != nil {
				return Packet{}, err
			}
			changed = true
			if err := recordRevision(ctx, pool, packetID, "", "closing_beats", string(prev), string(next), editorUserID); err != nil {
				return Packet{}, err
			}
		}
	}

	if !changed {
		return current, nil
	}
	updated, _, err := LoadPacketForEditing(ctx, pool, packetID)
	return updated, err
}

// UpdateTopic applies a topic patch.
//
// AFTER any change it re-runs validateAcyclic over the packet's whole topic
// graph. Today's patch fields cannot introduce a cycle, but this editor is
// exactly the surface where a future field could -- and validateAcyclic
// otherwise only runs on load, which means a bad graph would be discovered
// by a Player mid-conversation rather than by the Director who caused it.
// Failing the write is the cheap end of that trade.
func UpdateTopic(ctx context.Context, pool *pgxpool.Pool, editorUserID, topicID string, patch TopicPatch) (Topic, error) {
	var packetID string
	var label, responseText string
	var sortOrder int
	var required bool
	if err := pool.QueryRow(ctx, `
		SELECT packet_id::text, label, response_text, sort_order, required_for_completion
		FROM dialogue_topics WHERE id = $1
	`, topicID).Scan(&packetID, &label, &responseText, &sortOrder, &required); err != nil {
		return Topic{}, errors.New("dialogue_topic_not_found")
	}

	touch := func(column string, args ...any) error {
		_, err := pool.Exec(ctx,
			`UPDATE dialogue_topics SET `+column+` = $2, updated_at = NOW(),
			 updated_by_user_id = $3::uuid, content_origin = 'authored' WHERE id = $1`,
			append([]any{topicID}, append(args, nullableID(editorUserID))...)...)
		return err
	}

	if patch.Label != nil && *patch.Label != label {
		if strings.TrimSpace(*patch.Label) == "" {
			return Topic{}, errors.New("label_required")
		}
		if err := touch("label", *patch.Label); err != nil {
			return Topic{}, err
		}
		if err := recordRevision(ctx, pool, packetID, topicID, "label", label, *patch.Label, editorUserID); err != nil {
			return Topic{}, err
		}
	}
	if patch.ResponseText != nil && *patch.ResponseText != responseText {
		if strings.TrimSpace(*patch.ResponseText) == "" {
			return Topic{}, errors.New("response_text_required")
		}
		if err := touch("response_text", *patch.ResponseText); err != nil {
			return Topic{}, err
		}
		if err := recordRevision(ctx, pool, packetID, topicID, "response_text", responseText, *patch.ResponseText, editorUserID); err != nil {
			return Topic{}, err
		}
	}
	if patch.SortOrder != nil && *patch.SortOrder != sortOrder {
		if err := touch("sort_order", *patch.SortOrder); err != nil {
			return Topic{}, err
		}
	}
	if patch.RequiredForCompletion != nil && *patch.RequiredForCompletion != required {
		if err := touch("required_for_completion", *patch.RequiredForCompletion); err != nil {
			return Topic{}, err
		}
	}

	topics, err := LoadTopics(ctx, pool, packetID)
	if err != nil {
		return Topic{}, err
	}
	if err := validateAcyclic(topics); err != nil {
		return Topic{}, err
	}
	for _, t := range topics {
		if t.ID == topicID {
			return t, nil
		}
	}
	return Topic{}, errors.New("dialogue_topic_not_found")
}

// ListRevisions returns the edit history for a packet, newest first.
func ListRevisions(ctx context.Context, pool *pgxpool.Pool, packetID string, limit int) ([]Revision, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
		SELECT r.id::text, r.packet_id::text, COALESCE(r.topic_id::text, ''), r.field_key,
		       r.previous_text, r.next_text,
		       COALESCE(NULLIF(u.display_name, ''), COALESCE(u.handle, '')), r.edited_at
		FROM dialogue_packet_revisions r
		LEFT JOIN users u ON u.id = r.edited_by_user_id
		WHERE r.packet_id = $1
		ORDER BY r.edited_at DESC
		LIMIT $2
	`, packetID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Revision{}
	for rows.Next() {
		var rev Revision
		if err := rows.Scan(&rev.ID, &rev.PacketID, &rev.TopicID, &rev.FieldKey,
			&rev.PreviousText, &rev.NextText, &rev.EditedByName, &rev.EditedAt); err != nil {
			return nil, err
		}
		out = append(out, rev)
	}
	return out, rows.Err()
}

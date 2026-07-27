package aftercare

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The Directors+ review surface (kernel-75 S9).
//
// READ-ONLY BY CONSTRUCTION (S1.15, S9.3). This file contains no update,
// no delete, no reply, no annotation, and no score. That is not an omission
// to be filled in later -- a Director replying to Aftercare through this
// surface is explicitly excluded by S14, and adding it would be a product
// decision rather than a refactor.
//
// No authority check lives here. The HTTP layer gates every caller with
// showruns.CanViewBackstage (to read) or CanManageShowRun (to export),
// exactly as tutorial.ListShowProgress does.

// ReviewRow is one (Player, Character) line of the Show's Aftercare table.
//
// DELIBERATELY NO DENOMINATOR. There is no Total, no Expected, no
// PercentComplete, and no ReadyCount field, because S1.11 forbids "2 of 4
// Players ready" framing -- the system must not depend on a declared or
// inferred group size. A type with no such field cannot grow one by
// accident, which is the point.
type ReviewRow struct {
	UserID          string `json:"user_id"`
	PlayerHandle    string `json:"player_handle"`
	PlayerName      string `json:"player_name"`
	CharacterCardID string `json:"character_card_id"`
	CharacterName   string `json:"character_name"`

	// TutorialStatus is the human phrase from tutorial.statusForMilestones,
	// resolved by the caller. Empty when this Player has no recorded
	// tutorial progress.
	TutorialStatus string `json:"tutorial_status"`

	// State distinguishes every case S9.5 requires: "none" (offered but not
	// opened), "draft", "submitted", "skipped".
	State string `json:"state"`

	Responses        map[string]string `json:"responses,omitempty"`
	PromptSetVersion int               `json:"prompt_set_version,omitempty"`
	SubmittedAt      *time.Time        `json:"submitted_at,omitempty"`
	SkippedAt        *time.Time        `json:"skipped_at,omitempty"`
	ConsecutiveSkips int               `json:"consecutive_skips"`

	// DoorIntention is the Player's own words at the locked gate, carried
	// here because S9.2 lists "unresolved door intention" as a column a
	// Director wants beside the reflection. Verbatim, never rewritten.
	DoorIntention string `json:"door_intention,omitempty"`
}

// ListForShow builds the Show's Aftercare table.
//
// The row set is driven by the ROSTER, not by the submissions table, so a
// Player who never opened Aftercare still appears with state "none". A
// review surface that only listed people who answered would quietly hide
// exactly the ones a Director most wants to notice.
func ListForShow(ctx context.Context, pool *pgxpool.Pool, showID string) ([]ReviewRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			m.user_id::text,
			COALESCE(u.handle, ''),
			COALESCE(NULLIF(u.display_name, ''), ''),
			COALESCE(m.character_card_id::text, ''),
			COALESCE(NULLIF(cc.name, ''), 'Unnamed Character'),
			COALESCE(sub.responses_json, '{}'::jsonb),
			COALESCE(sub.prompt_set_version, 0),
			sub.submitted_at,
			skip.skipped_at,
			COALESCE(fs.submitted_text, '')
		FROM show_run_roster_members m
		JOIN shows s ON s.id = $1 AND s.show_run_id = m.show_run_id
		JOIN users u ON u.id = m.user_id
		LEFT JOIN character_cards cc ON cc.id = m.character_card_id
		LEFT JOIN aftercare_submissions sub
		       ON sub.user_id = m.user_id
		      AND sub.character_card_id = m.character_card_id
		      AND sub.show_id = s.id
		LEFT JOIN LATERAL (
			SELECT MAX(skipped_at) AS skipped_at
			FROM aftercare_skips k
			WHERE k.user_id = m.user_id
			  AND k.character_card_id = m.character_card_id
			  AND k.show_id = s.id
		) skip ON TRUE
		LEFT JOIN LATERAL (
			SELECT submitted_text
			FROM participant_freeform_submissions f
			WHERE f.user_id = m.user_id
			  AND f.character_card_id = m.character_card_id
			  AND f.show_id = s.id
			ORDER BY f.created_at ASC
			LIMIT 1
		) fs ON TRUE
		WHERE m.removed_at IS NULL
		  AND LOWER(m.role) = 'player'
		ORDER BY COALESCE(NULLIF(cc.name, ''), u.handle) ASC, m.user_id ASC
	`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ReviewRow{}
	userIDs := []string{}
	for rows.Next() {
		var r ReviewRow
		var responsesJSON []byte
		var submittedAt, skippedAt *time.Time
		if err := rows.Scan(
			&r.UserID, &r.PlayerHandle, &r.PlayerName,
			&r.CharacterCardID, &r.CharacterName,
			&responsesJSON, &r.PromptSetVersion,
			&submittedAt, &skippedAt, &r.DoorIntention,
		); err != nil {
			return nil, err
		}
		r.SubmittedAt = submittedAt
		r.SkippedAt = skippedAt

		switch {
		case submittedAt != nil:
			r.State = "submitted"
			r.Responses = map[string]string{}
			_ = json.Unmarshal(responsesJSON, &r.Responses)
		case skippedAt != nil:
			r.State = "skipped"
		default:
			r.State = "none"
		}
		out = append(out, r)
		userIDs = append(userIDs, r.UserID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// A draft is deliberately reported as "draft" WITHOUT its content. The
	// Player has not chosen to share it, and S8.2 is explicit that a draft
	// is not a submission -- a Director may see that someone is mid-thought,
	// never what they are mid-thinking.
	for i := range out {
		if out[i].State != "none" || out[i].CharacterCardID == "" {
			continue
		}
		var hasDraft bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM aftercare_response_drafts
			WHERE user_id = $1 AND character_card_id = $2 AND show_id = $3)
		`, out[i].UserID, out[i].CharacterCardID, showID).Scan(&hasDraft); err != nil {
			return nil, err
		}
		if hasDraft {
			out[i].State = "draft"
		}
	}

	for i := range out {
		count, err := ConsecutiveSkips(ctx, pool, out[i].UserID)
		if err != nil {
			return nil, err
		}
		out[i].ConsecutiveSkips = count
	}

	return out, nil
}

// PromptLabel resolves a stored response key to its authored question, for
// export headers and table columns.
func PromptLabel(key string) string {
	if p, ok := promptByKey(key); ok {
		return p.Label
	}
	return strings.TrimSpace(key)
}

package stageobjects

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5"
)

// recordAudit writes one actions row describing a canonical state change.
//
// Kernel 90 §23 step 7 asks that Cue execution "preserve audit/action
// evidence according to existing Cue/Action conventions"; doing it here
// rather than only in the Cue path means the manual Director control leaves
// the same evidence, which is the whole point of §24's parity requirement --
// a Showing Review that could tell "the Director hid it" from "a Cue hid it"
// by whether a row exists would be recording an implementation detail.
//
// Raw SQL rather than importing backend/internal/actions, following
// cues.insertCueActionRow, which writes its own rows the same way for the
// same reason: actions is a heavy package and this is one INSERT of a shape
// that is already established.
//
// Audit is best-effort and deliberately non-fatal. actions.session_id is NOT
// NULL, so a Show with no live session (a Director preparing between
// sessions) genuinely has nowhere to put the row -- and refusing the state
// change for want of an audit row would make canonical state unwritable
// exactly when a Director is setting up. The state change is the product
// behaviour; the row is evidence of it.
func recordAudit(ctx context.Context, tx pgx.Tx, showID, actorID, op string, ref Ref, resolved Resolved) {
	sessionID := activeSessionIDForShow(ctx, tx, showID)
	if sessionID == "" {
		return
	}

	payload := map[string]any{
		"operation":   op,
		"object_kind": ref.Kind,
		"object_id":   ref.ID,
	}
	// The label is evidence for a human reading a Showing Review ("hid the
	// wall sconce"), never identity -- §22 keeps identity to kind+id, and
	// nothing reads this field back.
	if label := strings.TrimSpace(resolved.Label); label != "" {
		payload["object_label"] = label
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return
	}

	_, _ = tx.Exec(ctx, `
		INSERT INTO actions (session_id, show_id, moment_id, actor_id, type, target, payload)
		VALUES (
			$1::uuid, $2::uuid,
			COALESCE((SELECT MAX(moment_id) FROM actions WHERE session_id = $1::uuid), 0) + 1,
			$3::uuid, $4, '{}'::jsonb, $5
		)
	`, sessionID, showID, actorID, "stage_object/"+op, payloadJSON)
}

func activeSessionIDForShow(ctx context.Context, q Querier, showID string) string {
	var sessionID string
	_ = q.QueryRow(ctx, `
		SELECT id::text FROM sessions
		WHERE show_id = $1::uuid AND status IN ('rehearsal', 'live')
		ORDER BY started_at DESC LIMIT 1
	`, showID).Scan(&sessionID)
	return sessionID
}

// InteractionInvocable is the invoke-time authority check for a bound
// participant interaction, and it is the reason a disabled interaction is
// genuinely disabled rather than merely un-offered.
//
// world/snapshot.go already stops a disabled interaction being ADVERTISED to
// a client. That is presentation. This function is what refuses a client that
// forges the call anyway -- §35's "Player cannot enable a Director-disabled
// interaction" and §53's PARTIAL trigger "disabled interaction still works".
//
// It answers only the Show-scoped half. Callers keep their existing
// participant_interactions.enabled check; the two are ANDed for the same
// reason applyInteractionState ANDs them (§20's documented bridge).
func InteractionInvocable(ctx context.Context, q Querier, showID, interactionID string) (bool, error) {
	showID = strings.TrimSpace(showID)
	interactionID = strings.TrimSpace(interactionID)
	if showID == "" || interactionID == "" {
		// No Show means no Show-scoped state can exist, so nothing here can
		// refuse the invocation. The caller's own global-enabled check still
		// applies.
		return true, nil
	}
	var enabled *bool
	err := q.QueryRow(ctx, `
		SELECT interaction_enabled
		FROM stage_object_states
		WHERE show_id = $1::uuid
		  AND object_kind = $2
		  AND object_id = $3::uuid
	`, showID, KindParticipantInteraction, interactionID).Scan(&enabled)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Absent state means authored default, which is enabled (§10).
			return true, nil
		}
		return false, err
	}
	if enabled == nil {
		return true, nil
	}
	return *enabled, nil
}

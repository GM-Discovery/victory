package messages

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Kernel 74 S8: the Directors+-only backstage note.
//
// Storage reuses the durable `messages` table rather than adding a table,
// so a Director who reconnects an hour later still sees the note for free
// (S8.2) -- messages is already the codebase's durable, per-recipient,
// rehydrating mailbox.
//
// It deliberately does NOT reuse HandleNoteCards, despite that being the
// nearest existing Player->Director note path. Two hard mismatches:
//
//  1. HandleNoteCards resolves its session through
//     identity.ResolveActiveCaveSessionID and stamps venue_slug "the-cave".
//     Kernel 74's note originates in Catharsis.
//  2. isNoteCardRecipientRole allows only director/cast/crew. S8.3 requires
//     Director, Producer, Operator, and authorized Stage Management Crew --
//     "cast" must NOT receive it, and Producer/Operator must.
//
// Bending note cards to fit would have widened a Player-facing endpoint's
// recipient rules; a sibling write path with its own recipient resolution
// keeps that boundary intact.
const MessageTypeBackstageNote = "backstage_note"

// backstageNoteRoles is the S8.3 recipient tier. Mirrors
// world.isBackstageRole (producer/director/operator/crew) rather than
// isNoteCardRecipientRole -- notably excluding "cast", who are performers,
// not backstage staff.
//
// Crew is included here at the session-role level; the crew tier's
// narrower "explicitly authorized Stage Management Crew" qualifier is
// applied by the caller through showruns authority, not re-derived here.
func backstageNoteRoles() []string {
	return []string{"producer", "director", "operator", "crew"}
}

// IsBackstageNoteRole reports whether a session role may read backstage
// notes. Exported so the read endpoint and the WebSocket push agree on one
// definition instead of two drifting copies.
func IsBackstageNoteRole(role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	for _, r := range backstageNoteRoles() {
		if r == role {
			return true
		}
	}
	return false
}

// BackstageNoteRecipients resolves every user in a session who should
// receive backstage notes. Returned as user IDs so the caller can both
// insert per-recipient rows and target hub.BroadcastToSessionUser at
// exactly those users -- the hub itself does no role filtering, so the
// filter must be applied when choosing who to push to, never by trusting
// a session-wide broadcast to be ignored by ineligible clients.
func BackstageNoteRecipients(ctx context.Context, pool *pgxpool.Pool, sessionID string) ([]string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT sp.user_id::text
		FROM session_participants sp
		WHERE sp.session_id = $1
		  AND LOWER(sp.role::text) = ANY($2::text[])
	`, sessionID, backstageNoteRoles())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// BackstageNoteInput is one note to fan out. SenderLabel is the authored
// display name for the note's origin ("Victory"), not the Player -- the
// Player's identity appears inside the authored Subject/Body the caller
// composes, and the note is a system observation rather than a message the
// Player chose to send.
type BackstageNoteInput struct {
	SessionID   string
	VenueSlug   string
	Subject     string
	Body        string
	SenderLabel string
}

// InsertBackstageNote writes one durable message row per Directors+
// recipient in the session and returns the recipient user IDs so the caller
// can push a targeted live update to each.
//
// Returning zero recipients is not an error: a Show can legitimately be
// running with no backstage user currently joined to the session, and the
// Player's own flow must not fail because nobody was listening. The
// intention itself is already durably recorded in
// participant_freeform_submissions regardless.
func InsertBackstageNote(ctx context.Context, pool *pgxpool.Pool, in BackstageNoteInput) ([]string, error) {
	in.Subject = strings.TrimSpace(in.Subject)
	in.Body = strings.TrimSpace(in.Body)
	if in.Subject == "" {
		in.Subject = "Backstage Note"
	}
	if in.Body == "" {
		return nil, errors.New("body_required")
	}
	senderLabel := strings.TrimSpace(in.SenderLabel)
	if senderLabel == "" {
		senderLabel = "Victory"
	}

	recipients, err := BackstageNoteRecipients(ctx, pool, in.SessionID)
	if err != nil {
		return nil, err
	}

	for _, userID := range recipients {
		if _, err := insertMessage(ctx, pool, messageInsert{
			MessageType: MessageTypeBackstageNote,
			ToUserID:    userID,
			SenderLabel: senderLabel,
			FromRole:    "system",
			Subject:     in.Subject,
			Body:        in.Body,
			VenueSlug:   strings.TrimSpace(in.VenueSlug),
			SessionID:   strings.TrimSpace(in.SessionID),
		}); err != nil {
			return nil, err
		}
	}
	return recipients, nil
}

// LoadBackstageNotesForSession returns the caller's own backstage notes for
// one session, newest last (chronological, matching how the Catharsis tray
// reads).
//
// The to_user_id = $2 predicate is the real boundary: even a Player who
// guesses the endpoint and a valid session_id gets only rows addressed to
// them, and no backstage note is ever addressed to a Player. The explicit
// role re-check in the handler is a second, independent gate rather than
// the only one.
func LoadBackstageNotesForSession(ctx context.Context, pool *pgxpool.Pool, sessionID, viewerUserID string) ([]Message, error) {
	rows, err := pool.Query(ctx, `
		SELECT
		  id::text, message_type, to_user_id::text, COALESCE(from_user_id::text, ''),
		  subject, body, COALESCE(venue_slug, ''), COALESCE(session_id::text, ''),
		  created_at, is_read
		FROM messages
		WHERE session_id = $1 AND to_user_id = $2 AND message_type = $3
		ORDER BY created_at ASC
	`, strings.TrimSpace(sessionID), strings.TrimSpace(viewerUserID), MessageTypeBackstageNote)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Message{}
	for rows.Next() {
		var m Message
		var createdAt time.Time
		if err := rows.Scan(
			&m.ID, &m.MessageType, &m.ToUserID, &m.FromUserID,
			&m.Subject, &m.Body, &m.VenueSlug, &m.SessionID, &createdAt, &m.Read,
		); err != nil {
			return nil, err
		}
		m.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		m.SenderLabel = "Victory"
		out = append(out, m)
	}
	return out, rows.Err()
}

// viewerSessionRole reads the caller's own role in the session. Returns ""
// when they are not a participant at all.
func viewerSessionRole(ctx context.Context, pool *pgxpool.Pool, sessionID, userID string) (string, error) {
	var role string
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(sp.role::text, '')
		FROM session_participants sp
		WHERE sp.session_id = $1 AND sp.user_id = $2
		LIMIT 1
	`, strings.TrimSpace(sessionID), strings.TrimSpace(userID)).Scan(&role)
	if err != nil {
		return "", nil
	}
	return role, nil
}

// HandleBackstageNotes handles GET /api/backstage-notes?session_id=...
//
// This is what lets the Director read the note inside Catharsis without
// opening Stage Management or the mailbox (S8.1). It is informational only:
// there is no GO, no cue, and no acknowledgement the Show waits on (S8.4).
func HandleBackstageNotes(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeForbiddenJSON(w, err)
			return
		}
		sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
		if sessionID == "" {
			writeErrorJSON(w, errors.New("session_id_required"))
			return
		}

		role, err := viewerSessionRole(ctx, pool, sessionID, userID)
		if err != nil {
			writeErrorJSON(w, err)
			return
		}
		if !IsBackstageNoteRole(role) {
			writeForbiddenJSON(w, errors.New("not_authorized"))
			return
		}

		notes, err := LoadBackstageNotesForSession(ctx, pool, sessionID, userID)
		if err != nil {
			writeErrorJSON(w, err)
			return
		}
		writeJSON(w, http.StatusOK, response{Ok: true, Data: map[string]any{"notes": notes}})
	}
}

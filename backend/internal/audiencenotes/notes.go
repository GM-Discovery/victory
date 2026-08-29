// Package audiencenotes implements Kernel 93 A19: durable, session-scoped
// notes an Audience member sends to the Show's Director from Catharsis.
// Delivery is always local -- a plain row in the shared `messages` mailbox
// (message_type = "audience_note"), the same durable table Kernel 11's note
// cards use -- so it never depends on the Discord Chat Bridge being
// connected. That is the main design, not a fallback: a Director's mailbox
// already shows every note the moment it's sent, bridge or no bridge.
//
// Kernel 11's own POST /api/note-cards route is Cave-anchored only
// (identity.ResolveActiveCaveSessionID) and hardcodes venue_slug="the-cave",
// so Catharsis Audience notes need this narrow sibling rather than widening
// that route -- same reasoning audienceadmission's own doc comment gives for
// not widening show_run_tickets.
//
// Exporting a session's accumulated notes to Discord as one batch is a
// separate, best-effort convenience: network.HandleExportAudienceNotesToDiscord.
package audiencenotes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

// MessageType is the messages.message_type value used for these notes, so
// the mailbox (and the Discord export batch query) can tell them apart from
// Kernel 11's own "note_card" rows.
const MessageType = "audience_note"

const maxBodyRunes = 250
const venueSlug = "catharsis"

// Note is the durable record returned to the sender after a successful
// submit -- deliberately thin, since the sender only needs to know it landed.
type Note struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

type submitRequest struct {
	Body string `json:"body"`
}

type response struct {
	Ok   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
}

// HandleSubmit is POST /api/session/catharsis/notes.
func HandleSubmit(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}
		senderUserID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil || strings.TrimSpace(senderUserID) == "" {
			writeJSON(w, http.StatusUnauthorized, response{Ok: false, Data: map[string]any{"error": "not_authenticated"}})
			return
		}

		var req submitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "invalid_json"}})
			return
		}

		note, err := Submit(ctx, pool, senderUserID, req.Body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": err.Error()}})
			return
		}

		writeJSON(w, http.StatusOK, response{Ok: true, Data: note})
	}
}

// Submit resolves senderUserID's currently live Catharsis session, confirms
// they are a genuine participant in it (not merely any authenticated
// account, matching Kernel 11's "senders must be ... currently anchored"
// rule applied to Catharsis instead of the Cave), resolves the session's
// earliest-seated director as recipient -- the exact fallback query
// messages.resolveNoteCardRecipient already established -- and inserts a
// durable mailbox row.
func Submit(ctx context.Context, pool *pgxpool.Pool, senderUserID, body string) (Note, error) {
	senderUserID = strings.TrimSpace(senderUserID)
	body = truncateRunes(strings.TrimSpace(body), maxBodyRunes)
	if senderUserID == "" {
		return Note{}, errors.New("not_authenticated")
	}
	if body == "" {
		return Note{}, errors.New("body_required")
	}

	var sessionID string
	if err := pool.QueryRow(ctx, `
		SELECT sess.id::text
		FROM sessions sess
		JOIN venues v ON v.id = sess.venue_id
		WHERE v.slug = $1 AND sess.status IN ('rehearsal', 'live')
		ORDER BY sess.started_at DESC LIMIT 1
	`, venueSlug).Scan(&sessionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Note{}, errors.New("no_active_showing")
		}
		return Note{}, err
	}

	var senderIsParticipant bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM session_participants
			WHERE session_id = $1 AND user_id = $2
		)
	`, sessionID, senderUserID).Scan(&senderIsParticipant); err != nil {
		return Note{}, err
	}
	if !senderIsParticipant {
		return Note{}, errors.New("not_session_participant")
	}

	var directorUserID string
	if err := pool.QueryRow(ctx, `
		SELECT sp.user_id::text
		FROM session_participants sp
		WHERE sp.session_id = $1
		  AND sp.role = 'director'
		ORDER BY sp.joined_at ASC
		LIMIT 1
	`, sessionID).Scan(&directorUserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Note{}, errors.New("no_director_available")
		}
		return Note{}, err
	}

	var note Note
	var createdAt time.Time
	if err := pool.QueryRow(ctx, `
		INSERT INTO messages (
			message_type, to_user_id, from_user_id, subject, body, venue_slug, session_id, is_read
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, FALSE)
		RETURNING id::text, session_id::text, body, created_at
	`, MessageType, directorUserID, senderUserID, "Audience Note", body, venueSlug, sessionID).Scan(
		&note.ID, &note.SessionID, &note.Body, &createdAt,
	); err != nil {
		return Note{}, err
	}
	note.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return note, nil
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(value) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[:max])
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

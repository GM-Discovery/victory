package network

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/identity"
)

// Kernel 93 A19: exports a Catharsis Showing's accumulated Audience notes
// (backend/internal/audiencenotes) into that venue's active Discord Chat
// Bridge thread as one batch message, if and only if the bridge happens to
// be connected right now. This sits entirely on top of audiencenotes' local,
// always-available mailbox delivery -- never a replacement for it, and never
// a requirement, matching the kernel doc's own "Chat Bridge failure is
// always a soft/warning-level result" convention (identity/chat_bridge.go).
//
// Deliberately no "already exported" bookkeeping: a Director triggers this
// by hand when they want a Discord copy of the batch, and re-exporting the
// same notes on a second click is a minor, visible nuisance (a Discord post
// says so itself), not worth a schema migration to prevent.

const audienceNotesExportMessageType = "audience_note"

type exportAudienceNotesResponse struct {
	Ok   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
}

// HandleExportAudienceNotesToDiscord is POST /api/session/catharsis/notes/export.
func HandleExportAudienceNotesToDiscord(pool *pgxpool.Pool, cfg identity.DiscordServerLinkConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, exportAudienceNotesResponse{Ok: false})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}
		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, exportAudienceNotesResponse{Ok: false, Data: map[string]any{"error": "not_authenticated"}})
			return
		}

		var sessionID string
		if err := pool.QueryRow(ctx, `
			SELECT sess.id::text
			FROM sessions sess
			JOIN venues v ON v.id = sess.venue_id
			WHERE v.slug = 'catharsis' AND sess.status IN ('rehearsal', 'live')
			ORDER BY sess.started_at DESC LIMIT 1
		`).Scan(&sessionID); err != nil {
			writeJSON(w, http.StatusBadRequest, exportAudienceNotesResponse{Ok: false, Data: map[string]any{"error": "no_active_showing"}})
			return
		}

		notes, err := loadAudienceNotesForExport(ctx, pool, userID, sessionID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, exportAudienceNotesResponse{Ok: false, Data: map[string]any{"error": "load_failed"}})
			return
		}
		if len(notes) == 0 {
			writeJSON(w, http.StatusOK, exportAudienceNotesResponse{Ok: true, Data: map[string]any{"exported": 0, "reason": "no_notes"}})
			return
		}

		venueSlug, locationID, err := loadChatBridgeVenueContext(ctx, pool, sessionID)
		if err != nil {
			writeJSON(w, http.StatusOK, exportAudienceNotesResponse{Ok: true, Data: map[string]any{"exported": 0, "reason": "chat_bridge_not_connected"}})
			return
		}
		thread, err := loadActiveDiscordSessionThread(ctx, pool, locationID, venueSlug)
		if err != nil || thread == nil {
			writeJSON(w, http.StatusOK, exportAudienceNotesResponse{Ok: true, Data: map[string]any{"exported": 0, "reason": "chat_bridge_not_connected"}})
			return
		}

		content := formatAudienceNotesBatch(notes)
		if _, err := postDiscordThreadMessage(ctx, cfg, thread.ThreadID, content); err != nil {
			writeJSON(w, http.StatusOK, exportAudienceNotesResponse{Ok: true, Data: map[string]any{"exported": 0, "reason": "discord_post_failed"}})
			return
		}

		writeJSON(w, http.StatusOK, exportAudienceNotesResponse{Ok: true, Data: map[string]any{"exported": len(notes)}})
	}
}

type audienceNoteExportRow struct {
	SenderLabel string
	Body        string
	CreatedAt   time.Time
}

func loadAudienceNotesForExport(ctx context.Context, pool *pgxpool.Pool, recipientUserID, sessionID string) ([]audienceNoteExportRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT
		  COALESCE(NULLIF(sender.display_name, ''), NULLIF(sender.handle, ''), 'Audience'),
		  m.body,
		  m.created_at
		FROM messages m
		LEFT JOIN users sender ON sender.id = m.from_user_id
		WHERE m.to_user_id = $1
		  AND m.session_id = $2
		  AND m.message_type = $3
		ORDER BY m.created_at ASC
	`, recipientUserID, sessionID, audienceNotesExportMessageType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []audienceNoteExportRow
	for rows.Next() {
		var row audienceNoteExportRow
		if err := rows.Scan(&row.SenderLabel, &row.Body, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func formatAudienceNotesBatch(notes []audienceNoteExportRow) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**Audience Notes (%d)**\n", len(notes))
	for i, note := range notes {
		fmt.Fprintf(&b, "%d. %s -- %s\n", i+1, note.SenderLabel, note.Body)
	}
	return truncateRunes(b.String(), discordChatBridgeMaxContentRunes)
}

package messages

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
	"victory/backend/internal/identity"
)

var errInvalidJSON = errors.New("invalid_json")
var errMessageNotFound = errors.New("message_not_found")
var errRecipientNotFound = errors.New("recipient_not_found")
var errNoCaveSession = errors.New("not_session_participant")

type Message struct {
	ID              string `json:"id"`
	MessageType     string `json:"message_type"`
	ToUserID        string `json:"to_user_id"`
	FromUserID      string `json:"from_user_id,omitempty"`
	FromHandle      string `json:"from_handle,omitempty"`
	FromDisplayName string `json:"from_display_name,omitempty"`
	FromRole        string `json:"from_role,omitempty"`
	SenderLabel     string `json:"sender_label"`
	Subject         string `json:"subject"`
	Body            string `json:"body"`
	VenueSlug       string `json:"venue_slug,omitempty"`
	SessionID       string `json:"session_id,omitempty"`
	CreatedAt       string `json:"created_at"`
	Read            bool   `json:"read"`
}

type response struct {
	Ok   bool        `json:"ok"`
	Data interface{} `json:"data,omitempty"`
}

type createMessageRequest struct {
	ToUserID string `json:"to_user_id"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	System   bool   `json:"system"`
}

type noteCardRequest struct {
	ToUserID        string `json:"to_user_id"`
	ToParticipantID string `json:"to_participant_id"`
	Subject         string `json:"subject"`
	Body            string `json:"body"`
	Context         struct {
		VenueSlug string `json:"venue_slug"`
		SessionID string `json:"session_id"`
	} `json:"context"`
}

func HandleMessages(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/messages" {
			writeJSON(w, http.StatusNotFound, response{Ok: false})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeForbiddenJSON(w, err)
			return
		}

		switch r.Method {
		case http.MethodGet:
			messages, err := loadMessages(ctx, pool, userID)
			if err != nil {
				writeErrorJSON(w, err)
				return
			}

			writeJSON(w, http.StatusOK, response{
				Ok: true,
				Data: map[string]any{
					"messages": messages,
				},
			})
		case http.MethodPost:
			if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
				writeErrorJSON(w, err)
				return
			} else if !ok {
				writeForbiddenJSON(w, errors.New("operator_access_required"))
				return
			}

			var req createMessageRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeErrorJSON(w, errInvalidJSON)
				return
			}

			req = sanitizeCreateMessageRequest(req)
			if req.ToUserID == "" || req.Subject == "" || req.Body == "" {
				writeErrorJSON(w, errors.New("to_user_id_subject_body_required"))
				return
			}

			row, err := createMessage(ctx, pool, userID, req)
			if err != nil {
				writeErrorJSON(w, err)
				return
			}

			writeJSON(w, http.StatusOK, response{Ok: true, Data: row})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}

func HandleNoteCards(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/note-cards" {
			writeJSON(w, http.StatusNotFound, response{Ok: false})
			return
		}

		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		senderUserID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeForbiddenJSON(w, err)
			return
		}

		activeSessionID, err := identity.ResolveActiveCaveSessionID(ctx, pool, senderUserID)
		if err != nil {
			writeForbiddenJSON(w, errNoCaveSession)
			return
		}

		senderIdentity, err := identity.ResolveSessionIdentity(ctx, pool, activeSessionID, senderUserID)
		if err != nil {
			writeForbiddenJSON(w, errNoCaveSession)
			return
		}

		var req noteCardRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorJSON(w, errInvalidJSON)
			return
		}

		req = sanitizeNoteCardRequest(req)
		if req.Body == "" {
			writeErrorJSON(w, errors.New("body_required"))
			return
		}
		if utf8.RuneCountInString(req.Body) > 250 {
			writeErrorJSON(w, errors.New("note_card_body_too_long"))
			return
		}

		recipientUserID, recipientLabel, recipientRole, err := resolveNoteCardRecipient(ctx, pool, activeSessionID, req)
		if err != nil {
			writeErrorJSON(w, err)
			return
		}

		subject := strings.TrimSpace(req.Subject)
		if subject == "" {
			subject = "Note Card"
		}

		row, err := insertMessage(ctx, pool, messageInsert{
			MessageType:     "note_card",
			ToUserID:        recipientUserID,
			FromUserID:      senderIdentity.UserID,
			FromHandle:      senderIdentity.Handle,
			FromDisplayName: senderIdentity.DisplayName,
			FromRole:        senderIdentity.Role,
			SenderLabel:     senderLabel(senderIdentity),
			Subject:         subject,
			Body:            req.Body,
			VenueSlug:       "the-cave",
			SessionID:       activeSessionID,
		})
		if err != nil {
			writeErrorJSON(w, err)
			return
		}

		row.ToUserID = recipientUserID
		row.MessageType = "note_card"
		row.VenueSlug = "the-cave"
		row.SessionID = activeSessionID
		row.FromUserID = senderIdentity.UserID
		row.FromHandle = senderIdentity.Handle
		row.FromDisplayName = senderIdentity.DisplayName
		row.FromRole = senderIdentity.Role
		row.SenderLabel = senderLabel(senderIdentity)
		_ = recipientLabel
		_ = recipientRole

		writeJSON(w, http.StatusOK, response{Ok: true, Data: row})
	}
}

func HandleMessageByID(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/messages/") {
			writeJSON(w, http.StatusNotFound, response{Ok: false})
			return
		}

		id := strings.TrimPrefix(r.URL.Path, "/api/messages/")
		id = strings.Trim(id, "/")
		if id == "" || strings.Contains(id, "/") {
			writeJSON(w, http.StatusNotFound, response{Ok: false})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeForbiddenJSON(w, err)
			return
		}

		switch r.Method {
		case http.MethodGet:
			row, err := getMessage(ctx, pool, userID, id)
			if err != nil {
				writeErrorJSON(w, err)
				return
			}

			writeJSON(w, http.StatusOK, response{Ok: true, Data: row})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}

type messageInsert struct {
	MessageType     string
	ToUserID        string
	FromUserID      string
	FromHandle      string
	FromDisplayName string
	FromRole        string
	SenderLabel     string
	Subject         string
	Body            string
	VenueSlug       string
	SessionID       string
}

func senderLabel(ident identity.SessionIdentity) string {
	label := strings.TrimSpace(ident.DisplayName)
	if label == "" {
		label = strings.TrimSpace(ident.Handle)
	}
	if label == "" {
		label = strings.TrimSpace(ident.UserID)
	}
	if label == "" {
		label = "Unknown Participant"
	}
	return label
}

func sanitizeNoteCardRequest(req noteCardRequest) noteCardRequest {
	req.ToUserID = strings.TrimSpace(req.ToUserID)
	req.ToParticipantID = strings.TrimSpace(req.ToParticipantID)
	req.Subject = truncateRunes(strings.TrimSpace(req.Subject), 120)
	req.Body = truncateRunes(strings.TrimSpace(req.Body), 250)
	req.Context.VenueSlug = strings.TrimSpace(req.Context.VenueSlug)
	req.Context.SessionID = strings.TrimSpace(req.Context.SessionID)
	return req
}

func resolveNoteCardRecipient(ctx context.Context, pool *pgxpool.Pool, sessionID string, req noteCardRequest) (string, string, string, error) {
	if req.ToParticipantID != "" {
		var recipientUserID, handle, displayName, role string
		err := pool.QueryRow(ctx, `
			SELECT
			  u.id::text,
			  COALESCE(NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			  COALESCE(NULLIF(u.display_name, ''), NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			  COALESCE(sp.role::text, 'audience')
			FROM session_participants sp
			JOIN users u ON u.id = sp.user_id
			WHERE sp.id = $1
			  AND sp.session_id = $2
			LIMIT 1
		`, req.ToParticipantID, sessionID).Scan(&recipientUserID, &handle, &displayName, &role)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return "", "", "", errors.New("recipient_not_session_participant")
			}
			return "", "", "", err
		}
		if !isNoteCardRecipientRole(role) {
			return "", "", "", errors.New("recipient_role_not_allowed")
		}
		return recipientUserID, displayNameOrHandle(displayName, handle, recipientUserID), role, nil
	}

	if req.ToUserID != "" {
		var recipientUserID, handle, displayName, role string
		err := pool.QueryRow(ctx, `
			SELECT
			  u.id::text,
			  COALESCE(NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			  COALESCE(NULLIF(u.display_name, ''), NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			  COALESCE(sp.role::text, 'audience')
			FROM session_participants sp
			JOIN users u ON u.id = sp.user_id
			WHERE sp.session_id = $1
			  AND sp.user_id = $2
			LIMIT 1
		`, sessionID, req.ToUserID).Scan(&recipientUserID, &handle, &displayName, &role)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return "", "", "", errors.New("recipient_not_session_participant")
			}
			return "", "", "", err
		}
		if !isNoteCardRecipientRole(role) {
			return "", "", "", errors.New("recipient_role_not_allowed")
		}
		return recipientUserID, displayNameOrHandle(displayName, handle, recipientUserID), role, nil
	}

	var recipientUserID, handle, displayName, role string
	err := pool.QueryRow(ctx, `
		SELECT
		  u.id::text,
		  COALESCE(NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
		  COALESCE(NULLIF(u.display_name, ''), NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
		  COALESCE(sp.role::text, 'audience')
		FROM session_participants sp
		JOIN users u ON u.id = sp.user_id
		WHERE sp.session_id = $1
		  AND sp.role = 'director'
		ORDER BY
		  sp.joined_at ASC
		LIMIT 1
	`, sessionID).Scan(&recipientUserID, &handle, &displayName, &role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", errors.New("no_director_available")
		}
		return "", "", "", err
	}
	if !isNoteCardRecipientRole(role) {
		return "", "", "", errors.New("recipient_role_not_allowed")
	}

	return recipientUserID, displayNameOrHandle(displayName, handle, recipientUserID), role, nil
}

func isNoteCardRecipientRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "director", "cast", "crew":
		return true
	default:
		return false
	}
}

func displayNameOrHandle(displayName, handle, userID string) string {
	displayName = strings.TrimSpace(displayName)
	if displayName != "" {
		return displayName
	}
	handle = strings.TrimSpace(handle)
	if handle != "" {
		return handle
	}
	userID = strings.TrimSpace(userID)
	if len(userID) > 8 {
		return userID[:8]
	}
	if userID != "" {
		return userID
	}
	return "Unknown Participant"
}

func insertMessage(ctx context.Context, pool *pgxpool.Pool, input messageInsert) (Message, error) {
	input.MessageType = strings.TrimSpace(input.MessageType)
	if input.MessageType == "" {
		input.MessageType = "message"
	}
	input.ToUserID = strings.TrimSpace(input.ToUserID)
	input.FromUserID = strings.TrimSpace(input.FromUserID)
	input.FromHandle = strings.TrimSpace(input.FromHandle)
	input.FromDisplayName = strings.TrimSpace(input.FromDisplayName)
	input.FromRole = strings.TrimSpace(input.FromRole)
	input.SenderLabel = strings.TrimSpace(input.SenderLabel)
	input.Subject = strings.TrimSpace(input.Subject)
	input.Body = strings.TrimSpace(input.Body)
	input.VenueSlug = strings.TrimSpace(input.VenueSlug)
	input.SessionID = strings.TrimSpace(input.SessionID)

	var fromUserArg any
	if input.FromUserID != "" {
		fromUserArg = input.FromUserID
	}
	var sessionIDArg any
	if input.SessionID != "" {
		sessionIDArg = input.SessionID
	}

	var row Message
	var createdAt time.Time
	err := pool.QueryRow(ctx, `
		INSERT INTO messages (
		  message_type,
		  to_user_id,
		  from_user_id,
		  subject,
		  body,
		  venue_slug,
		  session_id,
		  is_read
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, FALSE)
		RETURNING
		  id::text,
		  message_type,
		  to_user_id::text,
		  COALESCE(from_user_id::text, ''),
		  COALESCE(NULLIF(subject, ''), ''),
		  COALESCE(NULLIF(body, ''), ''),
		  COALESCE(NULLIF(venue_slug, ''), ''),
		  COALESCE(session_id::text, ''),
		  created_at,
		  is_read
	`, input.MessageType, input.ToUserID, fromUserArg, input.Subject, input.Body, input.VenueSlug, sessionIDArg).Scan(
		&row.ID,
		&row.MessageType,
		&row.ToUserID,
		&row.FromUserID,
		&row.Subject,
		&row.Body,
		&row.VenueSlug,
		&row.SessionID,
		&createdAt,
		&row.Read,
	)
	if err != nil {
		return Message{}, err
	}

	row.FromHandle = input.FromHandle
	row.FromDisplayName = input.FromDisplayName
	row.FromRole = input.FromRole
	row.SenderLabel = input.SenderLabel
	row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return finalizeMessage(row), nil
}

func requireAuthenticatedUser(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (string, error) {
	sessionCookie := ""
	if c, err := r.Cookie("victory_session"); err == nil {
		sessionCookie = c.Value
	}

	userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
	if err != nil || strings.TrimSpace(userID) == "" {
		return "", errors.New("not_authenticated")
	}

	return userID, nil
}

func loadMessages(ctx context.Context, pool *pgxpool.Pool, userID string) ([]Message, error) {
	rows, err := pool.Query(ctx, `
		SELECT
		  m.id::text,
		  COALESCE(NULLIF(m.message_type, ''), 'message'),
		  m.to_user_id::text,
		  COALESCE(m.from_user_id::text, ''),
		  COALESCE(NULLIF(sender.handle, ''), ''),
		  COALESCE(NULLIF(sender.display_name, ''), ''),
		  COALESCE(sender_role.role, ''),
		  CASE
			WHEN m.from_user_id IS NULL THEN 'System'
			ELSE COALESCE(NULLIF(sender.display_name, ''), NULLIF(sender.handle, ''), LEFT(sender.id::text, 8), 'System')
		  END AS sender_label,
		  COALESCE(m.subject, ''),
		  COALESCE(m.body, ''),
		  COALESCE(NULLIF(m.venue_slug, ''), ''),
		  COALESCE(m.session_id::text, ''),
		  m.created_at,
		  COALESCE(m.is_read, FALSE)
		FROM messages m
		LEFT JOIN users sender ON sender.id = m.from_user_id
		LEFT JOIN LATERAL (
		  SELECT lm.role::text AS role
		  FROM location_memberships lm
		  WHERE lm.user_id = sender.id
		    AND lm.active = TRUE
		  ORDER BY
			CASE lm.role
			  WHEN 'producer' THEN 1
			  WHEN 'director' THEN 2
			  WHEN 'cast' THEN 3
			  WHEN 'crew' THEN 4
			  WHEN 'audience' THEN 5
			  ELSE 99
			END,
			lm.created_at ASC
		  LIMIT 1
		) sender_role ON TRUE
		WHERE m.to_user_id = $1
		ORDER BY m.created_at DESC, m.id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var msg Message
		var createdAt time.Time
		if err := rows.Scan(
			&msg.ID,
			&msg.MessageType,
			&msg.ToUserID,
			&msg.FromUserID,
			&msg.FromHandle,
			&msg.FromDisplayName,
			&msg.FromRole,
			&msg.SenderLabel,
			&msg.Subject,
			&msg.Body,
			&msg.VenueSlug,
			&msg.SessionID,
			&createdAt,
			&msg.Read,
		); err != nil {
			return nil, err
		}
		msg.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		msg = finalizeMessage(msg)
		out = append(out, msg)
	}

	return out, rows.Err()
}

func getMessage(ctx context.Context, pool *pgxpool.Pool, userID, messageID string) (Message, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer tx.Rollback(ctx)

	msg, err := loadMessageByID(ctx, tx, userID, messageID)
	if err != nil {
		return Message{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE messages
		SET is_read = TRUE
		WHERE id = $1
		  AND to_user_id = $2
	`, messageID, userID); err != nil {
		return Message{}, err
	}

	msg.Read = true
	if err := tx.Commit(ctx); err != nil {
		return Message{}, err
	}

	return finalizeMessage(msg), nil
}

func loadMessageByID(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID, messageID string) (Message, error) {
	var msg Message
	var createdAt time.Time

	err := q.QueryRow(ctx, `
		SELECT
		  m.id::text,
		  COALESCE(NULLIF(m.message_type, ''), 'message'),
		  m.to_user_id::text,
		  COALESCE(m.from_user_id::text, ''),
		  COALESCE(NULLIF(sender.handle, ''), ''),
		  COALESCE(NULLIF(sender.display_name, ''), ''),
		  COALESCE(sender_role.role, ''),
		  CASE
			WHEN m.from_user_id IS NULL THEN 'System'
			ELSE COALESCE(NULLIF(sender.display_name, ''), NULLIF(sender.handle, ''), LEFT(sender.id::text, 8), 'System')
		  END AS sender_label,
		  COALESCE(m.subject, ''),
		  COALESCE(m.body, ''),
		  COALESCE(NULLIF(m.venue_slug, ''), ''),
		  COALESCE(m.session_id::text, ''),
		  m.created_at,
		  COALESCE(m.is_read, FALSE)
		FROM messages m
		LEFT JOIN users sender ON sender.id = m.from_user_id
		LEFT JOIN LATERAL (
		  SELECT lm.role::text AS role
		  FROM location_memberships lm
		  WHERE lm.user_id = sender.id
		    AND lm.active = TRUE
		  ORDER BY
			CASE lm.role
			  WHEN 'producer' THEN 1
			  WHEN 'director' THEN 2
			  WHEN 'cast' THEN 3
			  WHEN 'crew' THEN 4
			  WHEN 'audience' THEN 5
			  ELSE 99
			END,
			lm.created_at ASC
		  LIMIT 1
		) sender_role ON TRUE
		WHERE m.id = $1
		  AND m.to_user_id = $2
		LIMIT 1
	`, messageID, userID).Scan(
		&msg.ID,
		&msg.MessageType,
		&msg.ToUserID,
		&msg.FromUserID,
		&msg.FromHandle,
		&msg.FromDisplayName,
		&msg.FromRole,
		&msg.SenderLabel,
		&msg.Subject,
		&msg.Body,
		&msg.VenueSlug,
		&msg.SessionID,
		&createdAt,
		&msg.Read,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Message{}, errMessageNotFound
		}
		return Message{}, err
	}

	msg.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return msg, nil
}

func createMessage(ctx context.Context, pool *pgxpool.Pool, operatorUserID string, req createMessageRequest) (Message, error) {
	var recipientExists bool
	if err := pool.QueryRow(ctx, `
		SELECT TRUE
		FROM users
		WHERE id = $1
	`, req.ToUserID).Scan(&recipientExists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Message{}, errRecipientNotFound
		}
		return Message{}, err
	}

	fromUserID := ""
	fromUserLabel := "System"
	var fromDisplayName string
	var fromHandle string
	if !req.System {
		fromUserID = strings.TrimSpace(operatorUserID)
		if fromUserID != "" {
			if err := pool.QueryRow(ctx, `
				SELECT
				  COALESCE(NULLIF(display_name, ''), ''),
				  COALESCE(NULLIF(handle, ''), '')
				FROM users
				WHERE id = $1
				LIMIT 1
			`, fromUserID).Scan(&fromDisplayName, &fromHandle); err == nil {
				fromUserLabel = strings.TrimSpace(fromDisplayName)
				if fromUserLabel == "" {
					fromUserLabel = strings.TrimSpace(fromHandle)
				}
			}
			if fromUserLabel == "" {
				fromUserLabel = "Operator"
			}
		}
	}

	var fromUserArg any
	if fromUserID != "" {
		fromUserArg = fromUserID
	}

	var row Message
	var createdAt time.Time
	err := pool.QueryRow(ctx, `
		INSERT INTO messages (
		  to_user_id,
		  from_user_id,
		  subject,
		  body,
		  is_read
		)
		VALUES ($1, $2, $3, $4, FALSE)
		RETURNING
		  id::text,
		  to_user_id::text,
		  COALESCE(from_user_id::text, ''),
		  COALESCE(NULLIF(subject, ''), ''),
		  COALESCE(NULLIF(body, ''), ''),
		  created_at,
		  is_read
	`, req.ToUserID, fromUserArg, req.Subject, req.Body).Scan(
		&row.ID,
		&row.ToUserID,
		&row.FromUserID,
		&row.Subject,
		&row.Body,
		&createdAt,
		&row.Read,
	)
	if err != nil {
		return Message{}, err
	}

	row.MessageType = "message"
	row.VenueSlug = ""
	row.SessionID = ""
	row.FromUserID = fromUserID
	row.FromHandle = fromHandle
	row.FromDisplayName = fromDisplayName
	row.SenderLabel = fromUserLabel
	row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return finalizeMessage(row), nil
}

func sanitizeCreateMessageRequest(req createMessageRequest) createMessageRequest {
	req.ToUserID = strings.TrimSpace(req.ToUserID)
	req.Subject = truncateRunes(strings.TrimSpace(req.Subject), 120)
	req.Body = truncateRunes(strings.TrimSpace(req.Body), 250)
	return req
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= max {
		return value
	}

	runes := []rune(value)
	if len(runes) <= max {
		return value
	}

	return string(runes[:max])
}

func finalizeMessage(msg Message) Message {
	msg.SenderLabel = strings.TrimSpace(msg.SenderLabel)
	if msg.SenderLabel == "" {
		if strings.TrimSpace(msg.FromUserID) == "" {
			msg.SenderLabel = "System"
		} else {
			msg.SenderLabel = "Sender"
		}
	}
	if strings.TrimSpace(msg.Subject) == "" {
		msg.Subject = "No subject"
	}
	if strings.TrimSpace(msg.Body) == "" {
		msg.Body = "No body"
	}
	return msg
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErrorJSON(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeJSON(w, http.StatusNotFound, response{
			Ok: false,
			Data: map[string]any{
				"error": "not_found",
			},
		})
	case errors.Is(err, errMessageNotFound), errors.Is(err, errRecipientNotFound):
		writeJSON(w, http.StatusNotFound, response{
			Ok: false,
			Data: map[string]any{
				"error": err.Error(),
			},
		})
	case errors.Is(err, errInvalidJSON):
		writeJSON(w, http.StatusBadRequest, response{
			Ok: false,
			Data: map[string]any{
				"error": "invalid_json",
			},
		})
	default:
		writeJSON(w, http.StatusBadRequest, response{
			Ok: false,
			Data: map[string]any{
				"error": err.Error(),
			},
		})
	}
}

func writeForbiddenJSON(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusForbidden, response{
		Ok: false,
		Data: map[string]any{
			"error": err.Error(),
		},
	})
}

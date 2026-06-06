package network

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/showings"
)

type sessionControlRequest struct {
	VenueSlug string `json:"venue_slug"`
	Command   string `json:"command"`
}

type sessionControlSessionResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at,omitempty"`
}

type sessionControlShowingResponse struct {
	ID                  string `json:"id"`
	SessionID           string `json:"session_id"`
	Status              string `json:"status"`
	AudienceViewEnabled bool   `json:"audience_view_enabled"`
	StartedAt           string `json:"started_at"`
	EndedAt             string `json:"ended_at,omitempty"`
	CreatedBy           string `json:"created_by"`
}

type sessionControlResponse struct {
	VenueSlug string                         `json:"venue_slug"`
	Command   string                         `json:"command"`
	State     string                         `json:"state"`
	Message   string                         `json:"message"`
	Session   *sessionControlSessionResponse `json:"session,omitempty"`
	Showing   *sessionControlShowingResponse `json:"showing,omitempty"`
}

type sessionControlRow struct {
	Session sessionControlSessionResponse
	Showing sessionControlShowingResponse
	Found   bool
}

func HandleSessionControl(hub *Hub, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		userID, err := currentUserIDFromRequest(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		allowed, err := canAccessDirectorConsole(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		var req sessionControlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}

		venueSlug := normalizeSessionControlVenueSlug(req.VenueSlug)
		if venueSlug == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "venue_not_session_enabled"})
			return
		}

		command := normalizeSessionControlCommand(req.Command)
		if command == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "session_command_required"})
			return
		}

		role := sessionControlParticipantRole(ctx, pool, userID)

		switch command {
		case "status":
			row, err := loadSessionControlRow(ctx, pool, venueSlug)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "session_lookup_failed"})
				return
			}
			resp := sessionControlResponse{
				VenueSlug: venueSlug,
				Command:   command,
				State:     sessionControlState(row),
				Message:   sessionControlMessage(row),
			}
			if row.Found {
				resp.Session = &row.Session
				resp.Showing = &row.Showing
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": resp})
			return

		case "start":
			row, err := startSessionControl(ctx, pool, venueSlug, userID, role)
			if err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]any{
					"ok":     false,
					"error":  "session_start_failed",
					"detail": err.Error(),
				})
				return
			}
			broadcastSessionControlShowing(hub, row)
			resp := sessionControlResponse{
				VenueSlug: venueSlug,
				Command:   command,
				State:     sessionControlState(row),
				Message:   sessionControlMessage(row),
				Session:   &row.Session,
				Showing:   &row.Showing,
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": resp})
			return

		case "end":
			row, err := endSessionControl(ctx, pool, venueSlug)
			if err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]any{
					"ok":     false,
					"error":  "session_end_failed",
					"detail": err.Error(),
				})
				return
			}
			if row.Found {
				broadcastSessionControlShowing(hub, row)
			}
			resp := sessionControlResponse{
				VenueSlug: venueSlug,
				Command:   command,
				State:     sessionControlState(row),
				Message:   sessionControlMessage(row),
			}
			if row.Found {
				resp.Session = &row.Session
				resp.Showing = &row.Showing
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": resp})
			return
		default:
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "unsupported_session_command"})
			return
		}
	}
}

func normalizeSessionControlVenueSlug(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "the-cave", "first-theater", "middle-school-stage":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

func normalizeSessionControlCommand(raw string) string {
	command := strings.ToLower(strings.TrimSpace(raw))
	switch command {
	case "", "status", "start", "end":
		return commandOrDefault(command)
	default:
		return ""
	}
}

func commandOrDefault(command string) string {
	if strings.TrimSpace(command) == "" {
		return "status"
	}
	return command
}

func sessionControlParticipantRole(ctx context.Context, pool *pgxpool.Pool, userID string) string {
	role, err := access.CurrentLocationRole(ctx, pool, userID)
	if err != nil {
		return "producer"
	}
	if strings.EqualFold(strings.TrimSpace(role), "audience") {
		if ok, err := access.IsOperatorUser(ctx, pool, userID); err == nil && ok {
			return "producer"
		}
	}
	return role
}

func loadSessionControlRow(ctx context.Context, q sessionControlQuerier, venueSlug string) (sessionControlRow, error) {
	var row sessionControlRow
	if err := q.QueryRow(ctx, `
		SELECT
			s.id::text,
			s.status::text,
			s.started_at::text,
			COALESCE(s.ended_at::text, ''),
			COALESCE(sh.id::text, ''),
			COALESCE(sh.session_id::text, ''),
			COALESCE(sh.status::text, ''),
			COALESCE(sh.audience_view_enabled, FALSE),
			COALESCE(sh.started_at::text, ''),
			COALESCE(sh.ended_at::text, ''),
			COALESCE(sh.created_by::text, '')
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		LEFT JOIN showings sh ON sh.session_id = s.id
		WHERE v.slug = $1
		  AND s.status IN ('rehearsal', 'live')
		ORDER BY s.started_at DESC
		LIMIT 1
	`, venueSlug).Scan(
		&row.Session.ID,
		&row.Session.Status,
		&row.Session.StartedAt,
		&row.Session.EndedAt,
		&row.Showing.ID,
		&row.Showing.SessionID,
		&row.Showing.Status,
		&row.Showing.AudienceViewEnabled,
		&row.Showing.StartedAt,
		&row.Showing.EndedAt,
		&row.Showing.CreatedBy,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sessionControlRow{}, nil
		}
		return sessionControlRow{}, err
	}

	row.Found = strings.TrimSpace(row.Session.ID) != ""
	if !row.Found {
		return sessionControlRow{}, nil
	}
	if strings.TrimSpace(row.Showing.ID) == "" {
		row.Showing = sessionControlShowingResponse{}
	}
	return row, nil
}

func startSessionControl(ctx context.Context, pool *pgxpool.Pool, venueSlug, userID, role string) (sessionControlRow, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return sessionControlRow{}, err
	}
	defer tx.Rollback(ctx)

	venueID, err := loadVenueIDBySlug(ctx, tx, venueSlug)
	if err != nil {
		return sessionControlRow{}, err
	}

	row, err := loadSessionControlRow(ctx, tx, venueSlug)
	if err != nil {
		return sessionControlRow{}, err
	}

	if !row.Found {
		var sessionID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO sessions (
				venue_id,
				status,
				started_at,
				ended_at
			)
			VALUES ($1, 'rehearsal', NOW(), NULL)
			RETURNING id::text
		`, venueID).Scan(&sessionID); err != nil {
			return sessionControlRow{}, err
		}

		if err := ensureSessionParticipant(ctx, tx, sessionID, userID, role); err != nil {
			return sessionControlRow{}, err
		}

		showing, err := showings.EnsureForSession(ctx, tx, sessionID, userID)
		if err != nil {
			return sessionControlRow{}, err
		}

		row = sessionControlRow{
			Found: true,
			Session: sessionControlSessionResponse{
				ID:        sessionID,
				Status:    "rehearsal",
				StartedAt: time.Now().UTC().Format(time.RFC3339),
			},
			Showing: sessionControlShowingResponse{
				ID:                  showing.ID,
				SessionID:           showing.SessionID,
				Status:              showing.Status,
				AudienceViewEnabled: showing.AudienceViewEnabled,
				StartedAt:           showing.StartedAt,
				EndedAt:             showing.EndedAt,
				CreatedBy:           showing.CreatedBy,
			},
		}
	} else {
		if err := ensureSessionParticipant(ctx, tx, row.Session.ID, userID, role); err != nil {
			return sessionControlRow{}, err
		}

		if _, err := tx.Exec(ctx, `
			UPDATE sessions
			SET ended_at = NULL
			WHERE id = $1::uuid
		`, row.Session.ID); err != nil {
			return sessionControlRow{}, err
		}

		if strings.TrimSpace(row.Showing.ID) == "" {
			showing, err := showings.EnsureForSession(ctx, tx, row.Session.ID, userID)
			if err != nil {
				return sessionControlRow{}, err
			}
			row.Showing = sessionControlShowingResponse{
				ID:                  showing.ID,
				SessionID:           showing.SessionID,
				Status:              showing.Status,
				AudienceViewEnabled: showing.AudienceViewEnabled,
				StartedAt:           showing.StartedAt,
				EndedAt:             showing.EndedAt,
				CreatedBy:           showing.CreatedBy,
			}
		} else if !strings.EqualFold(strings.TrimSpace(row.Showing.Status), strings.TrimSpace(row.Session.Status)) || strings.EqualFold(strings.TrimSpace(row.Showing.Status), "closed") {
			if err := tx.QueryRow(ctx, `
				UPDATE showings
				SET status = $2::session_status,
				    audience_view_enabled = ($2::session_status = 'live'),
				    ended_at = NULL
				WHERE id = $1::uuid
				RETURNING
					id::text,
					session_id::text,
					status::text,
					audience_view_enabled,
					started_at::text,
					COALESCE(ended_at::text, ''),
					created_by::text
			`, row.Showing.ID, row.Session.Status).Scan(
				&row.Showing.ID,
				&row.Showing.SessionID,
				&row.Showing.Status,
				&row.Showing.AudienceViewEnabled,
				&row.Showing.StartedAt,
				&row.Showing.EndedAt,
				&row.Showing.CreatedBy,
			); err != nil {
				return sessionControlRow{}, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return sessionControlRow{}, err
	}

	return row, nil
}

func endSessionControl(ctx context.Context, pool *pgxpool.Pool, venueSlug string) (sessionControlRow, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return sessionControlRow{}, err
	}
	defer tx.Rollback(ctx)

	row, err := loadSessionControlRow(ctx, tx, venueSlug)
	if err != nil {
		return sessionControlRow{}, err
	}
	if !row.Found {
		return row, nil
	}

	if strings.TrimSpace(row.Showing.ID) != "" && !strings.EqualFold(strings.TrimSpace(row.Showing.Status), "closed") {
		closed, err := showings.CloseBySession(ctx, tx, row.Session.ID)
		if err != nil {
			return sessionControlRow{}, err
		}
		row.Showing = sessionControlShowingResponse{
			ID:                  closed.ID,
			SessionID:           closed.SessionID,
			Status:              closed.Status,
			AudienceViewEnabled: closed.AudienceViewEnabled,
			StartedAt:           closed.StartedAt,
			EndedAt:             closed.EndedAt,
			CreatedBy:           closed.CreatedBy,
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE sessions
		SET status = 'closed',
		    ended_at = COALESCE(ended_at, NOW())
		WHERE id = $1::uuid
	`, row.Session.ID); err != nil {
		return sessionControlRow{}, err
	}

	row.Session.Status = "closed"
	if row.Session.EndedAt == "" {
		row.Session.EndedAt = time.Now().UTC().Format(time.RFC3339)
	}

	if err := tx.Commit(ctx); err != nil {
		return sessionControlRow{}, err
	}

	return row, nil
}

func ensureSessionParticipant(ctx context.Context, q sessionControlExecQuerier, sessionID, userID, role string) error {
	role = strings.TrimSpace(role)
	if role == "" {
		role = "producer"
	}

	_, err := q.Exec(ctx, `
		INSERT INTO session_participants (session_id, user_id, role)
		VALUES ($1::uuid, $2::uuid, $3::location_role)
		ON CONFLICT (session_id, user_id) DO UPDATE
		SET role = EXCLUDED.role,
		    left_at = NULL
	`, sessionID, userID, role)
	return err
}

func loadVenueIDBySlug(ctx context.Context, q sessionControlQuerier, venueSlug string) (string, error) {
	var venueID string
	if err := q.QueryRow(ctx, `
		SELECT id::text
		FROM venues
		WHERE slug = $1
		LIMIT 1
	`, venueSlug).Scan(&venueID); err != nil {
		return "", err
	}
	return venueID, nil
}

func sessionControlState(row sessionControlRow) string {
	if !row.Found || strings.TrimSpace(row.Showing.ID) == "" {
		return "closed"
	}

	switch strings.ToLower(strings.TrimSpace(row.Showing.Status)) {
	case "rehearsal":
		return "ready"
	case "live":
		return "active"
	default:
		return "closed"
	}
}

func sessionControlMessage(row sessionControlRow) string {
	switch sessionControlState(row) {
	case "ready":
		return "Session ready. Use /mic on when you're ready."
	case "active":
		return "Session active. Showing open."
	default:
		return "Session closed. Use /session start to open it."
	}
}

func broadcastSessionControlShowing(hub *Hub, row sessionControlRow) {
	if hub == nil || !row.Found || strings.TrimSpace(row.Showing.ID) == "" {
		return
	}

	msgOut, _ := json.Marshal(map[string]any{
		"type":    "showing/update",
		"showing": row.Showing,
	})
	hub.Broadcast(msgOut)
}

type sessionControlQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type sessionControlExecQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

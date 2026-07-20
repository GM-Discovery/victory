package showings

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Showing struct {
	ID                  string `json:"id"`
	ProductionID        string `json:"production_id"`
	VenueID             string `json:"venue_id"`
	RunID               string `json:"run_id,omitempty"`
	Status              string `json:"status"`
	AudienceViewEnabled bool   `json:"audience_view_enabled"`
	StartedAt           string `json:"started_at"`
	EndedAt             string `json:"ended_at,omitempty"`
	CreatedBy           string `json:"created_by"`
	SessionID           string `json:"session_id"`
}

type showingQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func EnsureForSession(ctx context.Context, q showingQuerier, sessionID, createdBy string) (Showing, error) {
	sessionID = strings.TrimSpace(sessionID)
	createdBy = strings.TrimSpace(createdBy)
	if sessionID == "" {
		return Showing{}, errors.New("session_id is required")
	}

	if showing, err := LoadBySession(ctx, q, sessionID); err == nil {
		if err := backfillShowingActions(ctx, q, sessionID, showing.ID); err != nil {
			return Showing{}, err
		}
		return showing, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Showing{}, err
	}

	if createdBy == "" {
		err := q.QueryRow(ctx, `
			SELECT sp.user_id::text
			FROM session_participants sp
			WHERE sp.session_id = $1::uuid
			ORDER BY sp.joined_at ASC
			LIMIT 1
		`, sessionID).Scan(&createdBy)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Showing{}, errors.New("not_session_participant")
			}
			return Showing{}, err
		}
	}

	var (
		status       string
		venueID      string
		productionID string
		startedAt    time.Time
	)
	if err := q.QueryRow(ctx, `
		SELECT
			s.status::text,
			v.id::text,
			COALESCE((
				SELECT p.id::text
				FROM productions p
				WHERE p.location_id = l.id
				ORDER BY p.created_at ASC
				LIMIT 1
			), ''),
			s.started_at
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		WHERE s.id = $1::uuid
		LIMIT 1
	`, sessionID).Scan(&status, &venueID, &productionID, &startedAt); err != nil {
		return Showing{}, err
	}
	if strings.TrimSpace(productionID) == "" {
		return Showing{}, errors.New("production_required")
	}

	audienceViewEnabled := strings.EqualFold(status, "live")
	endedAt := ""
	if strings.EqualFold(status, "closed") {
		endedAt = time.Now().UTC().Format(time.RFC3339)
	}

	var showing Showing
	if err := q.QueryRow(ctx, `
		INSERT INTO showings (
			session_id,
			production_id,
			venue_id,
			run_id,
			status,
			audience_view_enabled,
			started_at,
			ended_at,
			created_by
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			$3::uuid,
			NULL,
			$4::session_status,
			$5,
			$6,
			NULLIF($7, '')::timestamptz,
			$8::uuid
		)
		RETURNING
			id::text,
			production_id::text,
			venue_id::text,
			COALESCE(run_id::text, ''),
			status::text,
			audience_view_enabled,
			started_at::text,
			COALESCE(ended_at::text, ''),
			created_by::text,
			session_id::text
	`, sessionID, productionID, venueID, status, audienceViewEnabled, startedAt, endedAt, createdBy).
		Scan(&showing.ID, &showing.ProductionID, &showing.VenueID, &showing.RunID, &showing.Status, &showing.AudienceViewEnabled, &showing.StartedAt, &showing.EndedAt, &showing.CreatedBy, &showing.SessionID); err != nil {
		return Showing{}, err
	}

	if err := backfillShowingActions(ctx, q, sessionID, showing.ID); err != nil {
		return Showing{}, err
	}

	return showing, nil
}

func LoadBySession(ctx context.Context, q showingQuerier, sessionID string) (Showing, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return Showing{}, errors.New("session_id is required")
	}

	var showing Showing
	if err := q.QueryRow(ctx, `
		SELECT
			id::text,
			production_id::text,
			venue_id::text,
			COALESCE(run_id::text, ''),
			status::text,
			audience_view_enabled,
			started_at::text,
			COALESCE(ended_at::text, ''),
			created_by::text,
			session_id::text
		FROM showings
		WHERE session_id = $1::uuid
		LIMIT 1
	`, sessionID).Scan(&showing.ID, &showing.ProductionID, &showing.VenueID, &showing.RunID, &showing.Status, &showing.AudienceViewEnabled, &showing.StartedAt, &showing.EndedAt, &showing.CreatedBy, &showing.SessionID); err != nil {
		return Showing{}, err
	}

	return showing, nil
}

func LoadByID(ctx context.Context, q showingQuerier, showingID string) (Showing, error) {
	showingID = strings.TrimSpace(showingID)
	if showingID == "" {
		return Showing{}, errors.New("showing_id is required")
	}

	var showing Showing
	if err := q.QueryRow(ctx, `
		SELECT
			id::text,
			production_id::text,
			venue_id::text,
			COALESCE(run_id::text, ''),
			status::text,
			audience_view_enabled,
			started_at::text,
			COALESCE(ended_at::text, ''),
			created_by::text,
			session_id::text
		FROM showings
		WHERE id = $1::uuid
		LIMIT 1
	`, showingID).Scan(&showing.ID, &showing.ProductionID, &showing.VenueID, &showing.RunID, &showing.Status, &showing.AudienceViewEnabled, &showing.StartedAt, &showing.EndedAt, &showing.CreatedBy, &showing.SessionID); err != nil {
		return Showing{}, err
	}

	return showing, nil
}

func CloseBySession(ctx context.Context, q showingQuerier, sessionID string) (Showing, error) {
	showing, err := LoadBySession(ctx, q, sessionID)
	if err != nil {
		return Showing{}, err
	}

	var closed Showing
	if err := q.QueryRow(ctx, `
		UPDATE showings
		SET status = 'closed',
		    audience_view_enabled = FALSE,
		    ended_at = COALESCE(ended_at, NOW())
		WHERE id = $1
		RETURNING
			id::text,
			production_id::text,
			venue_id::text,
			COALESCE(run_id::text, ''),
			status::text,
			audience_view_enabled,
			started_at::text,
			COALESCE(ended_at::text, ''),
			created_by::text,
			session_id::text
	`, showing.ID).Scan(&closed.ID, &closed.ProductionID, &closed.VenueID, &closed.RunID, &closed.Status, &closed.AudienceViewEnabled, &closed.StartedAt, &closed.EndedAt, &closed.CreatedBy, &closed.SessionID); err != nil {
		return Showing{}, err
	}

	return closed, nil
}

func UpdateAudienceViewByID(ctx context.Context, q showingQuerier, showingID string, enabled bool) (Showing, error) {
	showing, err := LoadByID(ctx, q, showingID)
	if err != nil {
		return Showing{}, err
	}
	if strings.EqualFold(strings.TrimSpace(showing.Status), "closed") {
		return Showing{}, errors.New("showing_closed")
	}

	var updated Showing
	if err := q.QueryRow(ctx, `
		UPDATE showings
		SET audience_view_enabled = $2
		WHERE id = $1
		RETURNING
			id::text,
			production_id::text,
			venue_id::text,
			COALESCE(run_id::text, ''),
			status::text,
			audience_view_enabled,
			started_at::text,
			COALESCE(ended_at::text, ''),
			created_by::text,
			session_id::text
	`, showing.ID, enabled).Scan(&updated.ID, &updated.ProductionID, &updated.VenueID, &updated.RunID, &updated.Status, &updated.AudienceViewEnabled, &updated.StartedAt, &updated.EndedAt, &updated.CreatedBy, &updated.SessionID); err != nil {
		return Showing{}, err
	}

	return updated, nil
}

func backfillShowingActions(ctx context.Context, q showingQuerier, sessionID, showingID string) error {
	sessionID = strings.TrimSpace(sessionID)
	showingID = strings.TrimSpace(showingID)
	if sessionID == "" || showingID == "" {
		return nil
	}

	_, err := q.Exec(ctx, `
		UPDATE actions
		SET showing_id = $2::uuid
		WHERE session_id = $1::uuid
		  AND showing_id IS NULL
	`, sessionID, showingID)
	return err
}

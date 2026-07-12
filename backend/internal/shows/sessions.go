package shows

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
)

// LinkSessionToShow sets sessions.show_id, letting a technical live Session
// point back at the Show it belongs to. Deliberately does not touch
// network/session_control.go's venue-triggered /session start flow -- this
// is a separate, manual, data-only association (Kernel 67 explicitly does
// not replace command-based session start behavior).
func LinkSessionToShow(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("session_id_required")
	}

	s, err := LoadShowByID(ctx, pool, showID)
	if err != nil {
		return err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return err
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not_authorized")
	}

	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sessions WHERE id = $1)`, sessionID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errors.New("session_not_found")
	}

	_, err = pool.Exec(ctx, `UPDATE sessions SET show_id = $1 WHERE id = $2`, showID, sessionID)
	return err
}

// UnlinkSessionFromShow clears sessions.show_id, but only if it currently
// points at this exact Show -- the extra guard means unlinking a session
// that has already been relinked to a different Show elsewhere is a no-op
// rather than silently clearing that other link.
func UnlinkSessionFromShow(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, sessionID string) error {
	s, err := LoadShowByID(ctx, pool, showID)
	if err != nil {
		return err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return err
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not_authorized")
	}

	_, err = pool.Exec(ctx, `UPDATE sessions SET show_id = NULL WHERE id = $1 AND show_id = $2`, sessionID, showID)
	return err
}

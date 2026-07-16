package shows

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/network"
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

// StartShowSessionResult is what StartShowSession returns -- combines the
// session-control start/resume result with the Show-link confirmation.
// VenueBusyWithOtherShow is set (and every other field left zero) instead
// of starting/linking anything when the venue's active session already
// belongs to a different Show and the caller didn't pass forceReattach.
type StartShowSessionResult struct {
	SessionID              string
	SessionStatus          string
	ShowingID              string
	WasResumed             bool
	VenueBusyWithOtherShow *VenueBusyInfo
}

type VenueBusyInfo struct {
	OtherShowID    string
	OtherShowTitle string
}

// StartShowSession is the Kernel 70A "Start Show Session" backstage
// action (§2.2): a Director picks a Show and a supported theater venue,
// and the server starts-or-resumes that venue's live Session -- via the
// exact same idempotent core network.StartSessionControl already uses for
// the /session start command, so this never creates a duplicate session
// when one is already active at the venue -- and links it to the Show
// server-side. No manual Session ID is ever handled by the caller.
//
// If the venue's active session is already linked to a DIFFERENT Show,
// this does not silently relink or duplicate: it reports
// VenueBusyWithOtherShow so the caller can prompt for an explicit
// forceReattach retry, per §2.2's "offer to resume or reattach rather than
// create unexplained duplicates."
func StartShowSession(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, venueSlug string, forceReattach bool) (StartShowSessionResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return StartShowSessionResult{}, errors.New("not_authenticated")
	}
	venueSlug = strings.ToLower(strings.TrimSpace(venueSlug))
	if venueSlug == "" {
		return StartShowSessionResult{}, errors.New("venue_slug_required")
	}

	s, err := LoadShowByID(ctx, pool, showID)
	if err != nil {
		return StartShowSessionResult{}, err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return StartShowSessionResult{}, err
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return StartShowSessionResult{}, err
	}
	if !ok {
		return StartShowSessionResult{}, errors.New("not_authorized")
	}

	// Check for a pre-existing active session at this venue before
	// starting/resuming one, so we know both (a) whether this call is a
	// resume (for WasResumed) and (b) whether it's already linked to a
	// different Show (for the busy/reattach path) -- both need the "does
	// an active session already exist" fact, so one query answers both.
	var existingSessionID string
	var existingShowID *string
	err = pool.QueryRow(ctx, `
		SELECT s.id::text, s.show_id::text FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		WHERE v.slug = $1 AND s.status IN ('rehearsal', 'live')
		ORDER BY s.started_at DESC LIMIT 1
	`, venueSlug).Scan(&existingSessionID, &existingShowID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return StartShowSessionResult{}, err
	}
	wasResumed := strings.TrimSpace(existingSessionID) != ""

	if !forceReattach && existingShowID != nil && strings.TrimSpace(*existingShowID) != "" && *existingShowID != showID {
		otherTitle := ""
		if otherShow, loadErr := LoadShowByID(ctx, pool, *existingShowID); loadErr == nil {
			otherTitle = otherShow.Title
		}
		return StartShowSessionResult{
			VenueBusyWithOtherShow: &VenueBusyInfo{OtherShowID: *existingShowID, OtherShowTitle: otherTitle},
		}, nil
	}

	role, err := access.CurrentLocationRole(ctx, pool, actorUserID)
	if err != nil || strings.TrimSpace(role) == "" {
		role = "producer"
	}

	row, err := network.StartSessionControl(ctx, pool, venueSlug, actorUserID, role)
	if err != nil {
		return StartShowSessionResult{}, err
	}

	if err := LinkSessionToShow(ctx, pool, actorUserID, showID, row.Session.ID); err != nil {
		return StartShowSessionResult{}, err
	}

	return StartShowSessionResult{
		SessionID:     row.Session.ID,
		SessionStatus: row.Session.Status,
		ShowingID:     row.Showing.ID,
		WasResumed:    wasResumed,
	}, nil
}

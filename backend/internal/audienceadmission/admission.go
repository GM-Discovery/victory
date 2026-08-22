// Package audienceadmission implements Kernel 93's canonical single-Showing
// Audience admission: "one user admitted to one Showing, as Audience." This
// is deliberately distinct from tickets.Ticket (Kernel 71's two-punch,
// Show-Run-scoped Player ticket) -- see that package's doc comment. Do not
// conflate the two.
//
// Admission targets whichever Showing Kernel 92 Showtime currently has live
// for a Show short code -- the same session-selection query
// showtime.Status/End already use (sessions WHERE show_id = $1 AND status IN
// ('rehearsal','live') ORDER BY started_at DESC LIMIT 1), so this package
// never invents a second live-event selector (spec §25).
package audienceadmission

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showings"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

// Admission is one row of audience_admissions.
type Admission struct {
	ID             string    `json:"id"`
	ShowingID      string    `json:"showing_id"`
	UserID         string    `json:"user_id"`
	UserHandle     string    `json:"user_handle,omitempty"`
	IssuedByUserID string    `json:"issued_by_user_id,omitempty"`
	IssuedAt       time.Time `json:"issued_at"`
}

// resolveCurrentShowingForShow mirrors showtime.Status's session-selection
// query verbatim (small, deliberate per-package duplication of a three-line
// query, matching this codebase's established convention -- see
// participation/resolver.go's activeRosterRole comment for the same rule
// applied elsewhere) rather than importing the showtime package, which would
// create an import cycle back through shows/showruns.
func resolveCurrentShowingForShow(ctx context.Context, pool *pgxpool.Pool, showID string) (showings.Showing, error) {
	var sessionID string
	err := pool.QueryRow(ctx, `
		SELECT sess.id::text
		FROM sessions sess
		WHERE sess.show_id = $1 AND sess.status IN ('rehearsal', 'live')
		ORDER BY sess.started_at DESC LIMIT 1
	`, showID).Scan(&sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return showings.Showing{}, errors.New("no_active_showing")
	}
	if err != nil {
		return showings.Showing{}, err
	}
	return showings.LoadBySession(ctx, pool, sessionID)
}

// resolveManagingShowAndShowing resolves shortCode to its Show, checks
// actorUserID can manage that Show's Show Run (the same authority
// showtime.Start/End/Status require), and resolves the Show's currently
// live/rehearsal Showing. Returns "not_authorized" or "no_active_showing" on
// failure, matching showtime's own error vocabulary.
func resolveManagingShowAndShowing(ctx context.Context, pool *pgxpool.Pool, actorUserID, shortCode string) (shows.Show, showings.Showing, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return shows.Show{}, showings.Showing{}, errors.New("not_authenticated")
	}

	s, err := shows.LoadShowByCode(ctx, pool, actorUserID, shortCode)
	if err != nil {
		return shows.Show{}, showings.Showing{}, err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return shows.Show{}, showings.Showing{}, err
	}
	canManage, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return shows.Show{}, showings.Showing{}, err
	}
	if !canManage {
		return shows.Show{}, showings.Showing{}, errors.New("not_authorized")
	}

	showing, err := resolveCurrentShowingForShow(ctx, pool, s.ID)
	if err != nil {
		return shows.Show{}, showings.Showing{}, err
	}
	return s, showing, nil
}

// IssueForShowCode admits targetHandle to shortCode's currently live/
// rehearsal Showing as Audience. Idempotent: re-issuing to an already
// admitted user refreshes issued_at rather than erroring, matching
// showruns.SelfJoinAsAudience's idempotency convention.
func IssueForShowCode(ctx context.Context, pool *pgxpool.Pool, actorUserID, shortCode, targetHandle string) (Admission, error) {
	targetHandle = strings.TrimSpace(strings.ToLower(targetHandle))
	if targetHandle == "" {
		return Admission{}, errors.New("handle_required")
	}

	_, showing, err := resolveManagingShowAndShowing(ctx, pool, actorUserID, shortCode)
	if err != nil {
		return Admission{}, err
	}

	var targetUserID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM users WHERE lower(handle) = $1`, targetHandle).Scan(&targetUserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Admission{}, errors.New("user_not_found")
		}
		return Admission{}, err
	}

	var a Admission
	row := pool.QueryRow(ctx, `
		INSERT INTO audience_admissions (showing_id, user_id, issued_by_user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (showing_id, user_id) DO UPDATE
		SET issued_by_user_id = EXCLUDED.issued_by_user_id,
		    issued_at = NOW()
		RETURNING id::text, showing_id::text, user_id::text, COALESCE(issued_by_user_id::text, ''), issued_at
	`, showing.ID, targetUserID, actorUserID)
	if err := row.Scan(&a.ID, &a.ShowingID, &a.UserID, &a.IssuedByUserID, &a.IssuedAt); err != nil {
		return Admission{}, err
	}
	a.UserHandle = targetHandle
	return a, nil
}

// ListForShowCode returns every Audience admission for shortCode's current
// live/rehearsal Showing, for the managing Director/Producer/Operator.
func ListForShowCode(ctx context.Context, pool *pgxpool.Pool, actorUserID, shortCode string) ([]Admission, error) {
	_, showing, err := resolveManagingShowAndShowing(ctx, pool, actorUserID, shortCode)
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(ctx, `
		SELECT aa.id::text, aa.showing_id::text, aa.user_id::text,
		       COALESCE(u.handle, ''), COALESCE(aa.issued_by_user_id::text, ''), aa.issued_at
		FROM audience_admissions aa
		JOIN users u ON u.id = aa.user_id
		WHERE aa.showing_id = $1
		ORDER BY aa.issued_at DESC
	`, showing.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Admission{}
	for rows.Next() {
		var a Admission
		if err := rows.Scan(&a.ID, &a.ShowingID, &a.UserID, &a.UserHandle, &a.IssuedByUserID, &a.IssuedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// HasValidAdmission reports whether userID holds an Audience admission for
// showingID. Exposed for callers (e.g. audience-facing "am I admitted?"
// checks) that need a plain boolean rather than the full row.
func HasValidAdmission(ctx context.Context, pool *pgxpool.Pool, showingID, userID string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM audience_admissions WHERE showing_id = $1 AND user_id = $2)
	`, showingID, userID).Scan(&exists)
	return exists, err
}

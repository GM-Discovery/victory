package tickets

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
)

// ticketColumns is always selected against a show_run_tickets row aliased
// "t" -- including single-table queries -- so the same constant is safe to
// reuse in ListIncomingForDirector's join against show_runs without any
// column-name ambiguity (show_runs also has its own "id").
const ticketColumns = `
	t.id::text, t.show_run_id::text, t.user_id::text, t.requested_role, t.initiated_by_side,
	t.player_punched_at, COALESCE(t.player_punched_by_user_id::text, ''),
	t.director_punched_at, COALESCE(t.director_punched_by_user_id::text, ''),
	t.status, COALESCE(t.message, ''), COALESCE(t.roster_membership_id::text, ''),
	t.created_at, t.updated_at, t.resolved_at
`

type ticketRow interface {
	Scan(dest ...any) error
}

func scanTicket(row ticketRow) (Ticket, error) {
	var t Ticket
	if err := row.Scan(
		&t.ID, &t.ShowRunID, &t.UserID, &t.RequestedRole, &t.InitiatedBySide,
		&t.PlayerPunchedAt, &t.PlayerPunchedByUserID,
		&t.DirectorPunchedAt, &t.DirectorPunchedByUserID,
		&t.Status, &t.Message, &t.RosterMembershipID,
		&t.CreatedAt, &t.UpdatedAt, &t.ResolvedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Ticket{}, errors.New("ticket_not_found")
		}
		return Ticket{}, err
	}
	return t, nil
}

// resolveProfileUserID maps an opaque Player Workbook / profile ID to the
// account's stable UUID -- a local copy of the same helper showruns/roster.go
// duplicates from playerrelationships (Kernel 61 contract reuse), matching
// this codebase's established per-package-duplication convention.
func resolveProfileUserID(ctx context.Context, pool *pgxpool.Pool, profileID string) (string, error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return "", errors.New("profile_id_required")
	}
	var userID string
	if err := pool.QueryRow(ctx, `
		SELECT user_id::text FROM player_profile_workbooks WHERE id = $1
	`, profileID).Scan(&userID); err != nil {
		return "", errors.New("profile_not_found")
	}
	return userID, nil
}

// RequestFromPlayer is the first punch, Player-initiated: a Player asks to
// join a Show Run. Grants nothing, creates no roster row -- it only records
// that the Player has punched and surfaces to the Director side.
func RequestFromPlayer(ctx context.Context, pool *pgxpool.Pool, actorUserID, showRunID, message string) (Ticket, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	showRunID = strings.TrimSpace(showRunID)
	if actorUserID == "" {
		return Ticket{}, errors.New("not_authenticated")
	}
	if showRunID == "" {
		return Ticket{}, errors.New("show_run_id_required")
	}
	if _, err := showruns.LoadShowRunByID(ctx, pool, showRunID); err != nil {
		return Ticket{}, err
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO show_run_tickets AS t (
			show_run_id, user_id, requested_role, initiated_by_side,
			player_punched_at, player_punched_by_user_id, status, message
		)
		VALUES ($1, $2, 'player', 'player', NOW(), $2, 'pending_director', NULLIF($3, ''))
		RETURNING `+ticketColumns,
		showRunID, actorUserID, message)
	t, err := scanTicket(row)
	if err != nil {
		if isUniqueViolation(err) {
			return Ticket{}, errors.New("ticket_already_pending")
		}
		return Ticket{}, err
	}
	return t, nil
}

// InviteFromDirector is the first punch, Director-initiated: a Director
// (or Producer/Operator managing the run) invites a specific Player.
// Grants nothing, creates no roster row.
func InviteFromDirector(ctx context.Context, pool *pgxpool.Pool, actorUserID, showRunID, targetProfileID, message string) (Ticket, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	showRunID = strings.TrimSpace(showRunID)
	if actorUserID == "" {
		return Ticket{}, errors.New("not_authenticated")
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, showRunID)
	if err != nil {
		return Ticket{}, err
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return Ticket{}, err
	}
	if !ok {
		return Ticket{}, errors.New("not_authorized")
	}

	targetUserID, err := resolveProfileUserID(ctx, pool, targetProfileID)
	if err != nil {
		return Ticket{}, err
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO show_run_tickets AS t (
			show_run_id, user_id, requested_role, initiated_by_side,
			director_punched_at, director_punched_by_user_id, status, message
		)
		VALUES ($1, $2, 'player', 'director', NOW(), $3, 'pending_player', NULLIF($4, ''))
		RETURNING `+ticketColumns,
		showRunID, targetUserID, actorUserID, message)
	t, err := scanTicket(row)
	if err != nil {
		if isUniqueViolation(err) {
			return Ticket{}, errors.New("ticket_already_pending")
		}
		return Ticket{}, err
	}
	return t, nil
}

// SecondPunch is the one atomic transaction that turns a pending ticket
// into valid Player participation. Whichever side is waiting punches here;
// the row lock below serializes concurrent attempts on the same ticket so
// two racing callers can never both create a roster row, and the roster
// insert's own partial-unique-index ON CONFLICT (the same one
// showruns.AddRosterMember already relies on) makes a duplicate active
// roster row structurally impossible even if the lock were somehow
// bypassed.
func SecondPunch(ctx context.Context, pool *pgxpool.Pool, actorUserID, ticketID string) (Ticket, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	ticketID = strings.TrimSpace(ticketID)
	if actorUserID == "" {
		return Ticket{}, errors.New("not_authenticated")
	}
	if ticketID == "" {
		return Ticket{}, errors.New("ticket_id_required")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return Ticket{}, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `SELECT `+ticketColumns+` FROM show_run_tickets t WHERE t.id = $1 FOR UPDATE`, ticketID)
	t, err := scanTicket(row)
	if err != nil {
		return Ticket{}, err
	}

	// Already resolved by a concurrent caller who got the lock first --
	// idempotent success, not an error, matching spec §11's "concurrent
	// second-punch attempts produce one valid result."
	if t.Status == StatusValid {
		return t, nil
	}
	if t.Status != StatusPendingPlayer && t.Status != StatusPendingDirector {
		return Ticket{}, errors.New("ticket_not_pending")
	}

	var directorActorForRoster string
	switch t.Status {
	case StatusPendingPlayer:
		// The Director already punched; the Player is the waiting side.
		// The Player must equal the ticket's own subject -- never a
		// client-supplied user id substituting for someone else.
		if actorUserID != t.UserID {
			return Ticket{}, errors.New("not_authorized")
		}
		if _, err := tx.Exec(ctx, `
			UPDATE show_run_tickets
			SET player_punched_at = NOW(), player_punched_by_user_id = $1, updated_at = NOW()
			WHERE id = $2
		`, actorUserID, ticketID); err != nil {
			return Ticket{}, err
		}
		directorActorForRoster = t.DirectorPunchedByUserID

	case StatusPendingDirector:
		// The Player already punched; a Director/Producer/Operator
		// managing this specific Show Run is the waiting side. The
		// show_run_id used for this authority check comes from the
		// just-locked ticket row, never from a client-supplied value.
		sr, err := showruns.LoadShowRunByID(ctx, pool, t.ShowRunID)
		if err != nil {
			return Ticket{}, err
		}
		canManage, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
		if err != nil {
			return Ticket{}, err
		}
		if !canManage {
			return Ticket{}, errors.New("not_authorized")
		}
		if _, err := tx.Exec(ctx, `
			UPDATE show_run_tickets
			SET director_punched_at = NOW(), director_punched_by_user_id = $1, updated_at = NOW()
			WHERE id = $2
		`, actorUserID, ticketID); err != nil {
			return Ticket{}, err
		}
		directorActorForRoster = actorUserID
	}

	if strings.TrimSpace(directorActorForRoster) == "" {
		// Defensive fallback -- should be unreachable since both punch
		// paths above always have a director-side actor by the time we
		// get here, but added_by_user_id is NOT NULL and must never be
		// silently left empty.
		directorActorForRoster = actorUserID
	}

	var rosterMembershipID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id)
		VALUES ($1, $2, 'player', $3)
		ON CONFLICT (show_run_id, user_id) WHERE removed_at IS NULL
		DO UPDATE SET role = 'player'
		RETURNING id::text
	`, t.ShowRunID, t.UserID, directorActorForRoster).Scan(&rosterMembershipID); err != nil {
		return Ticket{}, err
	}

	row = tx.QueryRow(ctx, `
		UPDATE show_run_tickets AS t
		SET status = 'valid', roster_membership_id = $1, resolved_at = NOW(), updated_at = NOW()
		WHERE t.id = $2
		RETURNING `+ticketColumns,
		rosterMembershipID, ticketID)
	t, err = scanTicket(row)
	if err != nil {
		return Ticket{}, err
	}

	// Compatibility side effect (spec §3.1): ensure a legacy access_grants
	// row exists for any not-yet-migrated reader outside this package.
	// Never read back by the canonical resolver as a truth source --
	// purely insurance for stragglers like assets/upload.go's
	// location_memberships-only check. Scoped by production_id (every Show
	// Run has one) rather than venue_id, since a Show Run isn't tied to a
	// single venue -- access_grants_requires_scope needs one or the other.
	if _, err := tx.Exec(ctx, `
		INSERT INTO access_grants (location_id, user_id, grant_type, production_id, granted_by_user_id)
		SELECT sr.location_id, $1, 'venue_access', sr.production_id, $2
		FROM show_runs sr WHERE sr.id = $3
		ON CONFLICT DO NOTHING
	`, t.UserID, directorActorForRoster, t.ShowRunID); err != nil {
		return Ticket{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Ticket{}, err
	}
	return t, nil
}

// Decline is the waiting side declining a pending ticket before validation.
// Grants nothing; the ticket stays auditable.
func Decline(ctx context.Context, pool *pgxpool.Pool, actorUserID, ticketID string) (Ticket, error) {
	return resolvePreValidation(ctx, pool, actorUserID, ticketID, StatusDeclined, waitingSide)
}

// Withdraw is the initiating side pulling back a pending ticket before
// validation.
func Withdraw(ctx context.Context, pool *pgxpool.Pool, actorUserID, ticketID string) (Ticket, error) {
	return resolvePreValidation(ctx, pool, actorUserID, ticketID, StatusWithdrawn, initiatingSide)
}

type actorSide int

const (
	waitingSide actorSide = iota
	initiatingSide
)

func resolvePreValidation(ctx context.Context, pool *pgxpool.Pool, actorUserID, ticketID, newStatus string, side actorSide) (Ticket, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	ticketID = strings.TrimSpace(ticketID)
	if actorUserID == "" {
		return Ticket{}, errors.New("not_authenticated")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return Ticket{}, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `SELECT `+ticketColumns+` FROM show_run_tickets t WHERE t.id = $1 FOR UPDATE`, ticketID)
	t, err := scanTicket(row)
	if err != nil {
		return Ticket{}, err
	}
	if t.Status != StatusPendingPlayer && t.Status != StatusPendingDirector {
		return Ticket{}, errors.New("ticket_not_pending")
	}

	var authorized bool
	switch side {
	case waitingSide:
		switch t.Status {
		case StatusPendingPlayer:
			authorized = actorUserID == t.UserID
		case StatusPendingDirector:
			sr, err := showruns.LoadShowRunByID(ctx, pool, t.ShowRunID)
			if err != nil {
				return Ticket{}, err
			}
			canManage, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
			if err != nil {
				return Ticket{}, err
			}
			authorized = canManage
		}
	case initiatingSide:
		switch t.InitiatedBySide {
		case InitiatedBySidePlayer:
			authorized = actorUserID == t.UserID
		case InitiatedBySideDirector:
			authorized = actorUserID == t.DirectorPunchedByUserID
		}
	}
	if !authorized {
		return Ticket{}, errors.New("not_authorized")
	}

	row = tx.QueryRow(ctx, `
		UPDATE show_run_tickets AS t
		SET status = $1, resolved_at = NOW(), updated_at = NOW()
		WHERE t.id = $2
		RETURNING `+ticketColumns,
		newStatus, ticketID)
	t, err = scanTicket(row)
	if err != nil {
		return Ticket{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Ticket{}, err
	}
	return t, nil
}

// ListMineAsPlayer returns every ticket where the caller is the Player
// subject -- both invitations awaiting the Player's punch and requests the
// Player already punched that await the Director.
func ListMineAsPlayer(ctx context.Context, pool *pgxpool.Pool, userID string) ([]Ticket, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("not_authenticated")
	}
	rows, err := pool.Query(ctx, `
		SELECT `+ticketColumns+`
		FROM show_run_tickets t
		WHERE t.user_id = $1
		ORDER BY t.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanTicketRows(rows)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if sr, err := showruns.LoadShowRunByID(ctx, pool, list[i].ShowRunID); err == nil {
			list[i].ShowRunTitle = sr.Title
		}
	}
	return list, nil
}

// ListIncomingForDirector returns every pending ticket for a Show Run the
// caller manages (Producer/Director/Operator).
func ListIncomingForDirector(ctx context.Context, pool *pgxpool.Pool, actorUserID string) ([]Ticket, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return nil, errors.New("not_authenticated")
	}
	rows, err := pool.Query(ctx, `
		SELECT `+ticketColumns+`
		FROM show_run_tickets t
		WHERE t.status IN ('pending_player', 'pending_director')
		ORDER BY t.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	all, err := scanTicketRows(rows)
	if err != nil {
		return nil, err
	}

	out := make([]Ticket, 0, len(all))
	for _, t := range all {
		sr, err := showruns.LoadShowRunByID(ctx, pool, t.ShowRunID)
		if err != nil {
			continue
		}
		canManage, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
		if err != nil || !canManage {
			continue
		}
		t.ShowRunTitle = sr.Title
		out = append(out, t)
	}
	return out, nil
}

func scanTicketRows(rows pgx.Rows) ([]Ticket, error) {
	var out []Ticket
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}

package showruns

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

const rosterMemberColumns = `
	id::text, show_run_id::text, user_id::text, role,
	COALESCE(custom_role_label, ''), program_visible,
	added_by_user_id::text, added_at, removed_at
`

func scanRosterMember(row pgx.Row) (RosterMember, error) {
	var m RosterMember
	if err := row.Scan(
		&m.ID, &m.ShowRunID, &m.UserID, &m.Role, &m.CustomRoleLabel,
		&m.ProgramVisible, &m.AddedByUserID, &m.AddedAt, &m.RemovedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RosterMember{}, errors.New("roster_member_not_found")
		}
		return RosterMember{}, err
	}
	return m, nil
}

// resolveProfileUserID maps an opaque Player Workbook / profile ID to the
// account's stable UUID -- a local copy of playerrelationships'
// unexported resolveSubjectUserID (Kernel 61 contract reuse), matching this
// codebase's existing per-package-duplication convention rather than newly
// exporting it from playerprofile.
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

// AddRosterMember adds targetProfileID to the run's roster with the given
// role. Adding someone as Audience is rejected if they have an active block
// on this run, unless the actor is Operator (Kernel 66's explicit override
// case).
func AddRosterMember(ctx context.Context, pool *pgxpool.Pool, actorUserID, showRunID, targetProfileID, role, customRoleLabel string, programVisible bool) (RosterMember, error) {
	sr, err := LoadShowRunByID(ctx, pool, showRunID)
	if err != nil {
		return RosterMember{}, err
	}
	ok, err := CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return RosterMember{}, err
	}
	if !ok {
		return RosterMember{}, errors.New("not_authorized")
	}

	targetUserID, err := resolveProfileUserID(ctx, pool, targetProfileID)
	if err != nil {
		return RosterMember{}, err
	}

	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return RosterMember{}, errors.New("role_required")
	}

	if role == "audience" {
		isOperator, err := access.IsOperatorUser(ctx, pool, actorUserID)
		if err != nil {
			return RosterMember{}, err
		}
		if !isOperator {
			blocked, err := IsUserBlocked(ctx, pool, showRunID, targetUserID)
			if err != nil {
				return RosterMember{}, err
			}
			if blocked {
				return RosterMember{}, errors.New("user_blocked")
			}
		}
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, custom_role_label, program_visible, added_by_user_id)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6)
		ON CONFLICT (show_run_id, user_id) WHERE removed_at IS NULL
		DO UPDATE SET role = $3, custom_role_label = NULLIF($4, ''), program_visible = $5
		RETURNING `+rosterMemberColumns,
		showRunID, targetUserID, role, customRoleLabel, programVisible, actorUserID)
	return scanRosterMember(row)
}

// UpdateRosterMemberRole changes an existing roster row's role/visibility in
// place -- it does not create a second row for the same user.
func UpdateRosterMemberRole(ctx context.Context, pool *pgxpool.Pool, actorUserID, showRunID, memberID, role, customRoleLabel string, programVisible bool) (RosterMember, error) {
	sr, err := LoadShowRunByID(ctx, pool, showRunID)
	if err != nil {
		return RosterMember{}, err
	}
	ok, err := CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return RosterMember{}, err
	}
	if !ok {
		return RosterMember{}, errors.New("not_authorized")
	}

	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return RosterMember{}, errors.New("role_required")
	}

	row := pool.QueryRow(ctx, `
		UPDATE show_run_roster_members
		SET role = $3, custom_role_label = NULLIF($4, ''), program_visible = $5
		WHERE id = $1 AND show_run_id = $2 AND removed_at IS NULL
		RETURNING `+rosterMemberColumns,
		memberID, showRunID, role, customRoleLabel, programVisible)
	return scanRosterMember(row)
}

// RemoveRosterMember soft-closes a roster row -- it is never deleted, so the
// run keeps an honest membership history.
func RemoveRosterMember(ctx context.Context, pool *pgxpool.Pool, actorUserID, showRunID, memberID string) error {
	sr, err := LoadShowRunByID(ctx, pool, showRunID)
	if err != nil {
		return err
	}
	ok, err := CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not_authorized")
	}
	_, err = pool.Exec(ctx, `
		UPDATE show_run_roster_members
		SET removed_at = NOW()
		WHERE id = $1 AND show_run_id = $2 AND removed_at IS NULL
	`, memberID, showRunID)
	return err
}

// SelfJoinAsAudience lets an authenticated user join a run's Audience
// themselves, when the run allows it and they are not blocked. Idempotent:
// repeated calls refresh the existing row rather than erroring, mirroring
// thirdplace.LeaveHeadshot's join idempotency.
func SelfJoinAsAudience(ctx context.Context, pool *pgxpool.Pool, userID, showRunID string) (RosterMember, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return RosterMember{}, errors.New("not_authenticated")
	}
	sr, err := LoadShowRunByID(ctx, pool, showRunID)
	if err != nil {
		return RosterMember{}, err
	}
	if !sr.AudienceSelfJoinEnabled {
		return RosterMember{}, errors.New("self_join_disabled")
	}
	hasMembership, err := access.HasActiveLocationMembership(ctx, pool, userID, sr.LocationID)
	if err != nil {
		return RosterMember{}, err
	}
	if !hasMembership {
		return RosterMember{}, errors.New("not_authorized")
	}

	blocked, err := IsUserBlocked(ctx, pool, showRunID, userID)
	if err != nil {
		return RosterMember{}, err
	}
	if blocked {
		return RosterMember{}, errors.New("user_blocked")
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, program_visible, added_by_user_id)
		VALUES ($1, $2, 'audience', TRUE, $2)
		ON CONFLICT (show_run_id, user_id) WHERE removed_at IS NULL
		DO UPDATE SET program_visible = TRUE
		RETURNING `+rosterMemberColumns, showRunID, userID)
	return scanRosterMember(row)
}

// ListInternalRoster returns every active roster row regardless of role or
// program_visible. Callers must already have authority-checked
// Producer/Director/Operator before calling this -- it performs no check of
// its own.
func ListInternalRoster(ctx context.Context, pool *pgxpool.Pool, showRunID string) ([]RosterMember, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+rosterMemberColumns+`
		FROM show_run_roster_members
		WHERE show_run_id = $1 AND removed_at IS NULL
		ORDER BY added_at ASC
	`, showRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RosterMember
	for rows.Next() {
		m, err := scanRosterMember(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListAudienceProgramMembers returns only program_visible active roster
// rows, with Audience ordered first -- the explicit "best seats" requirement
// (Kernel 66). This deliberately does not reuse the Producer-first CASE
// ordering used for authority-priority elsewhere in this codebase; that
// ordering is correct for "who has the most power" and wrong for this view.
func ListAudienceProgramMembers(ctx context.Context, pool *pgxpool.Pool, showRunID string) ([]RosterMember, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+rosterMemberColumns+`
		FROM show_run_roster_members
		WHERE show_run_id = $1 AND removed_at IS NULL AND program_visible = TRUE
		ORDER BY
			CASE WHEN role = 'audience' THEN 0 ELSE 1 END,
			added_at ASC
	`, showRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RosterMember
	for rows.Next() {
		m, err := scanRosterMember(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

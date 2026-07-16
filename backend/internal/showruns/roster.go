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
	added_by_user_id::text, added_at, removed_at, COALESCE(character_card_id::text, '')
`

func scanRosterMember(row pgx.Row) (RosterMember, error) {
	var m RosterMember
	if err := row.Scan(
		&m.ID, &m.ShowRunID, &m.UserID, &m.Role, &m.CustomRoleLabel,
		&m.ProgramVisible, &m.AddedByUserID, &m.AddedAt, &m.RemovedAt, &m.CharacterCardID,
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

	// Kernel 71: this endpoint's ON CONFLICT ... DO UPDATE means it can
	// both create a fresh Player row AND silently promote an existing
	// non-player row to Player -- either would let an ordinary Director
	// bypass the two-punch ticket flow entirely by calling this generic
	// roster-add endpoint directly instead of the UI's gated "Invite a
	// Player" control. Reject outright unless Operator; the tickets
	// package's SecondPunch is the only ordinary path that may create an
	// active Player roster row (spec §4.3, §6.2).
	if role == "player" {
		isOperator, err := access.IsOperatorUser(ctx, pool, actorUserID)
		if err != nil {
			return RosterMember{}, err
		}
		if !isOperator {
			return RosterMember{}, errors.New("player_requires_ticket")
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

	// Kernel 71: promoting an existing row to Player is the same consent
	// event a fresh ticket's second punch grants -- an ordinary Director
	// must not be able to sidestep the two-punch flow by adding someone
	// under a different role first and then PATCHing it to Player.
	// Operator retains the explicit, logged admin-correction path (spec
	// §6.2); the added_by_user_id column already records who performed it.
	if role == "player" {
		var currentRole string
		if err := pool.QueryRow(ctx, `
			SELECT role FROM show_run_roster_members WHERE id = $1 AND show_run_id = $2 AND removed_at IS NULL
		`, memberID, showRunID).Scan(&currentRole); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return RosterMember{}, errors.New("roster_member_not_found")
			}
			return RosterMember{}, err
		}
		if currentRole != "player" {
			isOperator, err := access.IsOperatorUser(ctx, pool, actorUserID)
			if err != nil {
				return RosterMember{}, err
			}
			if !isOperator {
				return RosterMember{}, errors.New("player_requires_ticket")
			}
		}
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

// LoadMyRosterMember returns the caller's own active roster row for a Show
// Run, or "roster_member_not_found" if they have none. Self-service, no
// CanManageShowRun/CanViewBackstage gate -- a ticket-only Player (no
// location_memberships) must be able to see their own roster status and
// selected Character without backstage authority.
func LoadMyRosterMember(ctx context.Context, pool *pgxpool.Pool, actorUserID, showRunID string) (RosterMember, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return RosterMember{}, errors.New("not_authenticated")
	}
	row := pool.QueryRow(ctx, `
		SELECT `+rosterMemberColumns+`
		FROM show_run_roster_members
		WHERE show_run_id = $1 AND user_id = $2 AND removed_at IS NULL
	`, showRunID, actorUserID)
	return scanRosterMember(row)
}

// SelectCharacter lets the caller pick or change which of their own active
// Characters they're presenting as for this Show Run (Kernel 71 §7). There
// is no target-user parameter -- the caller can only ever act on their own
// roster row, closing "client-supplied user ID cannot substitute another
// subject" by construction rather than by a check. Pass "" for
// characterCardID to explicitly unselect (e.g. the frontend detected the
// previously-selected Character was archived and needs to clear it).
func SelectCharacter(ctx context.Context, pool *pgxpool.Pool, actorUserID, showRunID, characterCardID string) (RosterMember, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	showRunID = strings.TrimSpace(showRunID)
	characterCardID = strings.TrimSpace(characterCardID)
	if actorUserID == "" {
		return RosterMember{}, errors.New("not_authenticated")
	}

	var memberID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text FROM show_run_roster_members
		WHERE show_run_id = $1 AND user_id = $2 AND removed_at IS NULL
	`, showRunID, actorUserID).Scan(&memberID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RosterMember{}, errors.New("not_a_roster_member")
		}
		return RosterMember{}, err
	}

	if characterCardID != "" {
		var ownerUserID string
		var isDeleted bool
		if err := pool.QueryRow(ctx, `
			SELECT owner_user_id::text, is_deleted FROM character_cards WHERE id = $1
		`, characterCardID).Scan(&ownerUserID, &isDeleted); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return RosterMember{}, errors.New("character_not_found")
			}
			return RosterMember{}, err
		}
		if ownerUserID != actorUserID {
			return RosterMember{}, errors.New("not_authorized")
		}
		if isDeleted {
			return RosterMember{}, errors.New("character_archived")
		}
	}

	row := pool.QueryRow(ctx, `
		UPDATE show_run_roster_members
		SET character_card_id = NULLIF($1, '')::uuid
		WHERE id = $2
		RETURNING `+rosterMemberColumns,
		characterCardID, memberID)
	return scanRosterMember(row)
}

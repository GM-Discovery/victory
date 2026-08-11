// Package rollaudience resolves the server-side audience for a theatrical
// roll projection (Kernel 86): who should receive the live payload, and who
// may see it later in snapshot/history. It is a small leaf package (only
// pgx as a dependency) so it can be imported by both backend/internal/actions
// (roll storage, in a transaction) and backend/internal/world (snapshot
// filtering, against the pool) without creating an import cycle -- the same
// avoidance reason backend/internal/world/kernel85_cohort_projection.go
// gives for duplicating cohort-assignment SQL instead of importing
// backend/internal/cohorts (which imports backend/internal/network, and
// network already imports both actions and world).
//
// "Director+" here means exactly what it means everywhere else in this
// codebase's authority checks (actions/authority.go's canActDiceRoll,
// cohorts.requireManage): role producer or director. It is deliberately
// narrower than world.isBackstageRole (which also admits operator/crew for
// stage-Elements visibility) -- Kernel 86's spec asks for "Director+
// authority", not the broader backstage tier.
package rollaudience

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Querier is satisfied by both *pgxpool.Pool (snapshot reads) and pgx.Tx
// (roll storage, inside StoreDiceRoll's transaction).
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

const (
	ModeShow     = "show"
	ModeCohort   = "cohort"
	ModeDirector = "director"
	ModePrivate  = "private"
)

// NormalizeMode maps a client-requested mode onto the four Kernel 86 modes.
// "" and "cohort" both mean "use the default", which Resolve then narrows
// further (Ungrouped fallback). "public" is preserved for backward
// compatibility with pre-Kernel-86 callers/tests as an alias for Show (spec
// §3: "Preserve backward compatibility with any current 'public' value").
func NormalizeMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "cohort":
		return ModeCohort
	case "public", "show":
		return ModeShow
	case "director":
		return ModeDirector
	case "private":
		return ModePrivate
	default:
		return ""
	}
}

// Decision is the fully-resolved audience for one roll: the mode actually
// in effect after fallback, plus the Show/Cohort it was resolved against.
type Decision struct {
	Mode     string
	ShowID   string
	CohortID string
}

// Resolve computes the audience for a roll request server-side. requestedMode
// should already have passed through NormalizeMode (empty string is
// rejected here, not treated as a default, so callers can distinguish
// "unsupported mode" from "no session" errors before calling Resolve).
func Resolve(ctx context.Context, q Querier, sessionID, actorID, mode string) (Decision, error) {
	sessionID = strings.TrimSpace(sessionID)
	actorID = strings.TrimSpace(actorID)
	if mode != ModeShow && mode != ModeCohort && mode != ModeDirector && mode != ModePrivate {
		return Decision{}, errors.New("unsupported_visibility_mode")
	}

	showID, err := resolveSessionShowID(ctx, q, sessionID)
	if err != nil {
		return Decision{}, err
	}
	d := Decision{Mode: mode, ShowID: showID}

	if mode != ModeCohort {
		return d, nil
	}

	// Cohort mode needs a Show to resolve cohorts against; a session with
	// no Show (e.g. a legacy/tutorial venue) has no cohort concept, so
	// fall back to Show mode -- there is nothing narrower to compute.
	if showID == "" {
		d.Mode = ModeShow
		return d, nil
	}

	cohortID, err := resolveCohortIDForActor(ctx, q, showID, actorID)
	if err != nil {
		return Decision{}, err
	}
	if cohortID == "" {
		// Ungrouped participant: spec §1.10 safe fallback.
		d.Mode = ModeShow
		return d, nil
	}
	d.CohortID = cohortID
	return d, nil
}

// VisibleToViewer reports whether viewerUserID (holding viewerRole in this
// session) may see a roll recorded with Decision d, authored by actorID.
// The roller always sees their own roll. showID is the *current* session's
// resolved show (passed in by the caller, e.g. world.LoadVenueSnapshot
// already has it) -- used only to re-resolve the viewer's own current
// cohort membership for the ModeCohort comparison against d.CohortID.
func VisibleToViewer(ctx context.Context, q Querier, d Decision, actorID, showID, viewerUserID, viewerRole string) (bool, error) {
	viewerUserID = strings.TrimSpace(viewerUserID)
	if viewerUserID != "" && viewerUserID == strings.TrimSpace(actorID) {
		return true, nil
	}

	switch d.Mode {
	case ModeShow, "":
		return true, nil
	case ModeCohort:
		if isDirectorPlusRole(viewerRole) {
			return true, nil
		}
		if d.CohortID == "" || showID == "" || viewerUserID == "" {
			return false, nil
		}
		viewerCohortID, err := resolveCohortIDForActor(ctx, q, showID, viewerUserID)
		if err != nil {
			return false, err
		}
		return viewerCohortID != "" && viewerCohortID == d.CohortID, nil
	case ModeDirector:
		return isDirectorPlusRole(viewerRole), nil
	case ModePrivate:
		return false, nil
	default:
		return true, nil
	}
}

// LiveRecipients resolves concrete recipient user IDs for targeted delivery
// of a just-stored roll. useSessionBroadcast=true (ModeShow) means "send to
// every socket on this session" -- the caller should use a session-scoped
// broadcast rather than an enumerated user-ID set, since Show mode is
// intentionally everyone already able to view this session/Show.
func LiveRecipients(ctx context.Context, q Querier, d Decision, actorID, sessionID string) (userIDs []string, useSessionBroadcast bool, err error) {
	actorID = strings.TrimSpace(actorID)
	switch d.Mode {
	case ModeShow, "":
		return nil, true, nil
	case ModeCohort:
		set := map[string]bool{actorID: true}
		if d.CohortID != "" {
			members, err := resolveCohortMemberIDs(ctx, q, d.CohortID)
			if err != nil {
				return nil, false, err
			}
			for _, m := range members {
				set[m] = true
			}
		}
		directors, err := resolveDirectorPlusIDs(ctx, q, sessionID)
		if err != nil {
			return nil, false, err
		}
		for _, m := range directors {
			set[m] = true
		}
		return setKeys(set), false, nil
	case ModeDirector:
		set := map[string]bool{actorID: true}
		directors, err := resolveDirectorPlusIDs(ctx, q, sessionID)
		if err != nil {
			return nil, false, err
		}
		for _, m := range directors {
			set[m] = true
		}
		return setKeys(set), false, nil
	case ModePrivate:
		return []string{actorID}, false, nil
	default:
		return nil, true, nil
	}
}

// IsDirectorPlus reports whether userID currently holds Director+
// authority (director or producer role) in sessionID -- exported for
// callers that need a single-user check outside the roll-storage/snapshot
// paths, e.g. Stage Effect pin/dismiss authority in network/ws.go.
func IsDirectorPlus(ctx context.Context, q Querier, sessionID, userID string) (bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false, nil
	}
	ids, err := resolveDirectorPlusIDs(ctx, q, sessionID)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if id == userID {
			return true, nil
		}
	}
	return false, nil
}

func isDirectorPlusRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "director", "producer":
		return true
	default:
		return false
	}
}

func setKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		if k != "" {
			out = append(out, k)
		}
	}
	return out
}

func resolveSessionShowID(ctx context.Context, q Querier, sessionID string) (string, error) {
	if sessionID == "" {
		return "", nil
	}
	var showID *string
	err := q.QueryRow(ctx, `SELECT show_id::text FROM sessions WHERE id = $1`, sessionID).Scan(&showID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if showID == nil {
		return "", nil
	}
	return *showID, nil
}

// resolveCohortIDForActor mirrors world/kernel85_cohort_projection.go's
// resolveCohortPlacementForViewer query shape but returns the cohort_id
// itself rather than its current Scene placement -- see that file's header
// comment for why this is a narrow duplicated query rather than an import
// of backend/internal/cohorts.
func resolveCohortIDForActor(ctx context.Context, q Querier, showID, userID string) (string, error) {
	if showID == "" || userID == "" {
		return "", nil
	}
	var cohortID *string
	err := q.QueryRow(ctx, `
		SELECT a.cohort_id::text
		FROM show_cohort_assignments a
		JOIN show_cohorts c ON c.id = a.cohort_id
		WHERE a.show_id = $1 AND a.user_id = $2 AND c.archived_at IS NULL
	`, showID, userID).Scan(&cohortID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if cohortID == nil {
		return "", nil
	}
	return *cohortID, nil
}

func resolveCohortMemberIDs(ctx context.Context, q Querier, cohortID string) ([]string, error) {
	if cohortID == "" {
		return nil, nil
	}
	rows, err := q.Query(ctx, `
		SELECT user_id::text FROM show_cohort_assignments WHERE cohort_id = $1
	`, cohortID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func resolveDirectorPlusIDs(ctx context.Context, q Querier, sessionID string) ([]string, error) {
	if sessionID == "" {
		return nil, nil
	}
	rows, err := q.Query(ctx, `
		SELECT user_id::text FROM session_participants
		WHERE session_id = $1 AND role::text IN ('director', 'producer')
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

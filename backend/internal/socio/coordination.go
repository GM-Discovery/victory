package socio

// Kernel 88 §12: Socio's authority-checked adapter around the generic
// venuecoordination.Registry (Kernel 83) for Current Turn -- the same
// pattern storyboards/coordination.go already is for Storyboards. Only
// Current Turn is wired here; the spec does not ask for Group Leader in
// Socio. Unlike Storyboards (which validates assignment targets against a
// live "who is watching the board" WS roster via hub.BoardWatcherUserIDs),
// Socio has no equivalent live-presence primitive, so targets are validated
// against Cohort roster membership instead (cohorts package), and sessions
// are started lazily on first read rather than on a "first watcher" WS
// hook -- there is no Socio analog of Storyboards' watch_board event to
// hang that hook on, and none is needed for this kernel's scope.

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
	"victory/backend/internal/venuecoordination"
)

// ErrCoordinationTargetNotInCohort mirrors storyboards'
// ErrCoordinationTargetNotInSession: the requested target must actually
// belong to the Cohort (or Ungrouped) whose Current Turn is being set.
var ErrCoordinationTargetNotInCohort = errors.New("coordination_target_not_in_cohort")

// SocioVenueSessionID is the venuecoordination session identity for one
// Cohort's Current Turn within a Show. Game Status is already Cohort-scoped
// (BuildGameStatusForCohort), so Current Turn is too; cohortID may be the
// literal "ungrouped" sentinel, same as everywhere else in this package.
func SocioVenueSessionID(showID, cohortID string) string {
	return strings.TrimSpace(showID) + ":" + strings.TrimSpace(cohortID)
}

// CoordinationView is the wire shape of live Current Turn state for one
// Cohort.
type CoordinationView struct {
	Active                     bool   `json:"active"`
	CurrentTurnUserID          string `json:"current_turn_user_id,omitempty"`
	CurrentTurnCharacterCardID string `json:"current_turn_character_card_id,omitempty"`
	CurrentTurnCharacterName   string `json:"current_turn_character_name,omitempty"`
}

// requireShowManageAuthority is the Director+/Producer/Operator check with
// no Character/roster context -- the part of requireShowCharacterAuthority
// (state.go) that doesn't depend on a specific characterCardID.
func requireShowManageAuthority(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID string) error {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return errors.New("not_authenticated")
	}
	s, err := shows.LoadShowByID(ctx, pool, showID)
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
	return nil
}

// resolveCohortIDForUser mirrors rollaudience.resolveCohortIDForActor's
// query shape (small, deliberately duplicated per this codebase's
// per-package convention for this exact kind of lookup rather than
// importing a larger authority-gated package function).
func resolveCohortIDForUser(ctx context.Context, pool *pgxpool.Pool, showID, userID string) (string, error) {
	var cohortID *string
	err := pool.QueryRow(ctx, `
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

// isActivePlayerOnShow mirrors cohorts' unexported helper of the same name
// -- whether userID currently holds an active Player roster row on showID's
// parent Show Run.
func isActivePlayerOnShow(ctx context.Context, pool *pgxpool.Pool, showID, userID string) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM show_run_roster_members rm
			JOIN shows s ON s.show_run_id = rm.show_run_id
			WHERE s.id = $1 AND rm.user_id = $2 AND rm.role = 'player' AND rm.removed_at IS NULL
		)
	`, showID, userID).Scan(&ok)
	return ok, err
}

// userInCohortOrUngrouped reports whether userID currently belongs to
// cohortID ("ungrouped" meaning: an active Player on the Show with no
// cohort assignment).
func userInCohortOrUngrouped(ctx context.Context, pool *pgxpool.Pool, showID, cohortID, userID string) (bool, error) {
	actual, err := resolveCohortIDForUser(ctx, pool, showID, userID)
	if err != nil {
		return false, err
	}
	if cohortID == "ungrouped" {
		if actual != "" {
			return false, nil
		}
		return isActivePlayerOnShow(ctx, pool, showID, userID)
	}
	return actual == cohortID, nil
}

// AssignCurrentTurn assigns Current Turn to targetUserID within a Cohort.
// Authority: Director+/Producer/Operator, or the current Current Turn
// holder passing it onward (mirrors storyboards.AssignCurrentTurn's "the
// current holder may pass it on" rule). targetUserID must actually belong
// to this Cohort (or Ungrouped).
func AssignCurrentTurn(ctx context.Context, pool *pgxpool.Pool, reg *venuecoordination.Registry, actorUserID, showID, cohortID, targetUserID string) (venuecoordination.State, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" {
		return venuecoordination.State{}, errors.New("target_user_id_required")
	}

	inCohort, err := userInCohortOrUngrouped(ctx, pool, showID, cohortID, targetUserID)
	if err != nil {
		return venuecoordination.State{}, err
	}
	if !inCohort {
		return venuecoordination.State{}, ErrCoordinationTargetNotInCohort
	}

	sessionID := SocioVenueSessionID(showID, cohortID)
	state, active := reg.Get(sessionID)
	if !active {
		state = reg.EnsureSession(sessionID, "socio", cohortID, "")
	}

	authErr := requireShowManageAuthority(ctx, pool, actorUserID, showID)
	authorized := authErr == nil
	if !authorized && state.CurrentTurnUserID != "" && state.CurrentTurnUserID == actorUserID {
		authorized = true
	}
	if !authorized {
		return venuecoordination.State{}, errors.New("not_authorized")
	}

	newState, ok := reg.SetCurrentTurn(sessionID, targetUserID)
	if !ok {
		return venuecoordination.State{}, errors.New("coordination_session_not_active")
	}
	return newState, nil
}

// BuildCoordinationView resolves live Current Turn state for a Cohort into
// the wire shape, including the Character bound to that user on this Show
// (if any) so the frontend can highlight the right seat without a second
// round trip. Lazily starts the coordination session on first read -- there
// is no explicit "session start" event to hook for Socio (see file header).
func BuildCoordinationView(ctx context.Context, pool *pgxpool.Pool, reg *venuecoordination.Registry, showID, cohortID string) (CoordinationView, error) {
	view := CoordinationView{}
	if reg == nil {
		return view, nil
	}
	sessionID := SocioVenueSessionID(showID, cohortID)
	state, active := reg.Get(sessionID)
	if !active {
		state = reg.EnsureSession(sessionID, "socio", cohortID, "")
	}
	view.Active = true
	if state.CurrentTurnUserID == "" {
		return view, nil
	}
	view.CurrentTurnUserID = state.CurrentTurnUserID

	s, err := shows.LoadShowByID(ctx, pool, showID)
	if err != nil {
		return view, err
	}
	var charID, charName *string
	err = pool.QueryRow(ctx, `
		SELECT rm.character_card_id::text, cc.name
		FROM show_run_roster_members rm
		LEFT JOIN character_cards cc ON cc.id = rm.character_card_id
		WHERE rm.show_run_id = $1 AND rm.user_id = $2 AND rm.removed_at IS NULL
	`, s.ShowRunID, state.CurrentTurnUserID).Scan(&charID, &charName)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return view, err
	}
	if charID != nil {
		view.CurrentTurnCharacterCardID = *charID
	}
	if charName != nil {
		view.CurrentTurnCharacterName = *charName
	}
	return view, nil
}

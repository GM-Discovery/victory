package drawing

// Turn/Leader handoff for the main stage, reusing Kernel 83's generic
// venuecoordination.Registry exactly the way storyboards/coordination.go
// does for boards -- see StageVenueSessionID's doc comment. This file is
// the "venue-specific" half: what counts as a valid target (a current
// session_participants row, not a live-watcher set, since the main stage
// tracks participation as durable rows rather than storyboards' pure
// live-watcher model) and who may assign (Director+, or the current
// holder passing their own role onward).

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/rollaudience"
	"victory/backend/internal/venuecoordination"
)

var ErrCoordinationTargetNotInSession = errors.New("coordination_target_not_in_session")

func isSessionParticipant(ctx context.Context, pool *pgxpool.Pool, sessionID, userID string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM session_participants WHERE session_id = $1 AND user_id = $2)
	`, sessionID, userID).Scan(&exists)
	return exists, err
}

func assignStageRole(ctx context.Context, pool *pgxpool.Pool, reg *venuecoordination.Registry, sessionID, showID, actorUserID, targetUserID string, isLeaderAssignment bool) (venuecoordination.State, error) {
	sessionID = strings.TrimSpace(sessionID)
	actorUserID = strings.TrimSpace(actorUserID)
	targetUserID = strings.TrimSpace(targetUserID)
	if reg == nil || sessionID == "" || targetUserID == "" {
		return venuecoordination.State{}, ErrCoordinationTargetNotInSession
	}

	targetOK, err := isSessionParticipant(ctx, pool, sessionID, targetUserID)
	if err != nil {
		return venuecoordination.State{}, err
	}
	if !targetOK {
		return venuecoordination.State{}, ErrCoordinationTargetNotInSession
	}

	reg.EnsureSession(StageVenueSessionID(sessionID), "stage", showID, "")
	state, _ := reg.Get(StageVenueSessionID(sessionID))

	isDirector, err := rollaudience.IsDirectorPlus(ctx, pool, sessionID, actorUserID)
	if err != nil {
		return venuecoordination.State{}, err
	}
	authorized := isDirector
	if !authorized && state.GroupLeaderUserID != "" && state.GroupLeaderUserID == actorUserID {
		authorized = true
	}
	if !authorized && !isLeaderAssignment && state.CurrentTurnUserID != "" && state.CurrentTurnUserID == actorUserID {
		authorized = true
	}
	if !authorized {
		return venuecoordination.State{}, errors.New("not_authorized")
	}

	var newState venuecoordination.State
	var ok bool
	if isLeaderAssignment {
		newState, ok = reg.SetGroupLeader(StageVenueSessionID(sessionID), targetUserID)
	} else {
		newState, ok = reg.SetCurrentTurn(StageVenueSessionID(sessionID), targetUserID)
	}
	if !ok {
		return venuecoordination.State{}, errors.New("coordination_session_not_active")
	}
	return newState, nil
}

// AssignGroupLeader is the authority-checked entry point for making
// targetUserID Group Leader of sessionID's live stage.
func AssignGroupLeader(ctx context.Context, pool *pgxpool.Pool, reg *venuecoordination.Registry, sessionID, showID, actorUserID, targetUserID string) (venuecoordination.State, error) {
	return assignStageRole(ctx, pool, reg, sessionID, showID, actorUserID, targetUserID, true)
}

// AssignCurrentTurn is the authority-checked entry point for handing
// Current Turn to targetUserID on sessionID's live stage.
func AssignCurrentTurn(ctx context.Context, pool *pgxpool.Pool, reg *venuecoordination.Registry, sessionID, showID, actorUserID, targetUserID string) (venuecoordination.State, error) {
	return assignStageRole(ctx, pool, reg, sessionID, showID, actorUserID, targetUserID, false)
}

// GetCoordination returns the current Group Leader/Current Turn state for
// sessionID, initializing an inactive session to an empty one (both
// unset) rather than erroring -- a fresh session simply has no leader/turn
// assigned yet.
func GetCoordination(reg *venuecoordination.Registry, sessionID, showID string) venuecoordination.State {
	if reg == nil {
		return venuecoordination.State{VenueSessionID: sessionID}
	}
	reg.EnsureSession(StageVenueSessionID(sessionID), "stage", showID, "")
	state, _ := reg.Get(StageVenueSessionID(sessionID))
	return state
}

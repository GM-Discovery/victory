package storyboards

// Kernel 83: Storyboards' authority-checked wrapper around the generic
// venuecoordination.Registry. This file owns everything venue-specific --
// what a Storyboards "venue session" identity is, who is allowed to assign
// Group Leader/Current Turn, and how to shape the roster/state for the
// wire -- while venuecoordination itself stays a dumb, reusable, in-memory
// state store with no notion of boards, tiers, or grants. See
// Construction/Venues/collaborative-venue-coordination-contract.md.

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/network"
	"victory/backend/internal/venuecoordination"
)

var (
	// ErrCoordinationSessionNotActive means no one is currently watching
	// this board -- there is nothing to read or mutate.
	ErrCoordinationSessionNotActive = errors.New("coordination_session_not_active")
	// ErrCoordinationTargetNotInSession means the requested target is not
	// among the board's current live watchers (spec 2.3: assignment
	// targets must belong to the session, never an arbitrary account ID).
	ErrCoordinationTargetNotInSession = errors.New("coordination_target_not_in_session")
)

// StoryboardVenueSessionID is the venuecoordination session identity for a
// live Storyboards board: the board_id itself. Storyboards has exactly one
// live coordination session per board at a time, scoped to whoever
// currently has it open via watch_board -- there is no separate
// session/production concept for this venue (Kernel 80's ws.go). See
// venue-session-state-lifecycle.md for why this mapping was chosen.
func StoryboardVenueSessionID(boardID string) string { return strings.TrimSpace(boardID) }

// PresenceEntry is one currently-connected watcher of a board, shaped for
// the Presence Tray.
type PresenceEntry struct {
	UserID      string `json:"user_id"`
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name"`
	Tier        string `json:"tier"`
}

// CoordinationView is the wire shape of live Group Leader/Current Turn
// state for one board, resolved with enough identity to render even when
// the assigned user is no longer present (spec 1.8: disconnecting keeps
// the assignment, Presence just marks them absent).
type CoordinationView struct {
	Active                 bool   `json:"active"`
	GroupLeaderUserID      string `json:"group_leader_user_id,omitempty"`
	GroupLeaderHandle      string `json:"group_leader_handle,omitempty"`
	GroupLeaderDisplayName string `json:"group_leader_display_name,omitempty"`
	GroupLeaderPresent     bool   `json:"group_leader_present"`
	CurrentTurnUserID      string `json:"current_turn_user_id,omitempty"`
	CurrentTurnHandle      string `json:"current_turn_handle,omitempty"`
	CurrentTurnDisplayName string `json:"current_turn_display_name,omitempty"`
	CurrentTurnPresent     bool   `json:"current_turn_present"`
}

func containsUser(ids []string, userID string) bool {
	for _, id := range ids {
		if id == userID {
			return true
		}
	}
	return false
}

// startCoordinationSessionIfFirstWatcher is called by ws.go exactly once,
// immediately after a connection becomes the first (by distinct user)
// watcher of board.ID. initiatingUserID is that connecting user -- Group
// Leader initializes to the board owner only when the owner is literally
// the one causing session start (spec 1.3: "when the owner is available");
// a non-owner opening the board first leaves Group Leader unset, and the
// owner joining moments later must not retroactively seize it (spec 5.2 --
// EnsureSession is a no-op once a session is already active).
func startCoordinationSessionIfFirstWatcher(reg *venuecoordination.Registry, board *Storyboard, initiatingUserID string) {
	if reg == nil || board == nil {
		return
	}
	leader := ""
	if board.OwnerUserID != "" && board.OwnerUserID == initiatingUserID {
		leader = initiatingUserID
	}
	reg.EnsureSession(StoryboardVenueSessionID(board.ID), "storyboards", board.ID, leader)
}

// endCoordinationSessionIfLastWatcherLeft is called by ws.go right after a
// disconnect/board-switch leaves boardID with zero distinct watchers.
func endCoordinationSessionIfLastWatcherLeft(reg *venuecoordination.Registry, boardID string) {
	if reg == nil {
		return
	}
	reg.EndSession(StoryboardVenueSessionID(boardID))
}

// AssignGroupLeader is the authority-checked entry point for "Make Group
// Leader" (spec 2.1): Director+/owner/Operator, or the current Group
// Leader passing leadership onward, may assign any user currently watching
// the board (including themselves -- spec 2.4, no separate "take
// leadership" action). hub supplies the live watcher roster used to
// confirm the session is active and to validate actor/target membership.
func AssignGroupLeader(ctx context.Context, pool *pgxpool.Pool, hub *network.Hub, reg *venuecoordination.Registry, board *Storyboard, actorUserID, targetUserID string) (venuecoordination.State, error) {
	return assignCoordinationRole(ctx, pool, hub, reg, board, actorUserID, targetUserID, true)
}

// AssignCurrentTurn is the authority-checked entry point for "Give Turn"
// (spec 2.2): Director+/owner/Operator, the current Group Leader, or the
// current Current Turn holder passing the turn onward, may assign it.
func AssignCurrentTurn(ctx context.Context, pool *pgxpool.Pool, hub *network.Hub, reg *venuecoordination.Registry, board *Storyboard, actorUserID, targetUserID string) (venuecoordination.State, error) {
	return assignCoordinationRole(ctx, pool, hub, reg, board, actorUserID, targetUserID, false)
}

func assignCoordinationRole(ctx context.Context, pool *pgxpool.Pool, hub *network.Hub, reg *venuecoordination.Registry, board *Storyboard, actorUserID, targetUserID string, isLeaderAssignment bool) (venuecoordination.State, error) {
	if board == nil || hub == nil || reg == nil {
		return venuecoordination.State{}, ErrCoordinationSessionNotActive
	}
	actorUserID = strings.TrimSpace(actorUserID)
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" {
		return venuecoordination.State{}, ErrCoordinationTargetNotInSession
	}

	sessionID := StoryboardVenueSessionID(board.ID)
	state, active := reg.Get(sessionID)
	if !active {
		return venuecoordination.State{}, ErrCoordinationSessionNotActive
	}

	watchers := hub.BoardWatcherUserIDs(board.ID)
	if !containsUser(watchers, actorUserID) {
		return venuecoordination.State{}, ErrNotAuthorized
	}
	if !containsUser(watchers, targetUserID) {
		return venuecoordination.State{}, ErrCoordinationTargetNotInSession
	}

	tier, err := resolveViewerTier(ctx, pool, actorUserID, board)
	if err != nil {
		return venuecoordination.State{}, err
	}

	authorized := tierAtLeastDirector(tier)
	if !authorized && state.GroupLeaderUserID != "" && state.GroupLeaderUserID == actorUserID {
		authorized = true
	}
	if !authorized && !isLeaderAssignment && state.CurrentTurnUserID != "" && state.CurrentTurnUserID == actorUserID {
		authorized = true
	}
	if !authorized {
		return venuecoordination.State{}, ErrNotAuthorized
	}

	var newState venuecoordination.State
	var ok bool
	if isLeaderAssignment {
		newState, ok = reg.SetGroupLeader(sessionID, targetUserID)
	} else {
		newState, ok = reg.SetCurrentTurn(sessionID, targetUserID)
	}
	if !ok {
		return venuecoordination.State{}, ErrCoordinationSessionNotActive
	}
	return newState, nil
}

// userIdentitySummary is a minimal (handle, display_name) pair used to
// render presence/coordination without a second N+1 query per caller.
type userIdentitySummary struct {
	Handle      string
	DisplayName string
}

func loadUserIdentities(ctx context.Context, pool *pgxpool.Pool, userIDs []string) (map[string]userIdentitySummary, error) {
	out := map[string]userIdentitySummary{}
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT id::text, handle, display_name FROM users WHERE id = ANY($1::uuid[])
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, handle, displayName string
		if err := rows.Scan(&id, &handle, &displayName); err != nil {
			return nil, err
		}
		out[id] = userIdentitySummary{Handle: handle, DisplayName: displayName}
	}
	return out, rows.Err()
}

// BuildPresenceRoster resolves the current live watchers of board.ID into
// Presence Tray entries, each with its own server-resolved viewer tier.
func BuildPresenceRoster(ctx context.Context, pool *pgxpool.Pool, hub *network.Hub, board *Storyboard) ([]PresenceEntry, error) {
	if hub == nil || board == nil {
		return []PresenceEntry{}, nil
	}
	watchers := hub.BoardWatcherUserIDs(board.ID)
	identities, err := loadUserIdentities(ctx, pool, watchers)
	if err != nil {
		return nil, err
	}
	out := make([]PresenceEntry, 0, len(watchers))
	for _, userID := range watchers {
		tier, err := resolveViewerTier(ctx, pool, userID, board)
		if err != nil {
			continue
		}
		if tier == TierNone {
			continue
		}
		id := identities[userID]
		out = append(out, PresenceEntry{
			UserID:      userID,
			Handle:      id.Handle,
			DisplayName: id.DisplayName,
			Tier:        tier,
		})
	}
	// Sorted by handle (then user_id as a tiebreak) rather than left in
	// hub.BoardWatcherUserIDs' Go-map iteration order -- spec 3.4 requires
	// the Presence Tray never appear to reorder itself, which a
	// nondeterministic roster order would look like on every re-render
	// even though nothing about turn/leader state is driving it.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Handle != out[j].Handle {
			return out[i].Handle < out[j].Handle
		}
		return out[i].UserID < out[j].UserID
	})
	return out, nil
}

// BuildCoordinationView resolves the live venuecoordination.State for
// board.ID (if any) into the wire shape, including identity for the
// leader/turn-holder even when they are no longer present.
func BuildCoordinationView(ctx context.Context, pool *pgxpool.Pool, hub *network.Hub, reg *venuecoordination.Registry, board *Storyboard) (CoordinationView, error) {
	view := CoordinationView{}
	if reg == nil || board == nil {
		return view, nil
	}
	state, active := reg.Get(StoryboardVenueSessionID(board.ID))
	view.Active = active
	if !active {
		return view, nil
	}

	var watchers []string
	if hub != nil {
		watchers = hub.BoardWatcherUserIDs(board.ID)
	}

	var lookupIDs []string
	if state.GroupLeaderUserID != "" {
		lookupIDs = append(lookupIDs, state.GroupLeaderUserID)
	}
	if state.CurrentTurnUserID != "" {
		lookupIDs = append(lookupIDs, state.CurrentTurnUserID)
	}
	identities, err := loadUserIdentities(ctx, pool, lookupIDs)
	if err != nil {
		return view, err
	}

	if state.GroupLeaderUserID != "" {
		view.GroupLeaderUserID = state.GroupLeaderUserID
		id := identities[state.GroupLeaderUserID]
		view.GroupLeaderHandle = id.Handle
		view.GroupLeaderDisplayName = id.DisplayName
		view.GroupLeaderPresent = containsUser(watchers, state.GroupLeaderUserID)
	}
	if state.CurrentTurnUserID != "" {
		view.CurrentTurnUserID = state.CurrentTurnUserID
		id := identities[state.CurrentTurnUserID]
		view.CurrentTurnHandle = id.Handle
		view.CurrentTurnDisplayName = id.DisplayName
		view.CurrentTurnPresent = containsUser(watchers, state.CurrentTurnUserID)
	}
	return view, nil
}

// Package venuecoordination implements Kernel 83: a generic, venue-agnostic,
// in-memory store for two ephemeral live-session coordination signals --
// Group Leader and Current Turn. It knows nothing about Victory roles,
// permissions, or any particular venue's authority model; it only tracks
// "which user, if any, currently holds each state for this venue session"
// and enforces the lifecycle rules that are true for every venue (session
// start initializes once, session end clears, nothing persists). Authority
// (who is allowed to call SetGroupLeader/SetCurrentTurn), target validation
// (is the target actually part of this session), and venue-session identity
// derivation are all the calling venue package's responsibility -- see
// storyboards/coordination.go for Storyboards' authority-checked wrapper.
package venuecoordination

import (
	"strings"
	"sync"
)

// State is the canonical coordination state for one active collaborative
// venue session. GroupLeaderUserID/CurrentTurnUserID are empty when unset.
type State struct {
	VenueSessionID    string `json:"venue_session_id"`
	VenueType         string `json:"venue_type"`
	VenueInstanceID   string `json:"venue_instance_id"`
	GroupLeaderUserID string `json:"group_leader_user_id,omitempty"`
	CurrentTurnUserID string `json:"current_turn_user_id,omitempty"`
}

// Registry holds one State per active venue session, entirely in memory.
// This is deliberate (spec 4.2, 5.4): coordination state is ephemeral
// live-session state, never durable Storyboard/campaign/profile state, so
// there is no table backing it and nothing to migrate.
type Registry struct {
	mu       sync.RWMutex
	sessions map[string]*State
}

func NewRegistry() *Registry {
	return &Registry{sessions: make(map[string]*State)}
}

// EnsureSession starts a session if one is not already active for
// venueSessionID, initializing Group Leader to initialLeaderUserID (which
// the caller has already decided is "the owner, if the owner is the one
// causing session start" -- pass "" when the owner isn't applicable/present,
// per spec 1.3). Current Turn always begins unset (spec 1.4). If a session
// is already active, this is a no-op that returns the existing state
// unchanged -- a later participant joining must never re-initialize or
// seize leadership (spec 5.2).
func (r *Registry) EnsureSession(venueSessionID, venueType, venueInstanceID, initialLeaderUserID string) State {
	venueSessionID = strings.TrimSpace(venueSessionID)

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.sessions[venueSessionID]; ok {
		return *existing
	}

	s := &State{
		VenueSessionID:    venueSessionID,
		VenueType:         venueType,
		VenueInstanceID:   venueInstanceID,
		GroupLeaderUserID: strings.TrimSpace(initialLeaderUserID),
	}
	r.sessions[venueSessionID] = s
	return *s
}

// EndSession clears all coordination state for venueSessionID. A later
// EnsureSession call for the same ID starts completely fresh (spec 1.5,
// 5.4) -- nothing here is retained across the call.
func (r *Registry) EndSession(venueSessionID string) {
	venueSessionID = strings.TrimSpace(venueSessionID)

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, venueSessionID)
}

// Get returns the current state for venueSessionID, and false if no
// session is active (e.g. it never started, or already ended).
func (r *Registry) Get(venueSessionID string) (State, bool) {
	venueSessionID = strings.TrimSpace(venueSessionID)

	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sessions[venueSessionID]
	if !ok {
		return State{}, false
	}
	return *s, true
}

// SetGroupLeader assigns Group Leader to targetUserID (may equal the
// existing leader, in which case this is a harmless no-op -- spec 4.5
// idempotency). Returns the resulting state and false if no session is
// active for venueSessionID. Authority is entirely the caller's job.
func (r *Registry) SetGroupLeader(venueSessionID, targetUserID string) (State, bool) {
	venueSessionID = strings.TrimSpace(venueSessionID)
	targetUserID = strings.TrimSpace(targetUserID)

	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[venueSessionID]
	if !ok {
		return State{}, false
	}
	s.GroupLeaderUserID = targetUserID
	return *s, true
}

// SetCurrentTurn assigns Current Turn to targetUserID. Same idempotency and
// authority contract as SetGroupLeader.
func (r *Registry) SetCurrentTurn(venueSessionID, targetUserID string) (State, bool) {
	venueSessionID = strings.TrimSpace(venueSessionID)
	targetUserID = strings.TrimSpace(targetUserID)

	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[venueSessionID]
	if !ok {
		return State{}, false
	}
	s.CurrentTurnUserID = targetUserID
	return *s, true
}

package network

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type PresenceUser struct {
	UserID      string `json:"user_id"`
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Persona     any    `json:"persona"`
	ConnectedAt string `json:"connected_at"`
}

type PresenceRegistry struct {
	mu       sync.RWMutex
	sessions map[string]map[string]*presenceEntry
}

type presenceEntry struct {
	user        PresenceUser
	connections int
}

func NewPresenceRegistry() *PresenceRegistry {
	return &PresenceRegistry{
		sessions: make(map[string]map[string]*presenceEntry),
	}
}

func (r *PresenceRegistry) Connect(sessionID string, user PresenceUser) (snapshot []PresenceUser, joined bool) {
	sessionID = strings.TrimSpace(sessionID)
	user.UserID = strings.TrimSpace(user.UserID)
	user.Handle = strings.TrimSpace(user.Handle)
	user.DisplayName = strings.TrimSpace(user.DisplayName)
	user.Role = strings.TrimSpace(user.Role)

	if user.ConnectedAt == "" {
		user.ConnectedAt = time.Now().UTC().Format(time.RFC3339)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.sessions[sessionID]; !ok {
		r.sessions[sessionID] = make(map[string]*presenceEntry)
	}

	entry, ok := r.sessions[sessionID][user.UserID]
	if ok {
		entry.connections++
		entry.user.Handle = user.Handle
		entry.user.DisplayName = user.DisplayName
		entry.user.Role = user.Role
		if entry.user.ConnectedAt == "" {
			entry.user.ConnectedAt = user.ConnectedAt
		}
		return r.snapshotLocked(sessionID), false
	}

	r.sessions[sessionID][user.UserID] = &presenceEntry{
		user:        user,
		connections: 1,
	}

	return r.snapshotLocked(sessionID), true
}

func (r *PresenceRegistry) Disconnect(sessionID, userID string) (snapshot []PresenceUser, left bool) {
	sessionID = strings.TrimSpace(sessionID)
	userID = strings.TrimSpace(userID)

	r.mu.Lock()
	defer r.mu.Unlock()

	entries, ok := r.sessions[sessionID]
	if !ok {
		return nil, false
	}

	entry, ok := entries[userID]
	if !ok {
		return r.snapshotLocked(sessionID), false
	}

	entry.connections--
	if entry.connections > 0 {
		return r.snapshotLocked(sessionID), false
	}

	delete(entries, userID)
	if len(entries) == 0 {
		delete(r.sessions, sessionID)
	}

	return r.snapshotLocked(sessionID), true
}

func (r *PresenceRegistry) Snapshot(sessionID string) []PresenceUser {
	sessionID = strings.TrimSpace(sessionID)

	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshotLocked(sessionID)
}

func (r *PresenceRegistry) snapshotLocked(sessionID string) []PresenceUser {
	entries := r.sessions[sessionID]
	if len(entries) == 0 {
		return []PresenceUser{}
	}

	out := make([]PresenceUser, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.user)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ConnectedAt == out[j].ConnectedAt {
			return out[i].DisplayName < out[j].DisplayName
		}
		return out[i].ConnectedAt < out[j].ConnectedAt
	})

	return out
}

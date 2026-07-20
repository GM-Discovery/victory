package network

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn      *websocket.Conn
	Send      chan []byte
	UserID    string
	SessionID string
	Presence  PresenceUser

	// WatchingProfileID is set by a client on the lightweight player-profile
	// websocket (ServeProfileWS) to declare which Player Workbook/Face it
	// wants `player_profile/projection_updated` invalidations for. Empty
	// means "not watching anything." Owner tabs and cross-user viewer tabs
	// use the exact same mechanism -- both just watch a profile_id (Kernel
	// 61A §9.1: "the smallest reusable delivery mechanism").
	WatchingProfileID string
}

type Hub struct {
	mu       sync.RWMutex
	clients  map[*Client]struct{}
	presence *PresenceRegistry
}

func NewHub() *Hub {
	return &Hub{
		clients:  make(map[*Client]struct{}),
		presence: NewPresenceRegistry(),
	}
}

func (h *Hub) Add(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}

func (h *Hub) Remove(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
	close(c.Send)
}

func (h *Hub) Broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		select {
		case c.Send <- msg:
		default:
			log.Printf("dropping slow websocket client")
		}
	}
}

func (h *Hub) BroadcastSession(sessionID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if c == nil || c.SessionID != sessionID {
			continue
		}
		select {
		case c.Send <- msg:
		default:
			log.Printf("dropping slow websocket client")
		}
	}
}

func (h *Hub) UpdateClientPresence(sessionID, userID string, persona any) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for c := range h.clients {
		if c.SessionID == sessionID && c.UserID == userID {
			c.Presence.Persona = persona
		}
	}
}

// SetClientWatchProfile records which profile_id a player-profile websocket
// client wants invalidations for. Mutating the field under the hub's lock
// (rather than letting the client's own goroutine write it unguarded)
// matches UpdateClientPresence's convention -- the hub's mutex protects
// per-client fields that broadcast methods read, not just the clients map.
func (h *Hub) SetClientWatchProfile(c *Client, profileID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	c.WatchingProfileID = profileID
}

// BroadcastProfileWatchers sends msg to every client currently watching
// profileID -- both the owner's own other tabs and any other authenticated
// user's Trailer-viewer tab land in the same set, since both watch by the
// same profile_id (Kernel 61A §9.1).
func (h *Hub) BroadcastProfileWatchers(profileID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if c == nil || c.WatchingProfileID == "" || c.WatchingProfileID != profileID {
			continue
		}
		select {
		case c.Send <- msg:
		default:
			log.Printf("dropping slow websocket client")
		}
	}
}

// BroadcastToSessionUser sends msg only to clients connected to sessionID
// AND authenticated as userID -- the Kernel 73 privacy requirement that
// Program Panel state (stance/Haggle rolls, purchase confirmations) never
// reaches other Players or Audience (spec S6.2, S12). Reuses the Client
// fields BroadcastSession and UpdateClientPresence already filter on
// (SessionID, UserID); no new per-client "watching" field is needed since,
// unlike the Kernel 61A player-profile WS (a separate lightweight endpoint
// with no venue/session context), the existing venue WS connection already
// carries both.
func (h *Hub) BroadcastToSessionUser(sessionID, userID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if c == nil || c.SessionID != sessionID || c.UserID != userID {
			continue
		}
		select {
		case c.Send <- msg:
		default:
			log.Printf("dropping slow websocket client")
		}
	}
}

func (h *Hub) Presence() *PresenceRegistry {
	return h.presence
}

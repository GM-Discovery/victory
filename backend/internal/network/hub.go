package network

import (
	"context"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
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

	// WatchingBoardID is set by a client on the lightweight Storyboards
	// websocket (storyboards.ServeStoryboardWS, Kernel 80) to declare which
	// board it wants live events for. Same "smallest reusable delivery
	// mechanism" idiom as WatchingProfileID: one board watched per
	// connection at a time, a client wanting a different board sends a new
	// watch_board message to re-point the same connection rather than the
	// hub tracking a set.
	WatchingBoardID string
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

// RevalidateSessions closes any connected client whose user no longer holds
// at least one non-revoked, non-expired auth session. Kernel 77 K77-05:
// revoking a session (logout, password reset, account deletion) previously
// only prevented a *new* WebSocket connection -- an already-open one kept
// working until it happened to disconnect on its own. Intended to run on a
// periodic ticker from main(), not per-request; a per-client DB lookup on
// every message would be needless load for something that only needs to
// catch up within a bounded window.
func (h *Hub) RevalidateSessions(ctx context.Context, pool *pgxpool.Pool) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	checked := map[string]bool{}
	for _, c := range clients {
		if c.UserID == "" {
			continue
		}
		stillValid, ok := checked[c.UserID]
		if !ok {
			err := pool.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM auth.sessions
					WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
				)
			`, c.UserID).Scan(&stillValid)
			if err != nil {
				// A transient DB error must not disconnect everyone; skip
				// this user this cycle and try again next tick.
				continue
			}
			checked[c.UserID] = stillValid
		}
		if !stillValid {
			_ = c.Conn.WriteJSON(map[string]any{"type": "error", "error": "session_revoked"})
			_ = c.Conn.Close()
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

// SetClientWatchBoard records which storyboard_id a Storyboards websocket
// client wants live events for (Kernel 80). Locked the same way
// SetClientWatchProfile is -- the hub's mutex protects per-client fields
// that broadcast methods read, not just the clients map.
func (h *Hub) SetClientWatchBoard(c *Client, boardID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	c.WatchingBoardID = boardID
}

// BoardWatcherUserIDs returns the distinct user IDs of every client
// currently watching boardID (a user may have more than one tab open on
// the same board). storyboards/events.go uses this to resolve each
// watcher's viewer tier before deciding what, if anything, to send them.
func (h *Hub) BoardWatcherUserIDs(boardID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	seen := map[string]bool{}
	var out []string
	for c := range h.clients {
		if c == nil || c.WatchingBoardID == "" || c.WatchingBoardID != boardID || c.UserID == "" {
			continue
		}
		if seen[c.UserID] {
			continue
		}
		seen[c.UserID] = true
		out = append(out, c.UserID)
	}
	return out
}

// SendToBoardWatcher sends msg only to connections for userID currently
// watching boardID (multi-tab safe) -- the per-viewer delivery primitive
// Storyboards live events (Kernel 80) build their hidden-card fan-out on,
// mirroring BroadcastToSessionUser's shape for (board, user) instead of
// (session, user). The Hub itself does no role/visibility filtering here
// either, per this package's standing invariant -- storyboards/events.go is
// what decides, per watcher, whether msg should exist at all before
// calling this.
func (h *Hub) SendToBoardWatcher(boardID, userID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if c == nil || c.WatchingBoardID != boardID || c.UserID != userID {
			continue
		}
		select {
		case c.Send <- msg:
		default:
			log.Printf("dropping slow websocket client")
		}
	}
}

// BroadcastBoardWatchers sends msg to every connection currently watching
// boardID, unfiltered (mirroring BroadcastProfileWatchers's shape for
// boards instead of profiles). Kernel 83 uses this for Presence Tray
// roster and Group Leader/Current Turn updates -- unlike card events
// (events.go), neither is ever hidden-from-audience content, so there is
// no per-viewer shaping to do before it reaches the Hub.
func (h *Hub) BroadcastBoardWatchers(boardID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if c == nil || c.WatchingBoardID == "" || c.WatchingBoardID != boardID {
			continue
		}
		select {
		case c.Send <- msg:
		default:
			log.Printf("dropping slow websocket client")
		}
	}
}

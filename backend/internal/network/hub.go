package network

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

type Client struct {
	Conn      *websocket.Conn
	Send      chan []byte
	UserID    string
	SessionID string
	Presence  PresenceUser

	// VenueSlug is the venue this connection was authorized for at connect
	// time (ServeVenueWS). Kernel 96: RevalidateSessions re-checks this
	// per-venue authorization, not just session validity -- a user removed
	// from a Show's roster or demoted mid-connection previously kept
	// receiving that venue's broadcasts until the socket happened to close
	// on its own. Empty on connection kinds that aren't venue-scoped (the
	// player-profile/Storyboards watch sockets), which skip that recheck.
	VenueSlug string

	// closeSendOnce guards close(Send). Kernel 88B gave the channel a second
	// closer -- the session-revocation path, which queues a final notice and
	// then needs writePump to drain it and shut down -- and closing a channel
	// twice panics.
	closeSendOnce sync.Once

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

// SendJSON queues a message for delivery to this one client.
//
// Kernel 88B: every write to a connection must go through the client's Send
// channel, which the single writePump goroutine drains. Handlers used to call
// c.Conn.WriteJSON directly from the read goroutine while writePump wrote from
// its own -- gorilla/websocket supports exactly one concurrent writer, so that
// was a latent "concurrent write to websocket connection" panic that only
// needed a broadcast to land while a handler was replying. Routing all writes
// through the one channel removes the race by construction rather than by
// timing luck.
//
// It also makes replies observable: a test can read Send, whereas a direct
// Conn write needs a live socket. That is what previously made every WS error
// path untestable while the success paths (already channel-based) were fine.
//
// Drop-on-full matches Hub.Broadcast: a client too slow to drain its buffer is
// dropped rather than allowed to block the read loop. Returns false when the
// message could not be queued.
func (c *Client) SendJSON(payload any) bool {
	if c == nil || c.Send == nil {
		return false
	}
	msg, err := json.Marshal(payload)
	if err != nil {
		log.Printf("websocket reply marshal failed: %v", err)
		return false
	}
	select {
	case c.Send <- msg:
		return true
	default:
		log.Printf("dropping slow websocket client")
		return false
	}
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
	c.CloseSend()
}

// CloseSend closes the client's outbound channel exactly once. writePump
// drains whatever is still buffered before exiting, so a message queued
// immediately before this call is still delivered, and writePump's deferred
// Conn.Close is what actually tears the socket down.
func (c *Client) CloseSend() {
	if c == nil || c.Send == nil {
		return
	}
	c.closeSendOnce.Do(func() { close(c.Send) })
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
	// Kernel 96: a user's session can stay perfectly valid while their
	// AUTHORITY over one specific venue changes mid-connection (removed
	// from a Show's roster, demoted). That's a separate question from
	// session validity, and two connections for the same user can be
	// authorized for two different venues at once (two tabs, two Shows) --
	// cached per (userID, venueSlug), not per userID.
	venueChecked := map[[2]string]bool{}
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
			// Queue the notice, then close the channel rather than the socket:
			// writePump drains the buffer first, so the client learns *why* it
			// was disconnected instead of just seeing the socket drop. Closing
			// Conn directly here would race writePump for the connection's
			// single writer slot, which is the bug this pass removes.
			c.SendJSON(map[string]any{"type": "error", "error": "session_revoked"})
			c.CloseSend()
			continue
		}

		if c.VenueSlug == "" {
			continue
		}
		key := [2]string{c.UserID, c.VenueSlug}
		stillAuthorized, ok := venueChecked[key]
		if !ok {
			var err error
			stillAuthorized, err = access.UserCanAccessVenueSlug(ctx, pool, c.UserID, c.VenueSlug)
			if err != nil {
				continue
			}
			venueChecked[key] = stillAuthorized
		}
		if !stillAuthorized {
			c.SendJSON(map[string]any{"type": "error", "error": "venue_access_revoked"})
			c.CloseSend()
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

// SendToUsers sends msg to every client connected to sessionID whose
// UserID is in userIDs (multi-tab safe, one pass over the client set) --
// the Kernel 86 targeted-delivery primitive for Cohort/Director/Private
// roll projections, extending BroadcastToSessionUser's single-user shape to
// a caller-resolved recipient set. The Hub does no audience/role resolution
// of its own here either, per this package's standing invariant: callers
// (backend/internal/rollaudience, backend/internal/network/ws.go) decide
// who belongs in userIDs before this is ever called.
func (h *Hub) SendToUsers(sessionID string, userIDs []string, msg []byte) {
	if len(userIDs) == 0 {
		return
	}
	want := make(map[string]struct{}, len(userIDs))
	for _, id := range userIDs {
		if id != "" {
			want[id] = struct{}{}
		}
	}
	if len(want) == 0 {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if c == nil || c.SessionID != sessionID {
			continue
		}
		if _, ok := want[c.UserID]; !ok {
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

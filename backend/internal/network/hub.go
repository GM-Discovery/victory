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

func (h *Hub) UpdateClientPresence(sessionID, userID string, persona any) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for c := range h.clients {
		if c.SessionID == sessionID && c.UserID == userID {
			c.Presence.Persona = persona
		}
	}
}

func (h *Hub) Presence() *PresenceRegistry {
	return h.presence
}

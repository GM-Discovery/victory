package network

// Small exported wrappers around this package's internal WebSocket
// plumbing (the upgrader, the outbound write pump, and the connection/
// message rate limiters), for lightweight endpoints that must live outside
// this package -- e.g. storyboards.ServeStoryboardWS (Kernel 80), which
// needs the storyboards package's own CanViewBoard check before accepting
// a watch_board message and therefore cannot live in network without
// network importing a feature package (a dependency direction this
// package must never take; see profile_ws.go's package comment). Nothing
// here changes behavior for the existing venue/profile sockets -- these
// just expose the same upgrader/writePump/checkMessageRate a second way.

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// Upgrade upgrades an HTTP request to a WebSocket connection using this
// package's shared upgrader (same origin-check policy as every other
// Victory socket).
func Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return upgrader.Upgrade(w, r, nil)
}

// WritePump drains c.Send to the client's connection until it closes.
// Callers must run this in its own goroutine, exactly like the venue and
// profile sockets do.
func WritePump(c *Client) {
	writePump(c)
}

// AllowNewConnection reports whether ip may open another new WebSocket
// connection right now (shared wsConnectionLimiter).
func AllowNewConnection(ip string) bool {
	return wsConnectionLimiter.Allow(ip)
}

// AllowMessage reports whether userID may process another inbound
// WebSocket message right now (shared wsMessageLimiter).
func AllowMessage(userID string) bool {
	return checkMessageRate(userID)
}

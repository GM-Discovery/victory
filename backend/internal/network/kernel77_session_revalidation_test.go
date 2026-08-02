package network

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/sessions"
)

// TestRevalidateSessionsClosesConnectionAfterRevocation proves Kernel 77
// K77-05: an already-open WebSocket connection is torn down once its user's
// session is revoked, not just refused on the next connection attempt.
func TestRevalidateSessionsClosesConnectionAfterRevocation(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()

	handle := "wsrevoke_" + time.Now().UTC().Format("150405.000000")
	var userID string
	if err := pool.QueryRow(ctx, `INSERT INTO users (handle, display_name) VALUES ($1, $2) RETURNING id::text`, handle, "WS Revoke Test").Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID) })

	if _, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil)); err != nil {
		t.Fatalf("create session: %v", err)
	}

	hub := NewHub()
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client := &Client{Conn: conn, Send: make(chan []byte, 4), UserID: userID}
		hub.Add(client)
		defer hub.Remove(client)

		// A minimal read loop, standing in for the real readPump: blocks
		// until the connection errors (including a server-initiated Close,
		// which is exactly what RevalidateSessions performs).
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("client dial failed: %v", err)
	}
	defer clientConn.Close()

	// Give the server a moment to register the client.
	time.Sleep(100 * time.Millisecond)

	// Revoke the session -- the same call HandleLogout/HandleResetPassword/
	// account deletion already make.
	if err := sessions.RevokeAllUserSessions(ctx, pool, userID); err != nil {
		t.Fatalf("revoke sessions: %v", err)
	}

	revalidateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	hub.RevalidateSessions(revalidateCtx, pool)

	// The server sends one "session_revoked" notice before closing, so the
	// client's next reads are: the notice itself (a normal message, not an
	// error), then a close. Read until the connection actually errors, with
	// an overall deadline so a bug that never closes it fails the test
	// instead of hanging.
	clientConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	closed := false
	for i := 0; i < 5; i++ {
		if _, _, err = clientConn.ReadMessage(); err != nil {
			closed = true
			break
		}
	}
	if !closed {
		t.Fatal("expected the connection to be closed after session revocation, but it is still open")
	}
}

// TestRevalidateSessionsLeavesActiveSessionConnected is the negative case:
// a client whose session is still valid must not be disconnected by a
// revalidation sweep.
func TestRevalidateSessionsLeavesActiveSessionConnected(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()

	handle := "wsactive_" + time.Now().UTC().Format("150405.000000")
	var userID string
	if err := pool.QueryRow(ctx, `INSERT INTO users (handle, display_name) VALUES ($1, $2) RETURNING id::text`, handle, "WS Active Test").Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID) })

	if _, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil)); err != nil {
		t.Fatalf("create session: %v", err)
	}

	hub := NewHub()
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client := &Client{Conn: conn, Send: make(chan []byte, 4), UserID: userID}
		hub.Add(client)
		defer hub.Remove(client)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("client dial failed: %v", err)
	}
	defer clientConn.Close()

	time.Sleep(100 * time.Millisecond)

	revalidateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	hub.RevalidateSessions(revalidateCtx, pool)

	// Must still be alive: a short-deadline read should time out (no
	// message was sent), not fail with a close error.
	clientConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _, err = clientConn.ReadMessage()
	if err == nil {
		t.Fatal("unexpected message received")
	}
	if websocket.IsCloseError(err) {
		t.Fatalf("connection was closed even though its session is still valid: %v", err)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "timeout") && !strings.Contains(strings.ToLower(err.Error()), "deadline") {
		t.Fatalf("expected a read timeout, got a different error: %v", err)
	}
}

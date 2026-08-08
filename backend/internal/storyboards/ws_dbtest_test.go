package storyboards

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/network"
	"victory/backend/internal/sessions"
	"victory/backend/internal/venuecoordination"
)

// dialWS wires a real ServeStoryboardWS behind an httptest.Server and
// dials it with an authenticated session cookie, exercising the exact
// auth-before-upgrade path production traffic uses rather than calling
// package functions directly.
func dialWS(t *testing.T, server *httptest.Server, pool *pgxpool.Pool, userID string) *websocket.Conn {
	t.Helper()
	raw, _, err := sessions.CreateSession(context.Background(), pool, userID, time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	header := http.Header{}
	header.Set("Cookie", sessions.CookieName+"="+raw)
	// The shared network.upgrader's CheckOrigin (Kernel 76 same-origin
	// posture) rejects any handshake with no Origin header at all -- set
	// one matching this test server's own host, exactly what a same-origin
	// browser client would send.
	header.Set("Origin", "http://"+u.Host)
	u.Scheme = "ws"
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	return conn
}

func sendJSON(t *testing.T, conn *websocket.Conn, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func readJSON(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	return out
}

func TestWSSnapshotOnWatchAndReconnect(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "sb_ws_owner")
	board := mustCreateBoard(t, pool, owner, "WS Board")

	hub := network.NewHub()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, venuecoordination.NewRegistry()))
	defer server.Close()

	conn := dialWS(t, server, pool, owner)
	defer conn.Close()

	sendJSON(t, conn, map[string]any{"type": "watch_board", "board_id": board.ID})
	msg := readJSON(t, conn)
	if msg["type"] != "snapshot" {
		t.Fatalf("expected snapshot on watch_board, got %v", msg["type"])
	}

	// Reconnect: a fresh connection watching the same board must also get
	// a fresh snapshot, not a replay.
	conn.Close()
	conn2 := dialWS(t, server, pool, owner)
	defer conn2.Close()
	sendJSON(t, conn2, map[string]any{"type": "watch_board", "board_id": board.ID})
	msg2 := readJSON(t, conn2)
	if msg2["type"] != "snapshot" {
		t.Fatalf("expected snapshot on reconnect watch_board, got %v", msg2["type"])
	}
}

func TestWSWatchRejectedForUnauthorizedUser(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "sb_ws_owner2")
	stranger := insertTestUser(t, pool, "sb_ws_stranger")
	board := mustCreateBoard(t, pool, owner, "WS Private Board")

	hub := network.NewHub()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, venuecoordination.NewRegistry()))
	defer server.Close()

	conn := dialWS(t, server, pool, stranger)
	defer conn.Close()

	sendJSON(t, conn, map[string]any{"type": "watch_board", "board_id": board.ID})
	msg := readJSON(t, conn)
	if msg["type"] != "error" {
		t.Fatalf("expected error for unauthorized watch, got %v", msg)
	}
}

func TestWSRevokedGrantRejectsNextWatch(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_ws_owner4")
	member := insertTestUser(t, pool, "sb_ws_member")
	board := mustCreateBoard(t, pool, owner, "WS Revoke Board")
	var memberHandle string
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, member).Scan(&memberHandle)
	g, err := AddGrant(ctx, pool, owner, board.ID, memberHandle, "crew")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}

	hub := network.NewHub()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, venuecoordination.NewRegistry()))
	defer server.Close()

	conn := dialWS(t, server, pool, member)
	defer conn.Close()
	sendJSON(t, conn, map[string]any{"type": "watch_board", "board_id": board.ID})
	msg := readJSON(t, conn)
	if msg["type"] != "snapshot" {
		t.Fatalf("expected snapshot while grant active, got %v", msg)
	}

	if err := RemoveGrant(ctx, pool, owner, board.ID, g.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	conn2 := dialWS(t, server, pool, member)
	defer conn2.Close()
	sendJSON(t, conn2, map[string]any{"type": "watch_board", "board_id": board.ID})
	msg2 := readJSON(t, conn2)
	if msg2["type"] != "error" {
		t.Fatalf("expected error after grant revoked, got %v", msg2)
	}
}

func TestWSCardEventDeliveredLiveAndHiddenFiltered(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_ws_owner3")
	audience := insertTestUser(t, pool, "sb_ws_audience")
	board := mustCreateBoard(t, pool, owner, "WS Card Board")
	var audienceHandle string
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, audience).Scan(&audienceHandle)
	if _, err := AddGrant(ctx, pool, owner, board.ID, audienceHandle, "audience"); err != nil {
		t.Fatalf("grant audience: %v", err)
	}
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)

	hub := network.NewHub()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, venuecoordination.NewRegistry()))
	defer server.Close()

	audienceConn := dialWS(t, server, pool, audience)
	defer audienceConn.Close()
	sendJSON(t, audienceConn, map[string]any{"type": "watch_board", "board_id": board.ID})
	readJSON(t, audienceConn) // snapshot; watch is set synchronously before this reply is sent

	visibleCard, err := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "Visible", "", "")
	if err != nil {
		t.Fatalf("create visible card: %v", err)
	}
	broadcastCardEvent(ctx, pool, hub, board.ID, EventCardAdded, visibleCard)

	msg := readJSON(t, audienceConn)
	if msg["type"] != EventCardAdded {
		t.Fatalf("expected card_added event for visible card, got %v", msg)
	}

	hiddenCard, err := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "Secret", "", "")
	if err != nil {
		t.Fatalf("create hidden card: %v", err)
	}
	hideVal := true
	hiddenCard, err = UpdateCard(ctx, pool, owner, board.ID, hiddenCard.ID, hiddenCard.Version, CardEdit{
		Title: hiddenCard.Title, HiddenFromAudience: &hideVal,
	})
	if err != nil {
		t.Fatalf("hide card: %v", err)
	}
	broadcastCardEvent(ctx, pool, hub, board.ID, EventCardAdded, hiddenCard)

	// A second, non-hidden event: if the hidden event had (incorrectly)
	// been delivered, it would arrive first and this assertion would catch
	// the leaked title.
	secondVisible, err := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "Visible2", "", "")
	if err != nil {
		t.Fatalf("create second visible card: %v", err)
	}
	broadcastCardEvent(ctx, pool, hub, board.ID, EventCardAdded, secondVisible)

	msg2 := readJSON(t, audienceConn)
	if msg2["type"] != EventCardAdded {
		t.Fatalf("expected card_added event, got %v", msg2)
	}
	data, _ := msg2["data"].(map[string]any)
	if data["title"] == "Secret" {
		t.Fatalf("hidden card content leaked to audience over WS")
	}
	if data["title"] != "Visible2" {
		t.Fatalf("expected the second visible card's event, got %v (hidden event may have leaked ahead of it)", data["title"])
	}
}

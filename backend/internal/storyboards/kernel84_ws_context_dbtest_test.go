package storyboards

// Kernel 84 regression proof for the WebSocket context-lifetime bug
// Kernel 83 flagged: ServeStoryboardWS used to create one
// context.WithTimeout(r.Context(), 5*time.Second) at handshake time and
// reuse it for every subsequent watch_board message on that connection --
// so any watch_board sent more than 5 seconds after connecting silently
// failed with an expired-context error. See ws.go's package comment and
// Construction/Operations/websocket-context-lifecycle.md.

import (
	"net/http/httptest"
	"testing"
	"time"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/network"
	"victory/backend/internal/venuecoordination"
)

func TestWSLateBoardSwitchSurvivesOldSetupTimeout(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "kc84_owner")
	boardA := mustCreateBoard(t, pool, owner, "Kernel84 WS Context Board A")
	boardB := mustCreateBoard(t, pool, owner, "Kernel84 WS Context Board B")

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	conn := dialWS(t, server, pool, owner)
	defer conn.Close()

	sendJSON(t, conn, map[string]any{"type": "watch_board", "board_id": boardA.ID})
	drainForType(t, conn, "snapshot")

	if s, active := reg.Get(StoryboardVenueSessionID(boardA.ID)); !active || s.GroupLeaderUserID != owner {
		t.Fatalf("expected board A session active with owner leader, got %+v active=%v", s, active)
	}

	// Hold the connection open strictly longer than the old 5-second
	// setup-context timeout before doing anything else on it -- this is
	// the exact window the bug only manifested past.
	time.Sleep(6 * time.Second)

	sendJSON(t, conn, map[string]any{"type": "watch_board", "board_id": boardB.ID})
	msg := drainForType(t, conn, "snapshot")
	data, _ := msg["data"].(map[string]any)
	board, _ := data["board"].(map[string]any)
	if board["id"] != boardB.ID {
		t.Fatalf("expected a fresh DB-backed snapshot for board B after the late switch, got %+v", msg)
	}

	// Old-board cleanup: the vacated board's coordination session must
	// have ended (its only watcher just left). New-board registration:
	// the new board's session must have started fresh with owner as
	// leader (owner is the one causing this transition).
	if _, active := reg.Get(StoryboardVenueSessionID(boardA.ID)); active {
		t.Fatalf("expected board A's coordination session to end after the late switch, still active")
	}
	if s, active := reg.Get(StoryboardVenueSessionID(boardB.ID)); !active || s.GroupLeaderUserID != owner {
		t.Fatalf("expected board B's coordination session active with owner leader after late switch, got %+v active=%v", s, active)
	}

	// Disconnect cleanup: no leaked watcher/session state remains once
	// the (now single) connection actually closes.
	conn.Close()
	waitFor(t, 2*time.Second, func() bool {
		_, active := reg.Get(StoryboardVenueSessionID(boardB.ID))
		return !active
	})
	if watchers := hub.BoardWatcherUserIDs(boardB.ID); len(watchers) != 0 {
		t.Fatalf("expected no leaked watchers on board B after disconnect, got %v", watchers)
	}
}

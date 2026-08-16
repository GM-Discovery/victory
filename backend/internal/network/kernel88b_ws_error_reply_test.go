package network

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/actions"
)

// Kernel 88B: per-request error routing depends on a server invariant --
// every handler that accepts a request_id echoes it back on every error path.
// Before this pass those error replies went out via c.Conn.WriteJSON, which
// needs a live websocket, so the invariant could only be verified by reading
// the code. Replies now go through Client.SendJSON and land on the Send
// channel, so they are ordinary values a test can assert on.
//
// Note every client below is constructed with a nil Conn. That is deliberate
// and is itself the structural proof: a handler that still wrote to the
// connection directly would panic here rather than fail an assertion.

func newTestClient(userID string) *Client {
	return &Client{UserID: userID, SessionID: "session-1", Send: make(chan []byte, 8)}
}

// nextReplyOfType drains the client's queue and returns the first message of
// the given type. Handlers may legitimately queue other traffic first.
func nextReplyOfType(t *testing.T, c *Client, wantType string) map[string]any {
	t.Helper()
	for {
		select {
		case raw := <-c.Send:
			var got map[string]any
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("reply was not valid JSON: %v (%s)", err, raw)
			}
			if got["type"] == wantType {
				return got
			}
		default:
			t.Fatalf("no %q reply was queued for the client", wantType)
			return nil
		}
	}
}

func TestPlayerMechanicRollDeniedEchoesRequestID(t *testing.T) {
	old := storePlayerMechanicRollFunc
	defer func() { storePlayerMechanicRollFunc = old }()
	storePlayerMechanicRollFunc = func(ctx context.Context, pool *pgxpool.Pool, req actions.PlayerMechanicRollRequest) (*actions.StoredAction, error) {
		return nil, &actions.ActionDeniedError{Reason: "not_your_turn"}
	}

	hub := NewHub()
	client := newTestClient("user-1")
	hub.Add(client)

	handleCavePayload(hub, nil, client, map[string]any{
		"type":       "roll/dice_own_mechanic",
		"session_id": "session-1",
		"request_id": "req-abc",
		"skill_id":   "insight",
	})

	got := nextReplyOfType(t, client, "error")
	if got["error"] != "not_your_turn" {
		t.Fatalf("error = %v, want not_your_turn", got["error"])
	}
	if got["request_id"] != "req-abc" {
		t.Fatalf("request_id = %v, want req-abc -- without it the client cannot route this refusal back to the control that sent it", got["request_id"])
	}
}

func TestPlayerMechanicRollStoreFailureEchoesRequestID(t *testing.T) {
	old := storePlayerMechanicRollFunc
	defer func() { storePlayerMechanicRollFunc = old }()
	storePlayerMechanicRollFunc = func(ctx context.Context, pool *pgxpool.Pool, req actions.PlayerMechanicRollRequest) (*actions.StoredAction, error) {
		return nil, errors.New("skill_not_found")
	}

	hub := NewHub()
	client := newTestClient("user-1")
	hub.Add(client)

	handleCavePayload(hub, nil, client, map[string]any{
		"type":       "roll/dice_own_mechanic",
		"session_id": "session-1",
		"request_id": "req-def",
		"skill_id":   "nonsense",
	})

	got := nextReplyOfType(t, client, "error")
	if got["error"] != "skill_not_found" {
		t.Fatalf("error = %v, want skill_not_found", got["error"])
	}
	if got["request_id"] != "req-def" {
		t.Fatalf("request_id = %v, want req-def -- the non-denial error branch must echo it too", got["request_id"])
	}
}

func TestDiceRollDeniedEchoesRequestID(t *testing.T) {
	old := storeDiceRollFunc
	defer func() { storeDiceRollFunc = old }()
	storeDiceRollFunc = func(ctx context.Context, pool *pgxpool.Pool, req actions.DiceRollRequest) (*actions.StoredAction, error) {
		return nil, &actions.ActionDeniedError{Reason: "not_authorized"}
	}

	hub := NewHub()
	client := newTestClient("user-1")
	hub.Add(client)

	handleCavePayload(hub, nil, client, map[string]any{
		"type":       "roll/dice",
		"session_id": "session-1",
		"request_id": "req-ghi",
		"expression": "2d6",
	})

	got := nextReplyOfType(t, client, "error")
	if got["request_id"] != "req-ghi" {
		t.Fatalf("request_id = %v, want req-ghi", got["request_id"])
	}
}

// A roll sent without a request_id must not grow an empty one: the client
// treats "" as "nobody is waiting on this" and falls through to the generic
// surfaces, and an empty-string key would be indistinguishable from a real id
// only by value, which is a trap worth closing here.
func TestErrorReplyOmitsRequestIDWhenNoneWasSent(t *testing.T) {
	old := storePlayerMechanicRollFunc
	defer func() { storePlayerMechanicRollFunc = old }()
	storePlayerMechanicRollFunc = func(ctx context.Context, pool *pgxpool.Pool, req actions.PlayerMechanicRollRequest) (*actions.StoredAction, error) {
		return nil, &actions.ActionDeniedError{Reason: "not_your_turn"}
	}

	hub := NewHub()
	client := newTestClient("user-1")
	hub.Add(client)

	handleCavePayload(hub, nil, client, map[string]any{
		"type":       "roll/dice_own_mechanic",
		"session_id": "session-1",
		"skill_id":   "insight",
	})

	got := nextReplyOfType(t, client, "error")
	if _, present := got["request_id"]; present {
		t.Fatalf("request_id must be absent when the client sent none, got %v", got["request_id"])
	}
}

// The server-authored-event rejection is a plain error path with no request
// id at all -- it exercises SendJSON on a handler that previously wrote
// straight to the connection, and must still reach the client.
func TestServerAuthoredEventRejectionIsDeliveredToTheClient(t *testing.T) {
	hub := NewHub()
	client := newTestClient("user-1")
	hub.Add(client)

	handleCavePayload(hub, nil, client, map[string]any{
		"type":       "character/projection_updated",
		"session_id": "session-1",
	})

	got := nextReplyOfType(t, client, "error")
	if got["error"] != "server_authored_event_only" {
		t.Fatalf("error = %v, want server_authored_event_only", got["error"])
	}
}

// ping/pong is the simplest reply in the file and used to be a direct
// connection write. It is the canary for "did the refactor break ordinary
// replies", and it also proves a nil Conn is now survivable.
func TestPingIsAnsweredThroughTheSendChannel(t *testing.T) {
	hub := NewHub()
	client := newTestClient("user-1")
	hub.Add(client)

	handleCavePayload(hub, nil, client, map[string]any{"type": "ping", "session_id": "session-1"})

	got := nextReplyOfType(t, client, "pong")
	if got["ts"] == nil || got["ts"] == "" {
		t.Fatal("pong must carry a timestamp")
	}
}

// SendJSON drops rather than blocks when a client cannot keep up. The read
// loop calls it, so blocking here would wedge the connection's reader.
func TestSendJSONDropsInsteadOfBlockingWhenTheBufferIsFull(t *testing.T) {
	c := &Client{UserID: "user-1", Send: make(chan []byte, 1)}
	if !c.SendJSON(map[string]any{"type": "first"}) {
		t.Fatal("the first message should fit in the buffer")
	}
	if c.SendJSON(map[string]any{"type": "second"}) {
		t.Fatal("a full buffer must drop rather than block the read loop")
	}
}

// CloseSend has two callers (Hub.Remove and the session-revocation path), so
// closing twice must be safe -- a double close panics.
func TestCloseSendIsIdempotent(t *testing.T) {
	hub := NewHub()
	client := newTestClient("user-1")
	hub.Add(client)

	client.CloseSend()
	client.CloseSend()
	hub.Remove(client) // a third close, via the ordinary disconnect path

	if _, open := <-client.Send; open {
		t.Fatal("the channel should be closed and drained")
	}
}

// A message queued immediately before CloseSend must still be delivered:
// that is what lets the revocation path tell a client *why* it was
// disconnected instead of just dropping the socket.
func TestQueuedMessageSurvivesCloseSend(t *testing.T) {
	c := &Client{UserID: "user-1", Send: make(chan []byte, 4)}
	c.SendJSON(map[string]any{"type": "error", "error": "session_revoked"})
	c.CloseSend()

	raw, ok := <-c.Send
	if !ok {
		t.Fatal("the queued notice was lost when the channel closed")
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if got["error"] != "session_revoked" {
		t.Fatalf("error = %v, want session_revoked", got["error"])
	}
}

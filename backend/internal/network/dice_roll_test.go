package network

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/actions"
)

func TestKernel52RollDiceActionBroadcastsCanonicalResult(t *testing.T) {
	oldStore := storeDiceRollFunc
	defer func() { storeDiceRollFunc = oldStore }()
	// Kernel 101 (101-12): the broadcast path this test exercises also
	// calls audienceDiceRollsHiddenFunc with the same nil pool passed to
	// handleCavePayload below -- storeDiceRollFunc's own mock never covered
	// that separate, unrelated real-DB read. Mocked to "not hidden" (dice
	// visible), matching this test's own subject: a public roll broadcasts
	// normally.
	oldHidden := audienceDiceRollsHiddenFunc
	defer func() { audienceDiceRollsHiddenFunc = oldHidden }()
	audienceDiceRollsHiddenFunc = func(ctx context.Context, pool *pgxpool.Pool, sessionID string) bool { return false }

	var req actions.DiceRollRequest
	storeDiceRollFunc = func(ctx context.Context, pool *pgxpool.Pool, incoming actions.DiceRollRequest) (*actions.StoredAction, error) {
		_ = ctx
		_ = pool
		req = incoming
		return &actions.StoredAction{
			ID:        "roll-1",
			SessionID: incoming.SessionID,
			MomentID:  42,
			ActorID:   incoming.ActorID,
			Actor: map[string]any{
				"user_id":      incoming.ActorID,
				"handle":       "producer",
				"display_name": "Producer Mira",
				"role":         "producer",
			},
			Type: "roll/dice",
			Payload: map[string]any{
				"request_id":      incoming.RequestID,
				"expression":      "2d6+3",
				"spec":            map[string]any{"count": 2, "sides": 6, "explode_on_max": false, "modifier": 3},
				"dice":            []map[string]any{{"index": 0, "chain": []int{4}, "subtotal": 4}, {"index": 1, "chain": []int{6}, "subtotal": 6}},
				"explosion_count": 0,
				"modifier":        3,
				"total":           13,
				"roll_version":    1,
				"visibility_mode": "public",
				"label":           "Parent One",
			},
		}, nil
	}

	hub := NewHub()
	client := &Client{UserID: "user-1", SessionID: "session-1", Send: make(chan []byte, 4)}
	hub.Add(client)
	defer hub.Remove(client)

	handleCavePayload(hub, nil, client, map[string]any{
		"type":       "roll/dice",
		"session_id": "session-1",
		"request_id": "req-123",
		"expression": "2d6+3",
		"visibility": "public",
		"label":      "Parent One",
		"actor_id":   "spoofed-user",
		"total":      999,
	})

	if req.ActorID != "user-1" {
		t.Fatalf("expected actor id to come from connection, got %q", req.ActorID)
	}
	if req.RequestID != "req-123" {
		t.Fatalf("expected request id to be forwarded, got %q", req.RequestID)
	}
	if req.Expression != "2d6+3" {
		t.Fatalf("expected expression to be forwarded, got %q", req.Expression)
	}

	msg := recvJSON(t, client.Send)
	if msg["type"] != "action" {
		t.Fatalf("expected action broadcast, got %v", msg["type"])
	}
	if got := msg["data"].(map[string]any)["type"]; got != "roll/dice" {
		t.Fatalf("expected roll/dice action, got %v", got)
	}

	raw, _ := json.Marshal(msg)
	t.Logf("ROLL %s", string(raw))
}

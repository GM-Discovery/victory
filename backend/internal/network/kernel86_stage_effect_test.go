package network

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/actions"
	"victory/backend/internal/stageeffects"
)

// mockStoredDiceRoll wires storeDiceRollFunc to return a canned
// StoredAction carrying the given audienceMode/cohortId in its Visibility
// map, matching exactly what actions.StoreDiceRoll now writes there
// (Kernel 86). Restores the previous func on test cleanup.
func mockStoredDiceRoll(t *testing.T, audienceMode, cohortID string) {
	t.Helper()
	old := storeDiceRollFunc
	t.Cleanup(func() { storeDiceRollFunc = old })

	storeDiceRollFunc = func(ctx context.Context, pool *pgxpool.Pool, incoming actions.DiceRollRequest) (*actions.StoredAction, error) {
		return &actions.StoredAction{
			ID:        "action-" + incoming.RequestID,
			SessionID: incoming.SessionID,
			ActorID:   incoming.ActorID,
			Type:      "roll/dice",
			Actor:     map[string]any{"user_id": incoming.ActorID},
			Payload: map[string]any{
				"expression":      "1d20",
				"total":           17,
				"modifier":        0,
				"explosion_count": 0,
				"label":           incoming.Label,
				"visibility_mode": audienceMode,
			},
			Visibility: map[string]any{
				"audienceMode": audienceMode,
				"cohortId":     cohortID,
				"toRoles":      []string{},
				"privateTo":    []string{},
			},
		}, nil
	}
}

// stageEffectFixture builds a bare stageeffects.Effect for direct registry/
// authority unit tests that don't need a full roll/dice round trip.
func stageEffectFixture(actorID string) stageeffects.Effect {
	return stageeffects.Effect{
		Type:       "dice_roll",
		Audience:   "private",
		ActorID:    actorID,
		DurationMs: 3500,
	}
}

func recvTypedJSON(t *testing.T, ch chan []byte, wantType string) map[string]any {
	t.Helper()
	select {
	case raw := <-ch:
		msg := map[string]any{}
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("unmarshal message: %v (raw=%s)", err, raw)
		}
		if msg["type"] != wantType {
			t.Fatalf("message type = %v, want %q (raw=%s)", msg["type"], wantType, raw)
		}
		return msg
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for a %q message", wantType)
	}
	return nil
}

// TestRollDiceProjectsStageEffectAfterAction proves a roll/dice request now
// produces two live messages in order -- the existing canonical "action"
// (unchanged shape) followed by a new "stage_effect" (Kernel 86 §6) derived
// from it -- rather than only the Action broadcast prior kernels relied on.
func TestRollDiceProjectsStageEffectAfterAction(t *testing.T) {
	mockStoredDiceRoll(t, "show", "")

	hub := NewHub()
	client := &Client{UserID: "user-1", SessionID: "session-1", Send: make(chan []byte, 4)}
	hub.Add(client)
	defer hub.Remove(client)

	handleCavePayload(hub, nil, client, map[string]any{
		"type":       "roll/dice",
		"session_id": "session-1",
		"request_id": "req-stage-effect",
		"expression": "1d20",
		"label":      "Perception",
	})

	action := recvTypedJSON(t, client.Send, "action")
	if got := action["data"].(map[string]any)["type"]; got != "roll/dice" {
		t.Fatalf("expected the action message to carry a roll/dice Action, got %v", got)
	}

	effectMsg := recvTypedJSON(t, client.Send, "stage_effect")
	effect := effectMsg["data"].(map[string]any)
	if effect["type"] != "dice_roll" {
		t.Fatalf("effect type = %v, want dice_roll", effect["type"])
	}
	if effect["source_action_id"] != "action-req-stage-effect" {
		t.Fatalf("source_action_id = %v, want the stored Action's ID", effect["source_action_id"])
	}
	if effect["audience"] != "show" {
		t.Fatalf("audience = %v, want show", effect["audience"])
	}
	if effect["pinned"] != false {
		t.Fatalf("a freshly created effect must not start pinned, got %v", effect["pinned"])
	}
	if id, _ := effect["effect_id"].(string); id == "" {
		t.Fatal("expected a non-empty effect_id")
	}
}

// TestStageEffectSelfPinAndDismiss proves the roller can pin their own
// projection and later dismiss it, entirely without a Director+ DB role
// check (the actor==roller shortcut in stageEffectAuthorized), and that the
// resulting stage_effect_pinned/stage_effect_dismissed messages are
// delivered back to the same targeted audience.
func TestStageEffectSelfPinAndDismiss(t *testing.T) {
	mockStoredDiceRoll(t, "show", "")

	hub := NewHub()
	client := &Client{UserID: "user-1", SessionID: "session-1", Send: make(chan []byte, 4)}
	hub.Add(client)
	defer hub.Remove(client)

	handleCavePayload(hub, nil, client, map[string]any{
		"type":       "roll/dice",
		"session_id": "session-1",
		"request_id": "req-pin-flow",
		"expression": "1d20",
	})
	_ = recvTypedJSON(t, client.Send, "action")
	effectMsg := recvTypedJSON(t, client.Send, "stage_effect")
	effectID, _ := effectMsg["data"].(map[string]any)["effect_id"].(string)
	if effectID == "" {
		t.Fatal("expected an effect_id to pin")
	}

	handleCavePayload(hub, nil, client, map[string]any{
		"type":      "stage_effect/pin",
		"effect_id": effectID,
	})
	pinnedMsg := recvTypedJSON(t, client.Send, "stage_effect_pinned")
	if pinned := pinnedMsg["data"].(map[string]any)["pinned"]; pinned != true {
		t.Fatalf("expected pinned=true after pin, got %v", pinned)
	}

	handleCavePayload(hub, nil, client, map[string]any{
		"type":      "stage_effect/dismiss",
		"effect_id": effectID,
	})
	dismissedMsg := recvTypedJSON(t, client.Send, "stage_effect_dismissed")
	if dismissedMsg["effect_id"] != effectID {
		t.Fatalf("dismissed effect_id = %v, want %q", dismissedMsg["effect_id"], effectID)
	}
}

// TestStageEffectPrivateRollNeverReachesAnotherSocket proves the live
// targeted-delivery half of kernel §10's guarantee: a Private roll's
// action+stage_effect messages must not reach any socket besides the
// roller's own, even another authenticated user on the very same session.
func TestStageEffectPrivateRollNeverReachesAnotherSocket(t *testing.T) {
	mockStoredDiceRoll(t, "private", "")

	hub := NewHub()
	roller := &Client{UserID: "user-1", SessionID: "session-1", Send: make(chan []byte, 4)}
	other := &Client{UserID: "user-2", SessionID: "session-1", Send: make(chan []byte, 4)}
	hub.Add(roller)
	hub.Add(other)
	defer hub.Remove(roller)
	defer hub.Remove(other)

	handleCavePayload(hub, nil, roller, map[string]any{
		"type":       "roll/dice",
		"session_id": "session-1",
		"request_id": "req-private-flow",
		"expression": "1d20",
	})
	_ = recvTypedJSON(t, roller.Send, "action")
	_ = recvTypedJSON(t, roller.Send, "stage_effect")

	select {
	case got := <-other.Send:
		t.Fatalf("an unrelated user received a private roll's live payload: %s", got)
	default:
	}
}

// TestStageEffectAuthorizedPrivateHasNoDirectorException is a direct unit
// test of stageEffectAuthorized (not routed through handleCavePayload/
// Conn.WriteJSON, which needs a real websocket connection for its error
// path): it proves a Private effect rejects every non-actor -- including
// Director+ -- purely from e.Audience, without ever reaching the
// rollaudience.IsDirectorPlus DB lookup (pool is nil here and must not be
// touched).
func TestStageEffectAuthorizedPrivateHasNoDirectorException(t *testing.T) {
	effect := stageEffectRegistry.Create("auth-test-session", stageEffectFixture("private-actor"))
	// Re-fetch through the registry rather than trusting the struct
	// returned by Create, matching how ws.go itself always looks effects
	// up before checking authority.
	effect, ok := stageEffectRegistry.Get("auth-test-session", effect.ID)
	if !ok {
		t.Fatal("expected the fixture effect to exist")
	}

	ctx := context.Background()
	if ok, err := stageEffectAuthorized(ctx, nil, effect, "private-actor"); err != nil || !ok {
		t.Fatalf("the actor must always be authorized on their own effect, ok=%v err=%v", ok, err)
	}
	if ok, err := stageEffectAuthorized(ctx, nil, effect, "someone-else"); err != nil || ok {
		t.Fatalf("a non-actor must never be authorized on a Private effect, ok=%v err=%v", ok, err)
	}
}

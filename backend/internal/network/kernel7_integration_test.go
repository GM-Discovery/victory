package network

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/actions"
)

func TestKernel8IdentitySurfaceAndPersonaNull(t *testing.T) {
	sessionID := "session-kernel7"
	producer := PresenceUser{
		UserID:      "user-producer",
		Handle:      "producer",
		DisplayName: "Producer Mira",
		Role:        "producer",
		Persona:     nil,
	}
	audience := PresenceUser{
		UserID:      "user-audience",
		Handle:      "audience",
		DisplayName: "Audience Grant",
		Role:        "audience",
		Persona:     nil,
	}

	registry := NewPresenceRegistry()
	snapshot, joined := registry.Connect(sessionID, producer)
	if !joined || len(snapshot) != 1 {
		t.Fatalf("expected first connect to join once, got joined=%v snapshot=%d", joined, len(snapshot))
	}
	t.Logf("PRESENCE first connect snapshot=%d joined=%v", len(snapshot), joined)

	snapshot, joined = registry.Connect(sessionID, producer)
	if joined || len(snapshot) != 1 {
		t.Fatalf("expected duplicate tab to dedupe, got joined=%v snapshot=%d", joined, len(snapshot))
	}
	t.Logf("PRESENCE duplicate connect snapshot=%d joined=%v", len(snapshot), joined)

	snapshot, joined = registry.Connect(sessionID, audience)
	if !joined || len(snapshot) != 2 {
		t.Fatalf("expected audience connect to add second visible user, got joined=%v snapshot=%d", joined, len(snapshot))
	}
	t.Logf("PRESENCE audience connect snapshot=%d joined=%v", len(snapshot), joined)

	snapshot, left := registry.Disconnect(sessionID, producer.UserID)
	if left || len(snapshot) != 2 {
		t.Fatalf("expected first disconnect to keep user visible, got left=%v snapshot=%d", left, len(snapshot))
	}
	t.Logf("PRESENCE first disconnect snapshot=%d left=%v", len(snapshot), left)

	snapshot, left = registry.Disconnect(sessionID, producer.UserID)
	if !left || len(snapshot) != 1 {
		t.Fatalf("expected final disconnect to remove user, got left=%v snapshot=%d", left, len(snapshot))
	}
	t.Logf("PRESENCE final disconnect snapshot=%d left=%v", len(snapshot), left)

	hub := NewHub()
	receiver := &Client{Send: make(chan []byte, 8)}
	hub.Add(receiver)
	defer hub.Remove(receiver)

	hub.Broadcast(mustJSON(map[string]any{
		"type":  "presence/snapshot",
		"users": snapshot,
	}))
	msg := recvJSON(t, receiver.Send)
	if msg["type"] != "presence/snapshot" {
		t.Fatalf("expected presence/snapshot, got %v", msg["type"])
	}
	assertPresencePersonaNull(t, msg)
	t.Logf("PRESENCE SNAPSHOT %s", testJSON(msg))

	broadcastPresenceEvent(hub, sessionID, "presence/join", producer)
	msg = recvJSON(t, receiver.Send)
	if msg["type"] != "presence/join" {
		t.Fatalf("expected presence/join, got %v", msg["type"])
	}
	assertPresencePersonaNull(t, msg)
	t.Logf("PRESENCE JOIN %s", testJSON(msg))

	broadcastPresenceEvent(hub, sessionID, "presence/leave", producer)
	msg = recvJSON(t, receiver.Send)
	if msg["type"] != "presence/leave" {
		t.Fatalf("expected presence/leave, got %v", msg["type"])
	}
	assertPresencePersonaNull(t, msg)
	t.Logf("PRESENCE LEAVE %s", testJSON(msg))

	oldSpeak := storeSpeakFunc
	oldReact := storeReactionFunc
	oldReveal := storeRevealFunc
	oldChat := storeChatMessageFunc
	defer func() {
		storeSpeakFunc = oldSpeak
		storeReactionFunc = oldReact
		storeRevealFunc = oldReveal
		storeChatMessageFunc = oldChat
	}()

	storeSpeakFunc = func(ctx context.Context, pool *pgxpool.Pool, sessionID, actorID, text string) (*actions.StoredAction, error) {
		return &actions.StoredAction{
			ID:        "action-speak",
			SessionID: sessionID,
			MomentID:  1,
			ActorID:   actorID,
			Actor: map[string]any{
				"user_id":      actorID,
				"handle":       "producer",
				"display_name": "Producer Mira",
				"role":         "director",
				"persona":      nil,
			},
			Persona: nil,
			Type:    "perform/speak",
			Payload: map[string]any{"text": text},
		}, nil
	}
	storeReactionFunc = func(ctx context.Context, pool *pgxpool.Pool, req actions.ReactRequest) (*actions.StoredAction, error) {
		return &actions.StoredAction{
			ID:        "action-react",
			SessionID: req.SessionID,
			MomentID:  2,
			ActorID:   req.ActorID,
			Actor: map[string]any{
				"user_id":      req.ActorID,
				"handle":       "audience",
				"display_name": "Audience Grant",
				"role":         "audience",
				"persona":      nil,
			},
			Persona: nil,
			Type:    "react/emote",
			Payload: map[string]any{"kind": req.Kind},
		}, nil
	}
	storeRevealFunc = func(ctx context.Context, pool *pgxpool.Pool, req actions.RevealRequest) (*actions.StoredAction, error) {
		return &actions.StoredAction{}, nil
	}
	storeChatMessageFunc = func(ctx context.Context, pool *pgxpool.Pool, req actions.ChatMessageRequest) (*actions.StoredAction, error) {
		return &actions.StoredAction{
			ID:        "action-chat",
			SessionID: req.SessionID,
			MomentID:  3,
			ActorID:   req.ActorID,
			Actor: map[string]any{
				"user_id":      req.ActorID,
				"handle":       "audience",
				"display_name": "Audience Grant",
				"role":         "audience",
				"persona":      nil,
			},
			Persona: nil,
			Type:    "chat/message",
			Payload: map[string]any{"text": req.Text},
		}, nil
	}

	actionHub := NewHub()
	actionReceiver := &Client{Send: make(chan []byte, 8)}
	actionHub.Add(actionReceiver)
	defer actionHub.Remove(actionReceiver)

	speaker := &Client{UserID: producer.UserID, SessionID: sessionID}
	handleCavePayload(actionHub, nil, speaker, map[string]any{
		"type":       "perform/speak",
		"session_id": sessionID,
		"text":       "The fire answers.",
		"actor_id":   "spoofed-user-id",
		"role":       "audience",
	})
	speech := recvJSON(t, actionReceiver.Send)
	if speech["type"] != "action" {
		t.Fatalf("expected action broadcast, got %v", speech["type"])
	}
	assertActionActor(t, speech["data"], producer.UserID, "director")
	assertPersonaNull(t, speech["data"])
	if got := actionText(t, speech["data"]); got != "The fire answers." {
		t.Fatalf("expected speech text, got %q", got)
	}
	t.Logf("SPEECH %s", testJSON(speech))

	reactor := &Client{UserID: audience.UserID, SessionID: sessionID}
	handleCavePayload(actionHub, nil, reactor, map[string]any{
		"type":       "react/emote",
		"session_id": sessionID,
		"kind":       "applause",
		"actor_id":   "spoofed-user-id",
		"role":       "producer",
	})
	reaction := recvJSON(t, actionReceiver.Send)
	if reaction["type"] != "action" {
		t.Fatalf("expected action broadcast, got %v", reaction["type"])
	}
	assertActionActor(t, reaction["data"], audience.UserID, "audience")
	assertPersonaNull(t, reaction["data"])
	if got := actionKind(t, reaction["data"]); got != "applause" {
		t.Fatalf("expected reaction kind, got %q", got)
	}
	t.Logf("REACTION %s", testJSON(reaction))

	chatty := &Client{UserID: audience.UserID, SessionID: sessionID}
	handleCavePayload(actionHub, nil, chatty, map[string]any{
		"type":       "chat/message",
		"session_id": sessionID,
		"text":       "This is just venue chat.",
		"actor_id":   "spoofed-user-id",
		"role":       "producer",
	})
	chat := recvJSON(t, actionReceiver.Send)
	if chat["type"] != "action" {
		t.Fatalf("expected action broadcast, got %v", chat["type"])
	}
	assertActionActor(t, chat["data"], audience.UserID, "audience")
	assertPersonaNull(t, chat["data"])
	if got := actionText(t, chat["data"]); got != "This is just venue chat." {
		t.Fatalf("expected chat text, got %q", got)
	}
	t.Logf("CHAT %s", testJSON(chat))

	t.Log("KERNEL21_PROOF PASS")
}

func TestKernel12IndexCardActionAuthorityAndBroadcast(t *testing.T) {
	oldCreate := storeIndexCardCreateFunc
	oldUpdate := storeIndexCardUpdateFunc
	defer func() {
		storeIndexCardCreateFunc = oldCreate
		storeIndexCardUpdateFunc = oldUpdate
	}()

	var createReq actions.IndexCardRequest
	var updateReq actions.IndexCardRequest

	storeIndexCardCreateFunc = func(ctx context.Context, pool *pgxpool.Pool, req actions.IndexCardRequest) (*actions.StoredAction, error) {
		createReq = req
		return &actions.StoredAction{
			ID:        "action-card-create",
			SessionID: req.SessionID,
			MomentID:  11,
			ActorID:   req.ActorID,
			Actor: map[string]any{
				"user_id":      req.ActorID,
				"handle":       "director",
				"display_name": "Director One",
				"role":         "director",
				"persona":      nil,
			},
			Persona: nil,
			Type:    "create/index_card",
			Payload: map[string]any{"front_text": req.FrontText, "back_text": req.BackText, "color": req.Color},
			Target:  map[string]any{"kind": "index_card", "element_id": "card-1", "element_slug": "index-card-director"},
		}, nil
	}
	storeIndexCardUpdateFunc = func(ctx context.Context, pool *pgxpool.Pool, req actions.IndexCardRequest) (*actions.StoredAction, error) {
		updateReq = req
		return &actions.StoredAction{
			ID:        "action-card-update",
			SessionID: req.SessionID,
			MomentID:  12,
			ActorID:   req.ActorID,
			Actor: map[string]any{
				"user_id":      req.ActorID,
				"handle":       "director",
				"display_name": "Director One",
				"role":         "director",
				"persona":      nil,
			},
			Persona: nil,
			Type:    "update/index_card",
			Payload: map[string]any{"front_text": req.FrontText, "back_text": req.BackText, "color": req.Color},
			Target:  map[string]any{"kind": "index_card", "element_id": req.ElementID, "element_slug": req.ElementSlug},
		}, nil
	}

	hub := NewHub()
	receiver := &Client{Send: make(chan []byte, 8)}
	hub.Add(receiver)
	defer hub.Remove(receiver)

	manager := &Client{UserID: "user-director", SessionID: "session-kernel12"}
	handleCavePayload(hub, nil, manager, map[string]any{
		"type":       "create/index_card",
		"session_id": "session-kernel12",
		"actor_id":   "spoofed-user-id",
		"front_text": "Call sheet",
		"back_text":  "Remember the monologue.",
		"color":      "#d9c7a6",
	})
	createBroadcast := recvJSON(t, receiver.Send)
	if createBroadcast["type"] != "action" {
		t.Fatalf("expected action broadcast, got %v", createBroadcast["type"])
	}
	assertPersonaNull(t, createBroadcast["data"])
	if createReq.ActorID != manager.UserID {
		t.Fatalf("expected create actor id %q, got %q", manager.UserID, createReq.ActorID)
	}
	if got := createReq.FrontText; got != "Call sheet" {
		t.Fatalf("expected create front text, got %q", got)
	}
	t.Logf("INDEX CARD CREATE %s", testJSON(createBroadcast))

	handleCavePayload(hub, nil, manager, map[string]any{
		"type":         "update/index_card",
		"session_id":   "session-kernel12",
		"actor_id":     "spoofed-user-id",
		"element_id":   "card-1",
		"element_slug": "index-card-director",
		"front_text":   "Updated call sheet",
		"back_text":    "Updated back",
		"color":        "#334455",
	})
	updateBroadcast := recvJSON(t, receiver.Send)
	if updateBroadcast["type"] != "action" {
		t.Fatalf("expected action broadcast, got %v", updateBroadcast["type"])
	}
	assertPersonaNull(t, updateBroadcast["data"])
	if updateReq.ActorID != manager.UserID {
		t.Fatalf("expected update actor id %q, got %q", manager.UserID, updateReq.ActorID)
	}
	if got := updateReq.ElementID; got != "card-1" {
		t.Fatalf("expected update element id card-1, got %q", got)
	}
	t.Logf("INDEX CARD UPDATE %s", testJSON(updateBroadcast))

	t.Log("KERNEL12_PROOF PASS")
}

func TestKernel13WorkshopPlacementAuthorityAndBroadcast(t *testing.T) {
	oldPlace := storePlaceElementFunc
	defer func() {
		storePlaceElementFunc = oldPlace
	}()

	var placeReq actions.PlaceElementRequest
	storePlaceElementFunc = func(ctx context.Context, pool *pgxpool.Pool, req actions.PlaceElementRequest) (*actions.StoredAction, error) {
		placeReq = req
		return &actions.StoredAction{
			ID:        "action-card-place",
			SessionID: req.SessionID,
			MomentID:  13,
			ActorID:   req.ActorID,
			Actor: map[string]any{
				"user_id":      req.ActorID,
				"handle":       "director",
				"display_name": "Director One",
				"role":         "director",
				"persona":      nil,
			},
			Persona: nil,
			Type:    "act/place_element",
			Target: map[string]any{
				"kind":         "index_card",
				"element_id":   req.ElementID,
				"element_slug": req.ElementSlug,
				"venue_slug":   req.VenueSlug,
				"layer":        req.Layer,
				"x":            req.X,
				"y":            req.Y,
			},
			Payload: map[string]any{
				"venue_slug": req.VenueSlug,
				"layer":      req.Layer,
				"x":          req.X,
				"y":          req.Y,
				"order":      req.Order,
			},
		}, nil
	}

	hub := NewHub()
	receiver := &Client{Send: make(chan []byte, 8)}
	hub.Add(receiver)
	defer hub.Remove(receiver)

	manager := &Client{UserID: "user-director", SessionID: "session-kernel13"}
	handleCavePayload(hub, nil, manager, map[string]any{
		"type":         "act/place_element",
		"session_id":   "session-kernel13",
		"actor_id":     "spoofed-user-id",
		"element_id":   "card-1",
		"element_slug": "index-card-director",
		"venue_slug":   "the-cave",
		"layer":        "tray",
		"x":            0,
		"y":            0,
	})
	broadcast := recvJSON(t, receiver.Send)
	if broadcast["type"] != "action" {
		t.Fatalf("expected action broadcast, got %v", broadcast["type"])
	}
	assertPersonaNull(t, broadcast["data"])
	assertActionActor(t, broadcast["data"], manager.UserID, "director")
	if placeReq.ActorID != manager.UserID {
		t.Fatalf("expected place actor id %q, got %q", manager.UserID, placeReq.ActorID)
	}
	if placeReq.VenueSlug != "the-cave" || placeReq.Layer != "tray" {
		t.Fatalf("expected placement venue/layer to be the-cave tray, got %+v", placeReq)
	}
	t.Logf("WORKSHOP PLACE TRAY %s", testJSON(broadcast))

	handleCavePayload(hub, nil, manager, map[string]any{
		"type":         "act/place_element",
		"session_id":   "session-kernel13",
		"actor_id":     "spoofed-user-id",
		"element_id":   "card-1",
		"element_slug": "index-card-director",
		"venue_slug":   "the-cave",
		"layer":        "stage",
		"x":            0,
		"y":            0,
	})
	broadcast = recvJSON(t, receiver.Send)
	if broadcast["type"] != "action" {
		t.Fatalf("expected action broadcast, got %v", broadcast["type"])
	}
	assertPersonaNull(t, broadcast["data"])
	assertActionActor(t, broadcast["data"], manager.UserID, "director")
	if got := actionVenueSlug(t, broadcast["data"]); got != "the-cave" {
		t.Fatalf("expected venue slug the-cave, got %q", got)
	}
	if got := actionLayer(t, broadcast["data"]); got != "stage" {
		t.Fatalf("expected stage placement, got %q", got)
	}
	t.Logf("WORKSHOP PLACE STAGE %s", testJSON(broadcast))

	t.Log("KERNEL13_PROOF PASS")
}

func recvJSON(t *testing.T, ch <-chan []byte) map[string]any {
	t.Helper()

	select {
	case raw := <-ch:
		var out map[string]any
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		return out
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for broadcast")
		return nil
	}
}

func assertActionActor(t *testing.T, action any, wantUserID, wantRole string) {
	t.Helper()

	m, _ := action.(map[string]any)
	actor, _ := m["actor"].(map[string]any)
	if got, _ := actor["user_id"].(string); got != wantUserID {
		t.Fatalf("expected actor user_id %s, got %q", wantUserID, got)
	}
	if got, _ := actor["role"].(string); got != wantRole {
		t.Fatalf("expected actor role %s, got %q", wantRole, got)
	}
}

func assertPersonaNull(t *testing.T, value any) {
	t.Helper()

	m, _ := value.(map[string]any)
	if persona, ok := m["persona"]; ok && persona != nil {
		t.Fatalf("expected persona to be null, got %v", persona)
	}
}

func assertPresencePersonaNull(t *testing.T, value any) {
	t.Helper()

	m, _ := value.(map[string]any)
	user, _ := m["user"].(map[string]any)
	if persona, ok := user["persona"]; ok && persona != nil {
		t.Fatalf("expected presence persona to be null, got %v", persona)
	}
	if users, ok := m["users"].([]any); ok {
		for _, entry := range users {
			row, _ := entry.(map[string]any)
			if persona, ok := row["persona"]; ok && persona != nil {
				t.Fatalf("expected presence persona to be null, got %v", persona)
			}
		}
	}
}

func actionText(t *testing.T, action any) string {
	t.Helper()

	m, _ := action.(map[string]any)
	payload, _ := m["payload"].(map[string]any)
	if got, _ := payload["text"].(string); got != "" {
		return got
	}
	return ""
}

func actionKind(t *testing.T, action any) string {
	t.Helper()

	m, _ := action.(map[string]any)
	payload, _ := m["payload"].(map[string]any)
	if got, _ := payload["kind"].(string); got != "" {
		return got
	}
	return ""
}

func actionVenueSlug(t *testing.T, action any) string {
	t.Helper()

	m, _ := action.(map[string]any)
	target, _ := m["target"].(map[string]any)
	if got, _ := target["venue_slug"].(string); got != "" {
		return got
	}
	payload, _ := m["payload"].(map[string]any)
	if got, _ := payload["venue_slug"].(string); got != "" {
		return got
	}
	return ""
}

func actionLayer(t *testing.T, action any) string {
	t.Helper()

	m, _ := action.(map[string]any)
	target, _ := m["target"].(map[string]any)
	if got, _ := target["layer"].(string); got != "" {
		return got
	}
	payload, _ := m["payload"].(map[string]any)
	if got, _ := payload["layer"].(string); got != "" {
		return got
	}
	return ""
}

func testJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

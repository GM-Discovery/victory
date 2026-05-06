package network

import "testing"

func TestPresenceRegistryConnectDisconnect(t *testing.T) {
	registry := NewPresenceRegistry()

	user := PresenceUser{
		UserID:      "user-1",
		Handle:      "web_user",
		DisplayName: "Web User",
		Role:        "cast",
	}

	snapshot, joined := registry.Connect("session-1", user)
	if !joined {
		t.Fatalf("expected first connect to join")
	}
	if len(snapshot) != 1 {
		t.Fatalf("expected 1 presence user, got %d", len(snapshot))
	}

	snapshot, joined = registry.Connect("session-1", user)
	if joined {
		t.Fatalf("expected duplicate connect to reuse presence entry")
	}
	if len(snapshot) != 1 {
		t.Fatalf("expected 1 presence user after duplicate connect, got %d", len(snapshot))
	}

	snapshot, left := registry.Disconnect("session-1", user.UserID)
	if left {
		t.Fatalf("expected first disconnect to keep presence alive")
	}
	if len(snapshot) != 1 {
		t.Fatalf("expected 1 presence user after first disconnect, got %d", len(snapshot))
	}

	snapshot, left = registry.Disconnect("session-1", user.UserID)
	if !left {
		t.Fatalf("expected second disconnect to remove presence")
	}
	if len(snapshot) != 0 {
		t.Fatalf("expected empty presence snapshot after final disconnect, got %d", len(snapshot))
	}
}

func TestPresenceRegistryUpdatePersona(t *testing.T) {
	registry := NewPresenceRegistry()

	user := PresenceUser{
		UserID:      "user-1",
		Handle:      "web_user",
		DisplayName: "Web User",
		Role:        "cast",
	}
	registry.Connect("session-1", user)

	persona := map[string]any{
		"character_card_id": "card-1",
		"name":              "Captain Lantern",
	}
	updated, ok := registry.Update("session-1", PresenceUser{
		UserID:  "user-1",
		Persona: persona,
	})
	if !ok {
		t.Fatalf("expected update to find presence user")
	}
	if updated.Persona == nil {
		t.Fatalf("expected persona to be projected")
	}

	snapshot := registry.Snapshot("session-1")
	if len(snapshot) != 1 {
		t.Fatalf("expected one presence user, got %d", len(snapshot))
	}
	if snapshot[0].Persona == nil {
		t.Fatalf("expected snapshot persona to be projected")
	}

	updated, ok = registry.Update("session-1", PresenceUser{UserID: "user-1"})
	if !ok {
		t.Fatalf("expected unequip update to find presence user")
	}
	if updated.Persona != nil {
		t.Fatalf("expected persona to clear, got %v", updated.Persona)
	}
}

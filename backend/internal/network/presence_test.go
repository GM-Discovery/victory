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

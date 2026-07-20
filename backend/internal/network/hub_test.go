package network

import "testing"

// TestBroadcastToSessionUserTargetsOnlySessionAndUser proves the Kernel 73
// privacy requirement (spec S6.2/S12): Program Panel invalidations reach
// only the exact (session, user) pair that triggered them -- never other
// Players in the same session, never the same user in a different session,
// never Audience. No websocket connection needed; Client.Send is a plain
// channel this test drains directly.
func TestBroadcastToSessionUserTargetsOnlySessionAndUser(t *testing.T) {
	hub := NewHub()

	targetClient := &Client{SessionID: "session-a", UserID: "user-1", Send: make(chan []byte, 1)}
	otherUserSameSession := &Client{SessionID: "session-a", UserID: "user-2", Send: make(chan []byte, 1)}
	sameUserOtherSession := &Client{SessionID: "session-b", UserID: "user-1", Send: make(chan []byte, 1)}

	hub.Add(targetClient)
	hub.Add(otherUserSameSession)
	hub.Add(sameUserOtherSession)
	defer func() {
		hub.Remove(targetClient)
		hub.Remove(otherUserSameSession)
		hub.Remove(sameUserOtherSession)
	}()

	msg := []byte(`{"type":"merchant/inventory_updated"}`)
	hub.BroadcastToSessionUser("session-a", "user-1", msg)

	select {
	case got := <-targetClient.Send:
		if string(got) != string(msg) {
			t.Fatalf("target client got %q, want %q", got, msg)
		}
	default:
		t.Fatal("target client (matching session+user) received nothing")
	}

	select {
	case got := <-otherUserSameSession.Send:
		t.Fatalf("a different user in the same session should not receive the push, got %q", got)
	default:
	}

	select {
	case got := <-sameUserOtherSession.Send:
		t.Fatalf("the same user in a different session should not receive the push, got %q", got)
	default:
	}
}

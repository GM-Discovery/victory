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

// TestSendToUsersTargetsExactSetWithinSession proves Kernel 86's targeted
// delivery primitive: a Cohort/Director/Private roll projection must reach
// every recipient in the caller-resolved set, on this session only, and
// nothing else -- not another user in the same session who isn't in the
// set, not the same user ID in a different session, and (multi-tab safety)
// both of a recipient's two open sockets.
func TestSendToUsersTargetsExactSetWithinSession(t *testing.T) {
	hub := NewHub()

	recipientTabOne := &Client{SessionID: "session-a", UserID: "user-1", Send: make(chan []byte, 1)}
	recipientTabTwo := &Client{SessionID: "session-a", UserID: "user-1", Send: make(chan []byte, 1)}
	otherRecipient := &Client{SessionID: "session-a", UserID: "user-2", Send: make(chan []byte, 1)}
	notInSet := &Client{SessionID: "session-a", UserID: "user-3", Send: make(chan []byte, 1)}
	sameUserOtherSession := &Client{SessionID: "session-b", UserID: "user-1", Send: make(chan []byte, 1)}

	for _, c := range []*Client{recipientTabOne, recipientTabTwo, otherRecipient, notInSet, sameUserOtherSession} {
		hub.Add(c)
	}
	defer func() {
		for _, c := range []*Client{recipientTabOne, recipientTabTwo, otherRecipient, notInSet, sameUserOtherSession} {
			hub.Remove(c)
		}
	}()

	msg := []byte(`{"type":"stage_effect"}`)
	hub.SendToUsers("session-a", []string{"user-1", "user-2"}, msg)

	for _, c := range []*Client{recipientTabOne, recipientTabTwo, otherRecipient} {
		select {
		case got := <-c.Send:
			if string(got) != string(msg) {
				t.Fatalf("recipient got %q, want %q", got, msg)
			}
		default:
			t.Fatal("an authorized recipient/tab received nothing")
		}
	}

	for _, c := range []*Client{notInSet, sameUserOtherSession} {
		select {
		case got := <-c.Send:
			t.Fatalf("an unauthorized client received the payload: %q", got)
		default:
		}
	}
}

// TestSendToUsersEmptySetIsANoOp guards against a caller passing an empty
// recipient set (e.g. a Private roll whose actor ID somehow resolved
// empty) ever silently falling through to "send to everyone."
func TestSendToUsersEmptySetIsANoOp(t *testing.T) {
	hub := NewHub()
	c := &Client{SessionID: "session-a", UserID: "user-1", Send: make(chan []byte, 1)}
	hub.Add(c)
	defer hub.Remove(c)

	hub.SendToUsers("session-a", nil, []byte(`{"type":"stage_effect"}`))

	select {
	case got := <-c.Send:
		t.Fatalf("expected no delivery for an empty recipient set, got %q", got)
	default:
	}
}

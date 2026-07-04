package actions

import (
	"context"
	"testing"
)

func TestCanActOOCMessageAllowlisted(t *testing.T) {
	tests := []struct {
		name           string
		role           string
		chatEnabled    bool
		talkingEnabled bool
		want           bool
		wantReason     string
	}{
		{name: "chat disabled denied", role: "cast", chatEnabled: false, talkingEnabled: false, want: false, wantReason: "policy_denied"},
		{name: "cast allowed when chat enabled", role: "cast", chatEnabled: true, talkingEnabled: false, want: true, wantReason: "allowed"},
		{name: "audience denied without talking", role: "audience", chatEnabled: true, talkingEnabled: false, want: false, wantReason: "insufficient_role"},
		{name: "audience allowed when talking enabled", role: "audience", chatEnabled: true, talkingEnabled: true, want: true, wantReason: "allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := fakeQuerier{role: tt.role, chatEnabled: tt.chatEnabled, talkingEnabled: tt.talkingEnabled}
			decision, err := CanAct(context.Background(), q, "user-1", "chat/ooc", "session-1", ActionTarget{Kind: "session"})
			if err != nil {
				t.Fatalf("CanAct chat/ooc returned error: %v", err)
			}
			if decision.Allowed != tt.want {
				t.Fatalf("CanAct chat/ooc allowed=%v, want %v", decision.Allowed, tt.want)
			}
			if decision.Reason != tt.wantReason {
				t.Fatalf("CanAct chat/ooc reason=%q, want %q", decision.Reason, tt.wantReason)
			}
		})
	}
}

func TestCanActOOCMessageBlockedWhenShowingClosed(t *testing.T) {
	decision, err := CanAct(context.Background(), fakeQuerier{role: "producer", showingStatus: "closed"}, "user-1", "chat/ooc", "session-1", ActionTarget{Kind: "session"})
	if err != nil {
		t.Fatalf("CanAct chat/ooc returned error: %v", err)
	}
	if decision.Allowed || decision.Reason != "showing_closed" {
		t.Fatalf("expected chat/ooc to be blocked when showing is closed, got %+v", decision)
	}
}

func TestStoreOOCMessageValidation(t *testing.T) {
	ctx := context.Background()

	if _, err := StoreOOCMessage(ctx, nil, OOCMessageRequest{ActorID: "user-1", Text: "hi"}); err == nil || err.Error() != "session_id is required" {
		t.Fatalf("expected session_id validation error, got %v", err)
	}
	if _, err := StoreOOCMessage(ctx, nil, OOCMessageRequest{SessionID: "session-1", Text: "hi"}); err == nil || err.Error() != "actor_id is required" {
		t.Fatalf("expected actor_id validation error, got %v", err)
	}
	if _, err := StoreOOCMessage(ctx, nil, OOCMessageRequest{SessionID: "session-1", ActorID: "user-1"}); err == nil || err.Error() != "text is required" {
		t.Fatalf("expected text validation error, got %v", err)
	}

	long := make([]byte, 251)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := StoreOOCMessage(ctx, nil, OOCMessageRequest{SessionID: "session-1", ActorID: "user-1", Text: string(long)}); err == nil || err.Error() != "message_too_long" {
		t.Fatalf("expected message_too_long validation error, got %v", err)
	}
}

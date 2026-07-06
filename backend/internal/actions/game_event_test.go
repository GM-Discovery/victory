package actions

import (
	"context"
	"testing"
)

func TestStoreGameEventRequiresSessionID(t *testing.T) {
	_, err := StoreGameEvent(context.Background(), nil, GameEventRequest{ActorID: "user-1", EventKind: "character_skill_added"})
	if err == nil || err.Error() != "session_id is required" {
		t.Fatalf("err = %v, want session_id is required", err)
	}
}

func TestStoreGameEventRequiresActorID(t *testing.T) {
	_, err := StoreGameEvent(context.Background(), nil, GameEventRequest{SessionID: "session-1", EventKind: "character_skill_added"})
	if err == nil || err.Error() != "actor_id is required" {
		t.Fatalf("err = %v, want actor_id is required", err)
	}
}

func TestStoreGameEventRequiresEventKind(t *testing.T) {
	_, err := StoreGameEvent(context.Background(), nil, GameEventRequest{SessionID: "session-1", ActorID: "user-1"})
	if err == nil || err.Error() != "event_kind is required" {
		t.Fatalf("err = %v, want event_kind is required", err)
	}
}

package network

import (
	"strings"
	"testing"

	"victory/backend/internal/actions"
)

func TestFormatDiscordChatMirrorContentIncludesPersonaAndRole(t *testing.T) {
	action := &actions.StoredAction{
		ID:               "action-1",
		ActorID:          "user-1",
		ActorDisplayName: "Grant",
		ActorHandle:      "grant",
		ActorRole:        "director",
		Persona: map[string]any{
			"display_name": "Xander",
		},
		Type: "chat/message",
		Payload: map[string]any{
			"text": "We should move before the lock opens.",
		},
	}

	content, truncated := formatDiscordChatMirrorContent(action)
	if truncated {
		t.Fatalf("expected content not to be truncated, got truncated")
	}
	want := "Grant / Xander [Director]: We should move before the lock opens."
	if content != want {
		t.Fatalf("unexpected content %q", content)
	}
}

func TestFormatDiscordChatMirrorContentTruncatesSafely(t *testing.T) {
	action := &actions.StoredAction{
		ID:               "action-2",
		ActorID:          "user-2",
		ActorDisplayName: "Grant",
		Type:             "chat/message",
		Payload: map[string]any{
			"text": strings.Repeat("a", 5000),
		},
	}

	content, truncated := formatDiscordChatMirrorContent(action)
	if !truncated {
		t.Fatalf("expected content to be truncated")
	}
	if len(content) > discordChatBridgeMaxContentRunes {
		t.Fatalf("expected content to fit in Discord limit, got length %d", len(content))
	}
	if !strings.HasPrefix(content, "Grant: ") {
		t.Fatalf("expected content prefix, got %q", content[:min(len(content), 32)])
	}
}

func TestDiscordChatBridgeEligibleVenues(t *testing.T) {
	for _, slug := range []string{"the-cave", "first-theater", "middle-school-stage"} {
		if _, ok := discordChatBridgeEligibleVenues[slug]; !ok {
			t.Fatalf("expected %s to be eligible", slug)
		}
	}
	for _, slug := range []string{"trailers", "workshop", "producers-office", "directors-chair"} {
		if _, ok := discordChatBridgeEligibleVenues[slug]; ok {
			t.Fatalf("expected %s to be ineligible", slug)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

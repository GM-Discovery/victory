package network

import "testing"

func TestSortPresenceForDirectorOrdersByRole(t *testing.T) {
	rows := []PresenceUser{
		{UserID: "a", Handle: "audience", DisplayName: "Audience Zoe", Role: "audience"},
		{UserID: "p", Handle: "producer", DisplayName: "Producer Mira", Role: "producer"},
		{UserID: "c", Handle: "cast", DisplayName: "Cast Eli", Role: "cast"},
		{UserID: "d", Handle: "director", DisplayName: "Director Liv", Role: "director"},
		{UserID: "crew", Handle: "crew", DisplayName: "Crew Sam", Role: "crew"},
	}

	sorted := sortPresenceForDirector(rows)
	got := []string{sorted[0].Role, sorted[1].Role, sorted[2].Role, sorted[3].Role, sorted[4].Role}
	want := []string{"producer", "director", "cast", "crew", "audience"}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected role order at %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestChatPolicyFromConfigDefaultsToTrue(t *testing.T) {
	policy := chatPolicyFromConfig(map[string]any{})
	if !policy.ChatEnabled || !policy.TalkingEnabled {
		t.Fatalf("expected chat policy defaults to true/true, got %+v", policy)
	}

	policy = chatPolicyFromConfig(map[string]any{
		"chat_enabled":    false,
		"talking_enabled": true,
	})
	if policy.ChatEnabled || !policy.TalkingEnabled {
		t.Fatalf("expected chat policy to reflect config, got %+v", policy)
	}
}

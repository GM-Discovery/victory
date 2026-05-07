package showings

import "testing"

func TestReviewCategoryAndHeadlineMapping(t *testing.T) {
	tests := []struct {
		actionType string
		category   string
		headline   string
	}{
		{actionType: "chat/message", category: "chat", headline: "Chat"},
		{actionType: "perform/speak", category: "speech", headline: "Stage Speech"},
		{actionType: "react/emote", category: "reactions", headline: "Reaction"},
		{actionType: "act/reveal_element", category: "reveal_hide", headline: "Reveal / Hide"},
		{actionType: "act/show_overlay", category: "overlay", headline: "Overlay"},
		{actionType: "create/index_card", category: "cards", headline: "Index Card"},
		{actionType: "persona/equip", category: "persona", headline: "Persona"},
	}

	for _, tt := range tests {
		t.Run(tt.actionType, func(t *testing.T) {
			if got := reviewCategoryForAction(tt.actionType); got != tt.category {
				t.Fatalf("reviewCategoryForAction(%q) = %q, want %q", tt.actionType, got, tt.category)
			}
			if got := reviewHeadlineForAction(tt.actionType); got != tt.headline {
				t.Fatalf("reviewHeadlineForAction(%q) = %q, want %q", tt.actionType, got, tt.headline)
			}
		})
	}
}

func TestHumanizeActionWord(t *testing.T) {
	if got := humanizeActionWord("standing_clap"); got != "Standing Clap" {
		t.Fatalf("humanizeActionWord returned %q, want %q", got, "Standing Clap")
	}
}


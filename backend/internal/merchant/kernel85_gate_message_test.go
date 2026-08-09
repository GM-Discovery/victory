package merchant

import "testing"

// TestTutorialGateMessageDistinguishesBlockedStates covers Kernel 85 §8.1:
// the locked courtyard door's two distinct blocked states ("no Character at
// all" vs. "has a Character but hasn't finished Kessa") must surface two
// distinct, correct Player-facing messages -- not the same generic prompt,
// and not the raw error code. The underlying gating error CODES
// (no_character_selected / not_a_roster_member / milestone_required) are
// asserted unchanged elsewhere (kernel74_tutorial_dbtest_test.go); this only
// covers the copy layered on top of them.
func TestTutorialGateMessageDistinguishesBlockedStates(t *testing.T) {
	noCharacterCases := []string{"no_character_selected", "not_a_roster_member"}
	for _, code := range noCharacterCases {
		got := tutorialGateMessage(code)
		if got != gateMessageNoCharacter {
			t.Fatalf("tutorialGateMessage(%q) = %q, want %q", code, got, gateMessageNoCharacter)
		}
	}

	got := tutorialGateMessage("milestone_required")
	if got != gateMessageKessaIncomplete {
		t.Fatalf("tutorialGateMessage(milestone_required) = %q, want %q", got, gateMessageKessaIncomplete)
	}

	if gateMessageNoCharacter == gateMessageKessaIncomplete {
		t.Fatal("the two blocked-state messages must be distinct")
	}

	// A code this function has no door-specific copy for must not fabricate
	// one -- the caller falls back to the code itself / other generic copy.
	if got := tutorialGateMessage("scene_not_current"); got != "" {
		t.Fatalf("expected no override for scene_not_current, got %q", got)
	}
}

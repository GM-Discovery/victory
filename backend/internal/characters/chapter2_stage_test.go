package characters

import "testing"

func TestApplyChapter2EnhancementDeclined(t *testing.T) {
	d6 := 6
	rolls := &Chapter2RollResult{D4: 2, D6: &d6}
	if got := ApplyChapter2Enhancement(rolls, false); got != 2 {
		t.Fatalf("ApplyChapter2Enhancement(declined) = %d, want 2 (original d4)", got)
	}
}

func TestApplyChapter2EnhancementAcceptedKeepsHigher(t *testing.T) {
	d6 := 6
	rolls := &Chapter2RollResult{D4: 2, D6: &d6}
	if got := ApplyChapter2Enhancement(rolls, true); got != 6 {
		t.Fatalf("ApplyChapter2Enhancement(accepted) = %d, want 6", got)
	}
}

func TestApplyChapter2EnhancementAcceptedButD4Higher(t *testing.T) {
	d6 := 1
	rolls := &Chapter2RollResult{D4: 4, D6: &d6}
	if got := ApplyChapter2Enhancement(rolls, true); got != 4 {
		t.Fatalf("ApplyChapter2Enhancement(d4 higher) = %d, want 4 (max kept)", got)
	}
}

func TestApplyChapter2EnhancementNoD6Rolled(t *testing.T) {
	rolls := &Chapter2RollResult{D4: 3, D6: nil}
	if got := ApplyChapter2Enhancement(rolls, true); got != 3 {
		t.Fatalf("ApplyChapter2Enhancement(no d6) = %d, want 3 (fallback to d4)", got)
	}
}

func TestLoadChapter2StateDefaultsFresh(t *testing.T) {
	s := loadChapter2State(map[string]any{})
	if s.StartingFP != Chapter2StartingFP || s.FPBalance != Chapter2StartingFP {
		t.Fatalf("fresh state FP = %d/%d, want %d/%d", s.StartingFP, s.FPBalance, Chapter2StartingFP, Chapter2StartingFP)
	}
	if s.CurrentStage != 1 {
		t.Fatalf("fresh state CurrentStage = %d, want 1", s.CurrentStage)
	}
	for _, attr := range AllChapter2Attributes {
		if s.Attributes[attr] != 0 {
			t.Fatalf("fresh state attribute %s = %d, want 0", attr, s.Attributes[attr])
		}
	}
}

func TestLoadChapter2StateRoundTrips(t *testing.T) {
	original := map[string]any{
		"chapter2": map[string]any{
			"version":       Chapter2Version,
			"starting_fp":   12,
			"fp_balance":    9,
			"current_stage": 2,
			"attributes":    map[string]any{"Spirit": 3},
			"stages":        map[string]any{},
		},
	}
	s := loadChapter2State(original)
	if s.FPBalance != 9 {
		t.Fatalf("FPBalance = %d, want 9 (loaded from existing context)", s.FPBalance)
	}
	if s.Attributes["Spirit"] != 3 {
		t.Fatalf("Spirit = %d, want 3", s.Attributes["Spirit"])
	}
}

package characters

import "testing"

func TestCommitChapter3ArchetypeRequiresAuth(t *testing.T) {
	_, _, err := CommitChapter3Archetype(nil, nil, "", "card-1", "Builder", false)
	if err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("err = %v, want not_authenticated", err)
	}
}

func TestCommitChapter3ArchetypeRequiresCardID(t *testing.T) {
	_, _, err := CommitChapter3Archetype(nil, nil, "user-1", "", "Builder", false)
	if err == nil || err.Error() != "character_card_id_required" {
		t.Fatalf("err = %v, want character_card_id_required", err)
	}
}

func TestCommitChapter3ArchetypeRequiresArchetypeKey(t *testing.T) {
	_, _, err := CommitChapter3Archetype(nil, nil, "user-1", "card-1", "", false)
	if err == nil || err.Error() != "archetype_key_required" {
		t.Fatalf("err = %v, want archetype_key_required", err)
	}
}

func TestCommitChapter3ArchetypeRejectsUnknownKey(t *testing.T) {
	_, _, err := CommitChapter3Archetype(nil, nil, "user-1", "card-1", "NotARealArchetype", false)
	if err == nil || err.Error() != "unknown_archetype_key" {
		t.Fatalf("err = %v, want unknown_archetype_key", err)
	}
}

func TestChapter2CompleteRequiresStage10(t *testing.T) {
	if chapter2Complete(map[string]any{}) {
		t.Fatalf("expected empty context to report chapter2 incomplete")
	}
	incomplete := map[string]any{
		"chapter2": map[string]any{
			"stages": map[string]any{
				"9": map[string]any{"completed": true},
			},
		},
	}
	if chapter2Complete(incomplete) {
		t.Fatalf("expected stage 9 alone to not satisfy chapter2 completion")
	}
	complete := map[string]any{
		"chapter2": map[string]any{
			"stages": map[string]any{
				"10": map[string]any{"completed": true},
			},
		},
	}
	if !chapter2Complete(complete) {
		t.Fatalf("expected stage 10 completed:true to satisfy chapter2 completion")
	}
}

func TestLoadChapter3FactRoundTrips(t *testing.T) {
	context := map[string]any{
		"chapter3": map[string]any{
			"archetype_key":   "Builder",
			"archetype_title": "The Builder",
			"confirmed":       true,
			"selection_count": float64(1),
		},
	}
	fact, ok := loadChapter3Fact(context)
	if !ok {
		t.Fatalf("expected confirmed fact to load")
	}
	if fact.ArchetypeKey != "Builder" || fact.SelectionCount != 1 {
		t.Fatalf("fact = %+v, unexpected fields", fact)
	}

	_, ok = loadChapter3Fact(map[string]any{})
	if ok {
		t.Fatalf("expected missing chapter3 key to report unconfirmed")
	}
}

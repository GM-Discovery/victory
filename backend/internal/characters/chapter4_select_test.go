package characters

import "testing"

func TestResolveChapter4GroupRequiresAuth(t *testing.T) {
	_, err := ResolveChapter4Group(nil, nil, "", "card-1")
	if err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("err = %v, want not_authenticated", err)
	}
}

func TestResolveChapter4GroupRequiresCardID(t *testing.T) {
	_, err := ResolveChapter4Group(nil, nil, "user-1", "")
	if err == nil || err.Error() != "character_card_id_required" {
		t.Fatalf("err = %v, want character_card_id_required", err)
	}
}

func TestCommitChapter4FirstSkillRequiresAuth(t *testing.T) {
	_, _, err := CommitChapter4FirstSkill(nil, nil, "", "card-1", "SKILL_CRAFT_CONSTRUCTION", false)
	if err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("err = %v, want not_authenticated", err)
	}
}

func TestCommitChapter4FirstSkillRequiresCardID(t *testing.T) {
	_, _, err := CommitChapter4FirstSkill(nil, nil, "user-1", "", "SKILL_CRAFT_CONSTRUCTION", false)
	if err == nil || err.Error() != "character_card_id_required" {
		t.Fatalf("err = %v, want character_card_id_required", err)
	}
}

func TestCommitChapter4FirstSkillRequiresSkillID(t *testing.T) {
	_, _, err := CommitChapter4FirstSkill(nil, nil, "user-1", "card-1", "", false)
	if err == nil || err.Error() != "skill_id_required" {
		t.Fatalf("err = %v, want skill_id_required", err)
	}
}

func TestCommitChapter4FirstSkillRejectsUnknownSkillID(t *testing.T) {
	_, _, err := CommitChapter4FirstSkill(nil, nil, "user-1", "card-1", "SKILL_NOT_REAL", false)
	if err == nil || err.Error() != "unknown_skill_id" {
		t.Fatalf("err = %v, want unknown_skill_id", err)
	}
}

func TestLoadChapter2AttributesRoundTrips(t *testing.T) {
	wbContext := map[string]any{
		"chapter2": map[string]any{
			"attributes": map[string]any{"Craft": float64(7), "Spirit": float64(3)},
		},
	}
	attrs := loadChapter2Attributes(wbContext)
	if attrs["Craft"] != 7 || attrs["Spirit"] != 3 {
		t.Fatalf("attrs = %+v, unexpected values", attrs)
	}
}

func TestLoadChapter2AttributesEmptyWhenMissing(t *testing.T) {
	attrs := loadChapter2Attributes(map[string]any{})
	if len(attrs) != 0 {
		t.Fatalf("expected empty map, got %+v", attrs)
	}
}

func TestLoadChapter4FactRoundTrips(t *testing.T) {
	wbContext := map[string]any{
		"chapter4": map[string]any{
			"skill_stable_id": "SKILL_CRAFT_CONSTRUCTION",
			"skill_name":      "Construction",
			"attribute_id":    "ATTR_CRAFT",
			"confirmed":       true,
			"selection_count": float64(1),
		},
	}
	fact, ok := loadChapter4Fact(wbContext)
	if !ok {
		t.Fatalf("expected confirmed fact to load")
	}
	if fact.SkillStableID != "SKILL_CRAFT_CONSTRUCTION" || fact.SelectionCount != 1 {
		t.Fatalf("fact = %+v, unexpected fields", fact)
	}

	if _, ok := loadChapter4Fact(map[string]any{}); ok {
		t.Fatalf("expected missing chapter4 key to report unconfirmed")
	}
}

func TestCountSelectedSkillsForAttribute(t *testing.T) {
	wbContext := map[string]any{
		"chapter4": map[string]any{
			"attribute_id": "ATTR_CRAFT",
			"confirmed":    true,
		},
	}
	if got := countSelectedSkillsForAttribute(wbContext, "ATTR_CRAFT"); got != 1 {
		t.Fatalf("count for matching attribute = %d, want 1", got)
	}
	if got := countSelectedSkillsForAttribute(wbContext, "ATTR_SPIRIT"); got != 0 {
		t.Fatalf("count for non-matching attribute = %d, want 0", got)
	}
	if got := countSelectedSkillsForAttribute(map[string]any{}, "ATTR_CRAFT"); got != 0 {
		t.Fatalf("count with no chapter4 fact = %d, want 0", got)
	}
}

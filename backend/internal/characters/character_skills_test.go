package characters

import "testing"

func TestAddCharacterSkillRequiresAuth(t *testing.T) {
	_, _, err := AddCharacterSkill(nil, nil, "", "card-1", "Alertness", nil)
	if err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("err = %v, want not_authenticated", err)
	}
}

func TestAddCharacterSkillRequiresCardID(t *testing.T) {
	_, _, err := AddCharacterSkill(nil, nil, "user-1", "", "Alertness", nil)
	if err == nil || err.Error() != "character_card_id_required" {
		t.Fatalf("err = %v, want character_card_id_required", err)
	}
}

func TestListCharacterSkillsRequiresAuth(t *testing.T) {
	_, err := ListCharacterSkills(nil, nil, "", "card-1")
	if err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("err = %v, want not_authenticated", err)
	}
}

func TestListCharacterSkillsRequiresCardID(t *testing.T) {
	_, err := ListCharacterSkills(nil, nil, "user-1", "")
	if err == nil || err.Error() != "character_card_id_required" {
		t.Fatalf("err = %v, want character_card_id_required", err)
	}
}

func TestResolveSkillByNameOrPrefixExact(t *testing.T) {
	skill, err := resolveSkillByNameOrPrefix("Archery")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill.Name != "Archery" {
		t.Fatalf("got %q, want Archery", skill.Name)
	}
}

func TestResolveSkillByNameOrPrefixUnambiguous(t *testing.T) {
	skill, err := resolveSkillByNameOrPrefix("archer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill.Name != "Archery" {
		t.Fatalf("got %q, want Archery", skill.Name)
	}
}

func TestResolveSkillByNameOrPrefixAmbiguous(t *testing.T) {
	// "Medi" matches Mediation, Medicine, and Meditation.
	_, err := resolveSkillByNameOrPrefix("Medi")
	if err == nil || err.Error() != "ambiguous_skill_name" {
		t.Fatalf("err = %v, want ambiguous_skill_name", err)
	}
}

func TestResolveSkillByNameOrPrefixUnknown(t *testing.T) {
	_, err := resolveSkillByNameOrPrefix("Zzzznotreal")
	if err == nil || err.Error() != "unknown_skill" {
		t.Fatalf("err = %v, want unknown_skill", err)
	}
}

func TestResolveSkillByNameOrPrefixRequiresName(t *testing.T) {
	_, err := resolveSkillByNameOrPrefix("   ")
	if err == nil || err.Error() != "skill_name_required" {
		t.Fatalf("err = %v, want skill_name_required", err)
	}
}

func TestSkillUnits(t *testing.T) {
	if got := skillUnits(false); got != 2 {
		t.Fatalf("full skill units = %d, want 2", got)
	}
	if got := skillUnits(true); got != 1 {
		t.Fatalf("helper skill units = %d, want 1", got)
	}
}

func TestFindCharacterSkillByNameRequiresName(t *testing.T) {
	_, err := FindCharacterSkillByName(nil, nil, "user-1", "card-1", "  ")
	if err == nil || err.Error() != "skill_name_required" {
		t.Fatalf("err = %v, want skill_name_required", err)
	}
}

func TestFindCharacterSkillByNameRequiresAuth(t *testing.T) {
	_, err := FindCharacterSkillByName(nil, nil, "", "card-1", "Alertness")
	if err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("err = %v, want not_authenticated", err)
	}
}

func TestFindCharacterSkillByNameRequiresCardID(t *testing.T) {
	_, err := FindCharacterSkillByName(nil, nil, "user-1", "", "Alertness")
	if err == nil || err.Error() != "character_card_id_required" {
		t.Fatalf("err = %v, want character_card_id_required", err)
	}
}

func TestRecordSkillAdvancementRequiresAuth(t *testing.T) {
	_, _, err := RecordSkillAdvancement(nil, nil, "", "card-1", "SKILL_X", "d4", 3)
	if err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("err = %v, want not_authenticated", err)
	}
}

func TestRecordSkillAdvancementRequiresCardID(t *testing.T) {
	_, _, err := RecordSkillAdvancement(nil, nil, "user-1", "", "SKILL_X", "d4", 3)
	if err == nil || err.Error() != "character_card_id_required" {
		t.Fatalf("err = %v, want character_card_id_required", err)
	}
}

func TestRecordSkillAdvancementRequiresSkillID(t *testing.T) {
	_, _, err := RecordSkillAdvancement(nil, nil, "user-1", "card-1", "  ", "d4", 3)
	if err == nil || err.Error() != "unknown_skill" {
		t.Fatalf("err = %v, want unknown_skill", err)
	}
}

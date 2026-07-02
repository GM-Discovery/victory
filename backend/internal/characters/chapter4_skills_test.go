package characters

import "testing"

func TestValidateChapter4SkillCatalogue(t *testing.T) {
	if err := ValidateChapter4SkillCatalogue(); err != nil {
		t.Fatalf("ValidateChapter4SkillCatalogue() = %v, want nil", err)
	}
}

func TestChapter4SkillsForAttributeReturnsTen(t *testing.T) {
	for _, attr := range Chapter4Attributes {
		skills := Chapter4SkillsForAttribute(attr.ID)
		if len(skills) != 10 {
			t.Fatalf("Chapter4SkillsForAttribute(%q) = %d skills, want 10", attr.ID, len(skills))
		}
	}
}

func TestChapter4SkillByName(t *testing.T) {
	skill, ok := Chapter4SkillByName("Construction")
	if !ok {
		t.Fatalf("expected Construction to resolve")
	}
	if skill.AttributeID != "ATTR_CRAFT" {
		t.Fatalf("Construction attribute = %q, want ATTR_CRAFT", skill.AttributeID)
	}

	if _, ok := Chapter4SkillByName("NotARealSkill"); ok {
		t.Fatalf("expected unknown skill name to not resolve")
	}
}

func TestChapter4SkillByID(t *testing.T) {
	skill, ok := Chapter4SkillByID("SKILL_GRACE_AESTHETIC_DESIGN")
	if !ok {
		t.Fatalf("expected SKILL_GRACE_AESTHETIC_DESIGN to resolve")
	}
	if skill.Name != "Aesthetic Design" || skill.AttributeName != "Grace" {
		t.Fatalf("skill = %+v, unexpected fields", skill)
	}
}

// TestEveryArchetypeKeySkillResolvesToTenCardGroup proves that, under the
// key-skill-attribute routing decision, every one of the 14 sample
// archetypes' key skills is guaranteed to appear in its own resolved
// ten-card group (since the group is derived FROM the key skill's
// attribute), closing out the routing conflict flagged in
// socio-skill-catalogue-audit-v0.1.md section 4/6.
func TestEveryArchetypeKeySkillResolvesToTenCardGroup(t *testing.T) {
	for _, archetype := range Chapter3Archetypes {
		skill, ok := Chapter4SkillByName(archetype.KeySkill)
		if !ok {
			t.Fatalf("archetype %q key skill %q does not resolve to any catalogue skill", archetype.Key, archetype.KeySkill)
		}
		group := Chapter4SkillsForAttribute(skill.AttributeID)
		found := false
		for _, s := range group {
			if s.ID == skill.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("archetype %q key skill %q not found in its own resolved group %q", archetype.Key, archetype.KeySkill, skill.AttributeID)
		}
	}
}

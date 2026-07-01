package characters

import "testing"

func TestValidateChapter3Archetypes(t *testing.T) {
	if err := ValidateChapter3Archetypes(); err != nil {
		t.Fatalf("ValidateChapter3Archetypes() = %v, want nil", err)
	}
}

func TestChapter3ArchetypeByKey(t *testing.T) {
	a, ok := Chapter3ArchetypeByKey("Builder")
	if !ok {
		t.Fatalf("expected Builder to exist")
	}
	if a.Title != "The Builder" || a.PrimaryAttribute != "Craft" || a.SecondaryAttribute != "Resolve" || a.KeySkill != "Construction" {
		t.Fatalf("Builder = %+v, unexpected fields", a)
	}

	if _, ok := Chapter3ArchetypeByKey("NotARealArchetype"); ok {
		t.Fatalf("expected unknown key to not resolve")
	}
}

func TestChapter3ArchetypesDisplayOrderIsContiguous(t *testing.T) {
	seen := make(map[int]bool, len(Chapter3Archetypes))
	for _, a := range Chapter3Archetypes {
		seen[a.DisplayOrder] = true
	}
	for i := 1; i <= len(Chapter3Archetypes); i++ {
		if !seen[i] {
			t.Fatalf("display_order %d missing from catalog", i)
		}
	}
}

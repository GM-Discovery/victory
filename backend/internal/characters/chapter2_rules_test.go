package characters

import "testing"

func TestValidateChapter2Rules(t *testing.T) {
	if err := ValidateChapter2Rules(); err != nil {
		t.Fatalf("ValidateChapter2Rules() = %v, want nil", err)
	}
}

func TestChapter2StageForNumber(t *testing.T) {
	stage, ok := Chapter2StageForNumber(9)
	if !ok {
		t.Fatalf("expected stage 9 to exist")
	}
	if stage.Name != "High School" || !stage.HasSpecialAllocation {
		t.Fatalf("stage 9 = %+v, want High School with special allocation", stage)
	}

	if _, ok := Chapter2StageForNumber(11); ok {
		t.Fatalf("expected stage 11 to not exist")
	}
}

func TestChapter2BonusChoiceByID(t *testing.T) {
	bc, ok := Chapter2BonusChoiceByID("BONUS_S05_C")
	if !ok {
		t.Fatalf("expected BONUS_S05_C to exist")
	}
	if bc.Name != "Helper" || bc.TargetAttribute != "Craft" {
		t.Fatalf("bonus = %+v, want Helper -> Craft", bc)
	}
}

func TestChapter2TraitByID(t *testing.T) {
	trait, ok := Chapter2TraitByID("TRAIT_S09_02_TECHNICAL_SAVANT")
	if !ok {
		t.Fatalf("expected trait to exist")
	}
	if trait.CostFP != 4 {
		t.Fatalf("trait cost = %d, want 4", trait.CostFP)
	}
}

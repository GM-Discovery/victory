package characters

import (
	"testing"

	"victory/backend/internal/dice"
)

func TestSkillLadderStepsParseAndRoll(t *testing.T) {
	for step := range SkillLadder {
		for _, explode := range []bool{true, false} {
			expr, err := StepExpression(step, explode)
			if err != nil {
				t.Fatalf("step %d explode=%v: StepExpression error: %v", step, explode, err)
			}
			spec, _, err := dice.ParseExpression(expr)
			if err != nil {
				t.Fatalf("step %d expr %q: ParseExpression error: %v", step, expr, err)
			}
			for _, g := range spec.Groups {
				if g.ExplodeOnMax != explode {
					t.Fatalf("step %d expr %q: group explode=%v, want %v", step, expr, g.ExplodeOnMax, explode)
				}
			}
		}
	}
}

func TestSkillLadderOutOfRange(t *testing.T) {
	if _, err := StepExpression(-1, false); err == nil {
		t.Fatal("expected error for negative step")
	}
	if _, err := StepExpression(SkillLadderMaxStep+1, false); err == nil {
		t.Fatal("expected error for step beyond max")
	}
}

func TestSkillLadderMeansMonotonic(t *testing.T) {
	for i := 1; i < len(SkillLadder); i++ {
		if SkillLadder[i].Mean <= SkillLadder[i-1].Mean {
			t.Fatalf("step %d mean %v not greater than step %d mean %v", i, SkillLadder[i].Mean, i-1, SkillLadder[i-1].Mean)
		}
	}
}

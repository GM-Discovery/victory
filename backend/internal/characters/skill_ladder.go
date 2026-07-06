package characters

import "fmt"

// SkillLadderStep is one rung of the canonical Socio- skill dice ladder
// (Kernel 60 §4), ordered by mean roll value per owner ruling — the source
// chart's placement of 4d12 and 5d12 was a math error; they sit here where
// their averages put them (step 19 and step 23).
type SkillLadderStep struct {
	Groups []dieGroupSpec
	Mean   float64
}

type dieGroupSpec struct {
	Count int
	Sides int
}

// SkillLadder is the canonical 36-step ladder (step 0 = d4 untrained surface
// through step 35 = 5d20). Index into the slice with the character skill's
// stored ladder_step.
var SkillLadder = []SkillLadderStep{
	{Groups: []dieGroupSpec{{1, 4}}, Mean: 2.5},
	{Groups: []dieGroupSpec{{1, 6}}, Mean: 3.5},
	{Groups: []dieGroupSpec{{1, 8}}, Mean: 4.5},
	{Groups: []dieGroupSpec{{1, 10}}, Mean: 5.5},
	{Groups: []dieGroupSpec{{1, 12}}, Mean: 6.5},
	{Groups: []dieGroupSpec{{1, 10}, {1, 4}}, Mean: 8},
	{Groups: []dieGroupSpec{{1, 10}, {1, 6}}, Mean: 9},
	{Groups: []dieGroupSpec{{1, 10}, {1, 8}}, Mean: 10},
	{Groups: []dieGroupSpec{{2, 10}}, Mean: 11},
	{Groups: []dieGroupSpec{{1, 10}, {1, 12}}, Mean: 12},
	{Groups: []dieGroupSpec{{1, 20}, {1, 4}}, Mean: 13},
	{Groups: []dieGroupSpec{{1, 20}, {1, 6}}, Mean: 14},
	{Groups: []dieGroupSpec{{1, 20}, {1, 8}}, Mean: 15},
	{Groups: []dieGroupSpec{{1, 20}, {1, 10}}, Mean: 16},
	{Groups: []dieGroupSpec{{1, 20}, {1, 12}}, Mean: 17},
	{Groups: []dieGroupSpec{{3, 12}}, Mean: 19.5},
	{Groups: []dieGroupSpec{{2, 20}, {1, 4}}, Mean: 23.5},
	{Groups: []dieGroupSpec{{2, 20}, {1, 6}}, Mean: 24.5},
	{Groups: []dieGroupSpec{{2, 20}, {1, 8}}, Mean: 25.5},
	{Groups: []dieGroupSpec{{4, 12}}, Mean: 26}, // moved from the chart's level-29 slot
	{Groups: []dieGroupSpec{{2, 20}, {1, 10}}, Mean: 26.5},
	{Groups: []dieGroupSpec{{2, 20}, {1, 12}}, Mean: 27.5},
	{Groups: []dieGroupSpec{{3, 20}}, Mean: 31.5},
	{Groups: []dieGroupSpec{{5, 12}}, Mean: 32.5}, // moved from the chart's level-37 slot
	{Groups: []dieGroupSpec{{3, 20}, {1, 4}}, Mean: 34},
	{Groups: []dieGroupSpec{{3, 20}, {1, 6}}, Mean: 35},
	{Groups: []dieGroupSpec{{3, 20}, {1, 8}}, Mean: 36},
	{Groups: []dieGroupSpec{{3, 20}, {1, 10}}, Mean: 37},
	{Groups: []dieGroupSpec{{3, 20}, {1, 12}}, Mean: 38},
	{Groups: []dieGroupSpec{{4, 20}}, Mean: 42},
	{Groups: []dieGroupSpec{{4, 20}, {1, 4}}, Mean: 44.5},
	{Groups: []dieGroupSpec{{4, 20}, {1, 6}}, Mean: 45.5},
	{Groups: []dieGroupSpec{{4, 20}, {1, 8}}, Mean: 46.5},
	{Groups: []dieGroupSpec{{4, 20}, {1, 10}}, Mean: 47.5},
	{Groups: []dieGroupSpec{{4, 20}, {1, 12}}, Mean: 48.5},
	{Groups: []dieGroupSpec{{5, 20}}, Mean: 52.5},
}

// SkillLadderMaxStep is the top of the ladder (5d20); /char advance returns
// skill_at_ladder_cap once a skill is here.
var SkillLadderMaxStep = len(SkillLadder) - 1

func formatGroups(groups []dieGroupSpec, explode bool) string {
	suffix := ""
	if explode {
		suffix = "!"
	}
	out := ""
	for i, g := range groups {
		if i > 0 {
			out += "+"
		}
		count := ""
		if g.Count != 1 {
			count = fmt.Sprintf("%d", g.Count)
		}
		out += fmt.Sprintf("%sd%d%s", count, g.Sides, suffix)
	}
	return out
}

// StepExpression returns the dice expression for a ladder step. explode
// selects the exploding form (each die group suffixed "!") used for normal
// skill checks, vs. the plain form used for /char advance rolls (Kernel 60
// §11 item 7 — advancement never explodes; a max face is already a
// disqualifying result).
func StepExpression(step int, explode bool) (string, error) {
	if step < 0 || step > SkillLadderMaxStep {
		return "", fmt.Errorf("ladder_step_out_of_range")
	}
	return formatGroups(SkillLadder[step].Groups, explode), nil
}

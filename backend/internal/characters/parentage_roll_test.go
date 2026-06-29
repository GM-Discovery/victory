package characters

import "testing"

type fixedRandomSource struct {
	values []int
	index  int
}

func (f *fixedRandomSource) Intn(maxExclusive int) (int, error) {
	v := f.values[f.index]
	f.index++
	return v, nil
}

func TestRollExplodingD20SingleNoExplosion(t *testing.T) {
	source := &fixedRandomSource{values: []int{4}} // face 5
	die, err := rollExplodingD20Single(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(die.Chain) != 1 || die.Chain[0] != 5 {
		t.Fatalf("chain = %v, want [5]", die.Chain)
	}
	if die.Subtotal != 5 {
		t.Fatalf("subtotal = %d, want 5", die.Subtotal)
	}
}

func TestRollExplodingD20SingleExplodesOnce(t *testing.T) {
	// First raw 19 -> face 20 (explodes). Second raw 19 -> face 20 again,
	// but the explosion die must not explode further.
	source := &fixedRandomSource{values: []int{19, 19}}
	die, err := rollExplodingD20Single(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(die.Chain) != 2 {
		t.Fatalf("chain = %v, want exactly 2 entries (base + one explosion)", die.Chain)
	}
	if die.Chain[0] != 20 || die.Chain[1] != 20 {
		t.Fatalf("chain = %v, want [20 20]", die.Chain)
	}
	if die.Subtotal != 40 {
		t.Fatalf("subtotal = %d, want 40", die.Subtotal)
	}
}

func TestComposeParentageRollResultUsesCanonicalChart(t *testing.T) {
	result, err := composeParentageRollResult("egg_donor", parentageRollRow{RollTotal: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["social_class"] != "Abandoned" {
		t.Fatalf("social_class = %v, want Abandoned", result["social_class"])
	}
	if result["starting_credit"] != 0 {
		t.Fatalf("starting_credit = %v, want 0", result["starting_credit"])
	}
}

func TestAttachCanonicalCatharsisParentageRowsRequiresDraftToken(t *testing.T) {
	_, err := attachCanonicalCatharsisParentageRows(nil, nil, "user-1", CharacterCardInput{
		WorkbookContext: map[string]any{"source": "catharsis"},
	})
	if err == nil || err.Error() != "draft_token_required" {
		t.Fatalf("err = %v, want draft_token_required", err)
	}
}

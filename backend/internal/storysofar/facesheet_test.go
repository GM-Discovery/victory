package storysofar

import (
	"fmt"
	"testing"
)

// canonicalOrder mirrors characters.AllChapter2Attributes. Duplicated here
// rather than imported because storysofar is a leaf package (see the package
// doc); if the canonical list ever changes, the dbtests exercising the real
// injected Rules will catch the drift.
var canonicalOrder = []string{
	"Spirit", "Might", "Empathy", "Grace", "Awareness",
	"Intellect", "Lore", "Presence", "Craft", "Resolve",
}

var testArchetypes = map[string]ArchetypeRule{
	"Observer": {Key: "Observer", Title: "The Observer", PrimaryAttribute: "Empathy", SecondaryAttribute: "Awareness", KeySkill: "Insight"},
	"Guardian": {Key: "Guardian", Title: "The Guardian", PrimaryAttribute: "Resolve", SecondaryAttribute: "Might", KeySkill: "Protection"},
}

func testRules() Rules {
	return Rules{
		AttributeOrder: canonicalOrder,
		Archetype: func(key string) (ArchetypeRule, bool) {
			a, ok := testArchetypes[key]
			return a, ok
		},
	}
}

func TestSelectPrimaryLensUsesArchetypeMapping(t *testing.T) {
	in := Inputs{
		CharacterName:    "Trang",
		ArchetypeKey:     "Observer",
		PrimaryAttribute: "Empathy",
		// Intellect scores higher, but S1.7 puts the archetype mapping
		// first: the archetype selects the lens, the scores do not.
		Attributes: map[string]int{"Empathy": 6, "Intellect": 9},
	}
	lens := SelectPrimaryLens(in, testRules())
	if lens.Attribute != "Empathy" {
		t.Fatalf("archetype mapping must win: got %q, want Empathy", lens.Attribute)
	}
	if !lens.FromArchetype {
		t.Fatal("FromArchetype must be true when the archetype chose the lens")
	}
}

func TestSelectPrimaryLensFallsBackToHighestWhenArchetypeMissing(t *testing.T) {
	in := Inputs{
		CharacterName: "Mara",
		Attributes:    map[string]int{"Empathy": 4, "Intellect": 9, "Craft": 7},
	}
	lens := SelectPrimaryLens(in, testRules())
	if lens.Attribute != "Intellect" {
		t.Fatalf("no archetype -> highest attribute: got %q, want Intellect", lens.Attribute)
	}
	if lens.FromArchetype {
		t.Fatal("FromArchetype must be false on the highest-score fallback")
	}
}

func TestSelectPrimaryLensFallsBackWhenArchetypeAttributeUnscored(t *testing.T) {
	// S1.7: "If the archetype mapping is absent or INVALID, use the highest
	// attribute." An archetype naming an attribute this Character has no
	// score for cannot back a sentence asserting it as a strength.
	in := Inputs{
		ArchetypeKey:     "Observer",
		PrimaryAttribute: "Empathy",
		Attributes:       map[string]int{"Intellect": 8, "Craft": 5},
	}
	lens := SelectPrimaryLens(in, testRules())
	if lens.Attribute != "Intellect" {
		t.Fatalf("unscored archetype attribute is invalid: got %q, want Intellect", lens.Attribute)
	}
}

func TestSelectPrimaryLensTieResolvedThroughArchetype(t *testing.T) {
	// S1.7: "If several attributes tie, prefer the attribute connected to
	// the Character's archetype." Awareness comes earlier in canonical order
	// than Empathy, so without the archetype rule it would win.
	in := Inputs{
		ArchetypeKey:     "Observer",
		PrimaryAttribute: "",
		Attributes:       map[string]int{"Awareness": 7, "Empathy": 7, "Craft": 7},
	}
	lens := SelectPrimaryLens(in, testRules())
	if lens.Attribute != "Empathy" {
		t.Fatalf("tie must resolve through archetype: got %q, want Empathy", lens.Attribute)
	}
}

func TestSelectPrimaryLensTieWithoutArchetypeIsCanonicalOrder(t *testing.T) {
	in := Inputs{Attributes: map[string]int{"Resolve": 5, "Spirit": 5, "Craft": 5}}
	lens := SelectPrimaryLens(in, testRules())
	// Spirit is first in canonical order.
	if lens.Attribute != "Spirit" {
		t.Fatalf("tie without archetype -> canonical order: got %q, want Spirit", lens.Attribute)
	}
}

func TestSelectPrimaryLensOmitsLineWhenNoFaceSheetHistory(t *testing.T) {
	// S1.7: "If the relevant Face Sheet line is missing, omit that quotation
	// rather than fabricating one." Every Character created before the
	// onboarding wrote history entries lands here.
	in := Inputs{
		ArchetypeKey:     "Observer",
		PrimaryAttribute: "Empathy",
		Attributes:       map[string]int{"Empathy": 7},
	}
	lens := SelectPrimaryLens(in, testRules())
	if lens.HasLine {
		t.Fatal("HasLine must be false when the Character has no stage lines")
	}
}

func TestSelectPrimaryLensMatchesFaceSheetLine(t *testing.T) {
	in := Inputs{
		ArchetypeKey:     "Observer",
		PrimaryAttribute: "Empathy",
		Attributes:       map[string]int{"Empathy": 7},
		StageLines: []StageLine{
			{ID: "a", Title: "Stage 1: Childhood", Body: "You learned to build things.", StageNumber: 1},
			{ID: "b", Title: "Stage 3: Youth", Body: "Empathy came early to you.", StageNumber: 3},
		},
	}
	lens := SelectPrimaryLens(in, testRules())
	if !lens.HasLine || lens.Line.ID != "b" {
		t.Fatalf("expected the Empathy line, got %+v", lens.Line)
	}
}

func TestSelectPrimaryLineWholeWordMatchingOnly(t *testing.T) {
	// "Lore" must not match inside "explore". A substring match here would
	// attach a Face Sheet quotation to a strength it does not evidence --
	// exactly the unsupported claim S5.2 forbids.
	in := Inputs{
		PrimaryAttribute: "Lore",
		Attributes:       map[string]int{"Lore": 8},
		StageLines: []StageLine{
			{ID: "a", Title: "Stage 2", Body: "You loved to explore the woods.", StageNumber: 2},
		},
	}
	lens := SelectPrimaryLens(in, testRules())
	if lens.HasLine {
		t.Fatalf("substring match must not count: matched %q", lens.Line.Body)
	}
}

func TestContainsWholeWord(t *testing.T) {
	cases := []struct {
		hay, needle string
		want        bool
	}{
		{"Empathy came early", "Empathy", true},
		{"empathy came early", "Empathy", true},
		{"You loved to explore", "Lore", false},
		{"a crafty sort", "Craft", false},
		{"skilled at Craft, always", "Craft", true},
		{"Craft", "Craft", true},
		{"", "Craft", false},
		{"Craft", "", false},
	}
	for _, c := range cases {
		if got := containsWholeWord(c.hay, c.needle); got != c.want {
			t.Errorf("containsWholeWord(%q, %q) = %v, want %v", c.hay, c.needle, got, c.want)
		}
	}
}

// TestSelectPrimaryLensIsDeterministicAcrossMapOrder is the guard against
// this package's single most likely bug.
//
// Inputs.Attributes is a Go map, and ranging over a map yields a random
// order on every run. If that order could reach the output, two Continue
// presses would produce different summaries, the dedupe UNIQUE index in
// migration 070 would stop absorbing retries, and a Player's history would
// silently duplicate. Every tie-break in SelectPrimaryLens is total
// precisely so this test can pass.
func TestSelectPrimaryLensIsDeterministicAcrossMapOrder(t *testing.T) {
	stages := []StageLine{
		{ID: "l1", Title: "Stage 1", Body: "Craft and Resolve both mattered.", StageNumber: 1, SortOrder: 0},
		{ID: "l2", Title: "Stage 2", Body: "Resolve and Craft both mattered.", StageNumber: 2, SortOrder: 0},
		{ID: "l3", Title: "Stage 2", Body: "Craft alone.", StageNumber: 2, SortOrder: 1},
	}

	var firstAttr string
	var firstLine string
	for i := 0; i < 200; i++ {
		// Rebuild the map each iteration with a rotated insertion order, so
		// Go's per-map hash seed and the insertion sequence both vary.
		attrs := map[string]int{}
		names := []string{"Craft", "Resolve", "Spirit", "Might", "Empathy"}
		for j := range names {
			n := names[(i+j)%len(names)]
			attrs[n] = 7 // deliberately all tied -- the hardest case
		}
		in := Inputs{
			CharacterName: "Determinism",
			Attributes:    attrs,
			StageLines:    stages,
		}
		lens := SelectPrimaryLens(in, testRules())
		if i == 0 {
			firstAttr = lens.Attribute
			firstLine = lens.Line.ID
			continue
		}
		if lens.Attribute != firstAttr {
			t.Fatalf("iteration %d: attribute drifted %q -> %q", i, firstAttr, lens.Attribute)
		}
		if lens.Line.ID != firstLine {
			t.Fatalf("iteration %d: line drifted %q -> %q", i, firstLine, lens.Line.ID)
		}
	}
	if firstAttr == "" {
		t.Fatal("expected a lens to be selected")
	}
}

// TestSelectFaceSheetLineTieBreaksAreTotal exercises the documented ladder
// (d) higher StageNumber and (e) lower SortOrder, which only decide the
// result once (a)-(c) have tied.
func TestSelectFaceSheetLineTieBreaksAreTotal(t *testing.T) {
	base := Inputs{
		PrimaryAttribute: "Craft",
		Attributes:       map[string]int{"Craft": 6},
	}

	t.Run("higher stage number wins", func(t *testing.T) {
		in := base
		in.StageLines = []StageLine{
			{ID: "a", Body: "Craft mattered.", StageNumber: 1, SortOrder: 0},
			{ID: "b", Body: "Craft mattered.", StageNumber: 5, SortOrder: 0},
		}
		lens := SelectPrimaryLens(in, testRules())
		if lens.Line.ID != "b" {
			t.Fatalf("got %q, want b", lens.Line.ID)
		}
	})

	t.Run("lower sort order wins within a stage", func(t *testing.T) {
		in := base
		in.StageLines = []StageLine{
			{ID: "a", Body: "Craft mattered.", StageNumber: 3, SortOrder: 4},
			{ID: "b", Body: "Craft mattered.", StageNumber: 3, SortOrder: 1},
		}
		lens := SelectPrimaryLens(in, testRules())
		if lens.Line.ID != "b" {
			t.Fatalf("got %q, want b", lens.Line.ID)
		}
	})

	t.Run("id breaks a full tie", func(t *testing.T) {
		in := base
		in.StageLines = []StageLine{
			{ID: "zz", Body: "Craft mattered.", StageNumber: 3, SortOrder: 1},
			{ID: "aa", Body: "Craft mattered.", StageNumber: 3, SortOrder: 1},
		}
		lens := SelectPrimaryLens(in, testRules())
		if lens.Line.ID != "aa" {
			t.Fatalf("got %q, want aa", lens.Line.ID)
		}
	})

	t.Run("attribute term beats key skill term", func(t *testing.T) {
		in := Inputs{
			ArchetypeKey:     "Observer",
			PrimaryAttribute: "Empathy",
			KeySkill:         "Insight",
			Attributes:       map[string]int{"Empathy": 6},
			StageLines: []StageLine{
				{ID: "skill", Body: "Insight served you well.", StageNumber: 9, SortOrder: 0},
				{ID: "attr", Body: "Empathy served you well.", StageNumber: 1, SortOrder: 0},
			},
		}
		lens := SelectPrimaryLens(in, testRules())
		// Candidate rank (a) outranks stage number (d): the attribute line
		// wins despite being an earlier life stage.
		if lens.Line.ID != "attr" {
			t.Fatalf("got %q, want attr", lens.Line.ID)
		}
	})
}

func TestSelectPrimaryLensNoAttributesAtAll(t *testing.T) {
	lens := SelectPrimaryLens(Inputs{}, testRules())
	if lens.Attribute != "" {
		t.Fatalf("expected no lens, got %q", lens.Attribute)
	}
	if lens.HasLine {
		t.Fatal("expected no line")
	}
}

func TestSelectPrimaryLensSurvivesMissingRules(t *testing.T) {
	// Defensive path: a caller that forgets Rules must still be
	// deterministic, so the failure shows up in a diff rather than
	// intermittently in production.
	in := Inputs{Attributes: map[string]int{"Zeta": 5, "Alpha": 5, "Mid": 5}}
	var first string
	for i := 0; i < 50; i++ {
		lens := SelectPrimaryLens(in, Rules{})
		if i == 0 {
			first = lens.Attribute
			continue
		}
		if lens.Attribute != first {
			t.Fatalf("missing Rules must still be deterministic: %q vs %q", first, lens.Attribute)
		}
	}
	if first != "Alpha" {
		t.Fatalf("expected alphabetical fallback, got %q", first)
	}
	_ = fmt.Sprint(first)
}

package storysofar

import (
	"strings"
	"testing"
	"time"
)

func fixedTime() time.Time { return time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC) }

func draftsByType(drafts []DraftEvent, eventType string) []DraftEvent {
	out := []DraftEvent{}
	for _, d := range drafts {
		if d.EventType == eventType {
			out = append(out, d)
		}
	}
	return out
}

func hasType(drafts []DraftEvent, eventType string) bool {
	return len(draftsByType(drafts, eventType)) > 0
}

// completeInputs is S5.4's Example A: a Character with everything recorded.
func completeInputs() Inputs {
	return Inputs{
		CharacterCardID:  "char-1",
		OwnerUserID:      "user-1",
		CharacterName:    "Trang",
		Pronouns:         "they/them",
		ArchetypeKey:     "Observer",
		ArchetypeTitle:   "The Observer",
		PrimaryAttribute: "Empathy",
		KeySkill:         "Insight",
		Attributes:       map[string]int{"Empathy": 8, "Awareness": 6},
		StageLines: []StageLine{
			{ID: "line-1", Title: "Stage 2: Childhood", Body: "You learned early to notice who was left outside the group, and Empathy came with it.", StageNumber: 2},
		},
		Milestones: map[string]bool{
			milestoneKessaCompleted: true,
			milestoneDoorSubmitted:  true,
			milestoneRaCompleted:    true,
			milestoneGateOpened:     true,
			milestoneTutorialDone:   true,
		},
		Inventory: []InventoryLine{
			{InventoryItemID: "inv-1", ItemName: "a healer's kit", Quantity: 1, AcquiredAt: fixedTime()},
			{InventoryItemID: "inv-2", ItemName: "a traveling cloak", Quantity: 1, AcquiredAt: fixedTime().Add(time.Minute)},
		},
		DoorIntention: "I study the hinges and test whether the door can be lifted instead of forced.",
		Attempts: []Attempt{
			{Kind: "stance", StanceKey: "Insight", Disposition: "curiosity", CreatedAt: fixedTime()},
			{Kind: "haggle", Success: true, Total: 14, TargetValue: 10, CreatedAt: fixedTime()},
		},
		Topics: []DialogueTopicSeen{
			{TopicID: "t1", TopicKey: "why-looking", Required: true},
			{TopicID: "t2", TopicKey: "crown-bet", Required: true},
			{TopicID: "t3", TopicKey: "beyond-the-door", Required: false},
		},
		ShowID:      "show-1",
		CompletedAt: fixedTime(),
	}
}

// sparseInputs is S5.4's Example B: a Character who completed the tutorial
// with almost nothing else recorded -- including a Character created before
// Kernel 75, which has NO character_interaction_attempts rows at all.
func sparseInputs() Inputs {
	return Inputs{
		CharacterCardID: "char-2",
		OwnerUserID:     "user-2",
		CharacterName:   "Mara",
		ArchetypeKey:    "Guardian",
		ArchetypeTitle:  "The Guardian",
		Attributes:      map[string]int{},
		Milestones: map[string]bool{
			milestoneKessaCompleted: true,
			milestoneDoorSubmitted:  true,
			milestoneRaCompleted:    true,
			milestoneGateOpened:     true,
			milestoneTutorialDone:   true,
		},
		ShowID:      "show-1",
		CompletedAt: fixedTime(),
	}
}

// TestGenerateIsPure is the property Continue's retry-safety rests on.
//
// The dedupe UNIQUE index in migration 070 can only absorb a retried
// Continue if the retry re-derives byte-identical (event_type, source_kind,
// source_ref) tuples. If Generate ever became clock-, random-, or
// map-order-dependent, retries would start writing second copies of a
// Player's history instead of colliding -- silently, and only in production.
func TestGenerateIsPure(t *testing.T) {
	rules := testRules()
	baseline := Generate(completeInputs(), rules)
	if len(baseline) == 0 {
		t.Fatal("expected clauses")
	}
	for i := 0; i < 200; i++ {
		// Fresh Inputs each iteration so the attribute map is rebuilt and
		// Go's map ordering varies.
		got := Generate(completeInputs(), rules)
		if len(got) != len(baseline) {
			t.Fatalf("iteration %d: clause count drifted %d -> %d", i, len(baseline), len(got))
		}
		for j := range got {
			if got[j] != baseline[j] {
				t.Fatalf("iteration %d clause %d drifted:\n old: %+v\n new: %+v", i, j, baseline[j], got[j])
			}
		}
	}
}

func TestGenerateDedupeKeysAreUnique(t *testing.T) {
	// The dedupe index is UNIQUE on (character, event_type, source_kind,
	// source_ref). If one Generate call produced two clauses with the same
	// triple, the second would be silently swallowed by ON CONFLICT DO
	// NOTHING and the Player would lose a line of their story.
	drafts := Generate(completeInputs(), testRules())
	seen := map[string]bool{}
	for _, d := range drafts {
		key := d.EventType + "\x00" + d.SourceKind + "\x00" + d.SourceRef
		if seen[key] {
			t.Fatalf("duplicate dedupe key within one Generate: %q", key)
		}
		seen[key] = true
	}
}

func TestGenerateCompleteDataProducesTheExpectedClauses(t *testing.T) {
	drafts := Generate(completeInputs(), testRules())
	for _, want := range []string{
		EventArrival, EventFaceSheetEcho, EventMerchantMet, EventMerchantStance,
		EventMerchantHaggle, EventEquipmentAcquired, EventDoorIntention,
		EventDialogueLearned, EventGateOpened, EventTutorialCompleted,
	} {
		if !hasType(drafts, want) {
			t.Errorf("missing clause %q", want)
		}
	}
}

func TestGenerateSparseDataOmitsRatherThanBlanks(t *testing.T) {
	drafts := Generate(sparseInputs(), testRules())
	if len(drafts) == 0 {
		t.Fatal("a sparse Character must still get a story")
	}
	// No attributes and no stage lines -> no Face Sheet quotation at all.
	if hasType(drafts, EventFaceSheetEcho) {
		t.Error("face_sheet_echo must be omitted when no line matches")
	}
	// Nothing may contain an empty-value artifact.
	for _, d := range drafts {
		for _, bad := range []string{"<nil>", "%!", "unknown", "  ", "\"\""} {
			if strings.Contains(d.Summary, bad) {
				t.Errorf("clause %q contains placeholder artifact %q: %s", d.EventType, bad, d.Summary)
			}
		}
		if strings.TrimSpace(d.Summary) == "" {
			t.Errorf("clause %q has an empty summary", d.EventType)
		}
	}
	if !hasType(drafts, EventTutorialCompleted) {
		t.Error("the completion anchor must always fire once the milestone exists")
	}
}

// TestGeneratePreKernel75CharacterDegradesCleanly is the regression guard for
// the one real cost of shipping durable attempt records without a backfill:
// a Character who played the tutorial under Kernel 74 has no stance or
// Haggle rows. That must read as a thinner story, never as a broken one.
func TestGeneratePreKernel75CharacterDegradesCleanly(t *testing.T) {
	in := completeInputs()
	in.Attempts = nil // exactly what a pre-K75 Character looks like

	drafts := Generate(in, testRules())
	if len(drafts) == 0 {
		t.Fatal("expected a story")
	}
	// The stance and haggle clauses still fire, but in their "did not"
	// forms, because Kessa was demonstrably met.
	stance := draftsByType(drafts, EventMerchantStance)
	if len(stance) != 1 || stance[0].SourceKind != SourceNone {
		t.Fatalf("expected the no-stance clause, got %+v", stance)
	}
	haggle := draftsByType(drafts, EventMerchantHaggle)
	if len(haggle) != 1 || haggle[0].SourceKind != SourceNone {
		t.Fatalf("expected the no-haggle clause, got %+v", haggle)
	}
	// And nothing claims a roll happened.
	for _, d := range drafts {
		if strings.Contains(strings.ToLower(d.Summary), "rolled") {
			t.Errorf("must not claim a roll that was never recorded: %s", d.Summary)
		}
	}
}

func TestGenerateStanceGroupIsMutuallyExclusive(t *testing.T) {
	withStance := completeInputs()
	if got := len(draftsByType(Generate(withStance, testRules()), EventMerchantStance)); got != 1 {
		t.Fatalf("stance clauses = %d, want exactly 1", got)
	}
	noStance := completeInputs()
	noStance.Attempts = []Attempt{{Kind: "haggle", Success: true, CreatedAt: fixedTime()}}
	if got := len(draftsByType(Generate(noStance, testRules()), EventMerchantStance)); got != 1 {
		t.Fatalf("stance clauses without a stance = %d, want exactly 1", got)
	}
}

func TestGenerateHaggleGroupIsMutuallyExclusive(t *testing.T) {
	cases := map[string][]Attempt{
		"success": {{Kind: "haggle", Success: true, CreatedAt: fixedTime()}},
		"failure": {{Kind: "haggle", Success: false, CreatedAt: fixedTime()}},
		"none":    {{Kind: "stance", StanceKey: "Insight", CreatedAt: fixedTime()}},
	}
	for name, attempts := range cases {
		in := completeInputs()
		in.Attempts = attempts
		got := draftsByType(Generate(in, testRules()), EventMerchantHaggle)
		if len(got) != 1 {
			t.Errorf("%s: haggle clauses = %d, want exactly 1", name, len(got))
		}
	}
}

func TestGenerateEquipmentGroupIsMutuallyExclusive(t *testing.T) {
	t.Run("no purchase yields exactly the S5.4 Example C clause", func(t *testing.T) {
		in := completeInputs()
		in.Inventory = nil
		got := draftsByType(Generate(in, testRules()), EventEquipmentAcquired)
		if len(got) != 1 {
			t.Fatalf("equipment clauses = %d, want 1", len(got))
		}
		if !strings.Contains(got[0].Summary, "did not require a purchase") {
			t.Fatalf("expected the no-purchase framing, got %q", got[0].Summary)
		}
	})

	t.Run("one item yields one clause", func(t *testing.T) {
		in := completeInputs()
		in.Inventory = in.Inventory[:1]
		got := draftsByType(Generate(in, testRules()), EventEquipmentAcquired)
		if len(got) != 1 {
			t.Fatalf("equipment clauses = %d, want 1", len(got))
		}
		if got[0].SourceRef != "inv-1" {
			t.Fatalf("expected the item to be attributable to its inventory row, got %q", got[0].SourceRef)
		}
	})

	t.Run("several items yield one clause each plus a summary", func(t *testing.T) {
		got := draftsByType(Generate(completeInputs(), testRules()), EventEquipmentAcquired)
		if len(got) != 3 {
			t.Fatalf("equipment clauses = %d, want 3 (2 items + summary)", len(got))
		}
		last := got[len(got)-1]
		if !strings.Contains(last.Summary, "a healer's kit and a traveling cloak") {
			t.Fatalf("summary must list both items in acquisition order, got %q", last.Summary)
		}
	})
}

// TestGenerateStoresDoorIntentionVerbatim guards Kernel 74 S1.4's settled
// rule. The Player's own words are never rewritten, never truncated, and
// never wrapped in authored narration -- a Player may type a complete
// sentence, so "You start to <input>" is unsafe grammar as well as rude.
func TestGenerateStoresDoorIntentionVerbatim(t *testing.T) {
	in := completeInputs()
	in.DoorIntention = "I refuse. <b>I wait</b> for whoever locked it & watch."

	got := draftsByType(Generate(in, testRules()), EventDoorIntention)
	if len(got) != 1 {
		t.Fatalf("door clauses = %d, want 1", len(got))
	}
	if !strings.Contains(got[0].Summary, in.DoorIntention) {
		t.Fatalf("intention must appear verbatim.\n want substring: %q\n got: %q", in.DoorIntention, got[0].Summary)
	}
}

func TestGenerateOmitsDoorClauseWhenNeverReached(t *testing.T) {
	in := sparseInputs()
	in.Milestones = map[string]bool{milestoneKessaCompleted: true}
	if hasType(Generate(in, testRules()), EventDoorIntention) {
		t.Error("no door milestone and no submission -> no door clause")
	}
}

func TestGenerateEmptyMilestonesProducesNothing(t *testing.T) {
	in := Inputs{CharacterName: "Nobody", Attributes: map[string]int{}, Milestones: map[string]bool{}}
	if got := Generate(in, testRules()); len(got) != 0 {
		t.Fatalf("a Character with no recorded milestones has no story yet, got %d clauses", len(got))
	}
}

func TestGenerateUsesRecordedPronouns(t *testing.T) {
	cases := map[string]string{
		"she/her":  "her",
		"he/him":   "his",
		"they/them": "their",
		"":         "their", // never inferred from a name
		"ze/zir":   "their",
	}
	for pronouns, want := range cases {
		in := completeInputs()
		in.Pronouns = pronouns
		if got := possessivePronoun(in); got != want {
			t.Errorf("possessivePronoun(%q) = %q, want %q", pronouns, got, want)
		}
	}
}

func TestGenerateOccurredAtDefaultsToCompletionTime(t *testing.T) {
	for _, d := range Generate(sparseInputs(), testRules()) {
		if d.OccurredAt.IsZero() {
			t.Errorf("clause %q has a zero occurred_at", d.EventType)
		}
	}
}

func TestJoinList(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{nil, ""},
		{[]string{"a"}, "a"},
		{[]string{"a", "b"}, "a and b"},
		{[]string{"a", "b", "c"}, "a, b, and c"},
	}
	for _, c := range cases {
		if got := joinList(c.in); got != c.want {
			t.Errorf("joinList(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

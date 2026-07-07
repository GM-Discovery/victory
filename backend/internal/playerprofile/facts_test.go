package playerprofile

import (
	"testing"
	"time"
)

func testCatalogueForFacts() Catalogue {
	return Catalogue{
		CatalogueKey:     "player-profile",
		CatalogueVersion: "1.0.0",
		Pages: []CataloguePage{
			{
				PageKey:   "identity_presentation",
				PageTitle: "Identity & Presentation",
				Fields: []CatalogueField{
					{FieldKey: "real_name", FieldLabel: "Real Name", FieldType: FieldTypeText, FaceEligible: true, FaceRegion: RegionIdentityHeader, DefaultPriority: 10},
					{FieldKey: FavoriteTTRPGsFieldKey, FieldLabel: "Favorite TTRPGs", FieldType: FieldTypeMultiSelectCustom, AllowCustom: true, MaxItems: FavoriteTTRPGsMaxItems, FaceEligible: true, FaceRegion: RegionAtAGlance, DefaultPriority: 50},
				},
			},
		},
	}
}

func mkEvent(id string, at time.Time, payload map[string]any) ProfileEvent {
	return ProfileEvent{ID: id, WorkbookID: "wb-1", EventType: "page_commit", PageKey: "identity_presentation", Payload: payload, CreatedAt: at}
}

func TestDeriveEffectiveFactsLatestEventWins(t *testing.T) {
	cat := testCatalogueForFacts()
	t0 := time.Now().Add(-2 * time.Hour)
	t1 := time.Now().Add(-1 * time.Hour)

	events := []ProfileEvent{
		mkEvent("e1", t0, map[string]any{"real_name": "Old Name"}),
		mkEvent("e2", t1, map[string]any{"real_name": "New Name"}),
	}

	facts := DeriveEffectiveFacts(cat, events)
	if facts["real_name"].DisplayValue != "New Name" {
		t.Fatalf("expected latest event to win, got %+v", facts["real_name"])
	}
	if facts["real_name"].SourceEventID != "e2" {
		t.Fatalf("expected source_event_id e2, got %s", facts["real_name"].SourceEventID)
	}
}

func TestDeriveEffectiveFactsDeletingLatestRevealsPrevious(t *testing.T) {
	cat := testCatalogueForFacts()
	t0 := time.Now().Add(-2 * time.Hour)
	t1 := time.Now().Add(-1 * time.Hour)

	all := []ProfileEvent{
		mkEvent("e1", t0, map[string]any{"real_name": "Old Name"}),
		mkEvent("e2", t1, map[string]any{"real_name": "New Name"}),
	}

	// Simulate deleting the newest event: the caller simply omits it from
	// the remaining-events list passed in (Kernel 61 §6.4, AC-16).
	remaining := all[:1]
	facts := DeriveEffectiveFacts(cat, remaining)
	if facts["real_name"].DisplayValue != "Old Name" {
		t.Fatalf("expected previous value to be revealed, got %+v", facts["real_name"])
	}
}

func TestDeriveEffectiveFactsRemovingOnlySourceRemovesFact(t *testing.T) {
	cat := testCatalogueForFacts()
	facts := DeriveEffectiveFacts(cat, nil)
	if _, ok := facts["real_name"]; ok {
		t.Fatalf("expected no fact when there are no remaining events, got %+v", facts["real_name"])
	}
}

func TestDeriveEffectiveFactsPreservesUnknownFieldKeys(t *testing.T) {
	cat := testCatalogueForFacts()
	events := []ProfileEvent{
		mkEvent("e1", time.Now(), map[string]any{"legacy_performance_age_range": "Mid 30s"}),
	}
	facts := DeriveEffectiveFacts(cat, events)
	if facts["legacy_performance_age_range"].DisplayValue != "Mid 30s" {
		t.Fatalf("expected unknown legacy key to be preserved as a private fact, got %+v", facts)
	}
}

func TestPageAnswersDifferFromFactsNoOp(t *testing.T) {
	current := map[string]ProfileFact{
		"real_name": {FieldKey: "real_name", ValueJSON: "Same Name", DisplayValue: "Same Name"},
	}
	answers := map[string]any{"real_name": "Same Name"}
	if PageAnswersDifferFromFacts(answers, current) {
		t.Fatalf("expected no material change to be detected")
	}
}

func TestPageAnswersDifferFromFactsDetectsChange(t *testing.T) {
	current := map[string]ProfileFact{
		"real_name": {FieldKey: "real_name", ValueJSON: "Old Name", DisplayValue: "Old Name"},
	}
	answers := map[string]any{"real_name": "New Name"}
	if !PageAnswersDifferFromFacts(answers, current) {
		t.Fatalf("expected material change to be detected")
	}
}

func TestPageAnswersDifferFromFactsNewEmptyFieldIsNoOp(t *testing.T) {
	current := map[string]ProfileFact{}
	answers := map[string]any{"real_name": ""}
	if PageAnswersDifferFromFacts(answers, current) {
		t.Fatalf("expected an empty new value with no existing fact to be a no-op")
	}
}

func TestBuildPageCommitSummaryUsesFieldLabels(t *testing.T) {
	cat := testCatalogueForFacts()
	page, _ := cat.PageByKey("identity_presentation")
	summary := BuildPageCommitSummary(page, map[string]any{"real_name": "New Name"})
	if summary == "" {
		t.Fatalf("expected non-empty summary")
	}
	if summary != "Updated Identity & Presentation: Real Name" {
		t.Fatalf("unexpected summary: %q", summary)
	}
}

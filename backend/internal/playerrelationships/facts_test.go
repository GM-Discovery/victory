package playerrelationships

import (
	"testing"
	"time"
)

func eventAt(id string, minute int, pageKey string, payload map[string]any) RelationshipEvent {
	return RelationshipEvent{
		ID:        id,
		EventType: "page_commit",
		PageKey:   pageKey,
		Payload:   payload,
		CreatedAt: time.Date(2026, 7, 9, 12, minute, 0, 0, time.UTC),
	}
}

func TestDeriveEffectiveFactsLaterEventWins(t *testing.T) {
	events := []RelationshipEvent{
		eventAt("e1", 0, "connection", map[string]any{"where_we_met": "Victory Theater"}),
		eventAt("e2", 5, "connection", map[string]any{"where_we_met": "Discord", "how_i_know_them": "Weekly game"}),
	}

	facts := DeriveEffectiveFacts(events)
	if facts["where_we_met"].DisplayValue != "Discord" {
		t.Fatalf("where_we_met = %q, want Discord", facts["where_we_met"].DisplayValue)
	}
	if facts["where_we_met"].SourceEventID != "e2" {
		t.Fatalf("source event = %q, want e2", facts["where_we_met"].SourceEventID)
	}
	if facts["how_i_know_them"].DisplayValue != "Weekly game" {
		t.Fatalf("how_i_know_them = %q", facts["how_i_know_them"].DisplayValue)
	}
}

func TestDeriveEffectiveFactsDeletionRevealsPrior(t *testing.T) {
	e1 := eventAt("e1", 0, "connection", map[string]any{"where_we_met": "Victory Theater"})
	e2 := eventAt("e2", 5, "connection", map[string]any{"where_we_met": "Discord"})

	// Deleting e2 (simply omitting it from the fold input) reveals e1's value
	// -- the deterministic-recompute behavior Kernel 62 §5.4 requires.
	facts := DeriveEffectiveFacts([]RelationshipEvent{e1})
	if facts["where_we_met"].DisplayValue != "Victory Theater" {
		t.Fatalf("after deletion, where_we_met = %q, want Victory Theater", facts["where_we_met"].DisplayValue)
	}
	_ = e2
}

func TestPageAnswersDifferFromFacts(t *testing.T) {
	current := DeriveEffectiveFacts([]RelationshipEvent{
		eventAt("e1", 0, "connection", map[string]any{"where_we_met": "Discord"}),
	})

	if PageAnswersDifferFromFacts(map[string]any{"where_we_met": "Discord"}, current) {
		t.Fatal("identical answers must be a no-op")
	}
	if !PageAnswersDifferFromFacts(map[string]any{"where_we_met": "In person"}, current) {
		t.Fatal("changed answer must register as different")
	}
	if PageAnswersDifferFromFacts(map[string]any{"how_i_know_them": ""}, current) {
		t.Fatal("new empty answer must not register as different")
	}
	if !PageAnswersDifferFromFacts(map[string]any{"how_i_know_them": "Game night"}, current) {
		t.Fatal("new non-empty answer must register as different")
	}
}

func TestBuildPageCommitSummaryUsesLabels(t *testing.T) {
	page := CataloguePage{
		PageKey:   "connection",
		PageTitle: "Connection",
		Fields: []CatalogueField{
			{FieldKey: "where_we_met", FieldLabel: "Where we met", FieldType: FieldTypeText},
		},
	}
	summary := BuildPageCommitSummary(page, map[string]any{"where_we_met": "Discord"})
	if summary != "Updated Connection: Where we met" {
		t.Fatalf("summary = %q", summary)
	}
}

func TestValidatePageAnswers(t *testing.T) {
	cat, err := LoadCatalogue()
	if err != nil {
		t.Fatalf("LoadCatalogue: %v", err)
	}
	page, _ := cat.PageByKey("connection")

	out, err := ValidatePageAnswers(page, map[string]any{"where_we_met": "  Victory Theater  "})
	if err != nil {
		t.Fatalf("ValidatePageAnswers: %v", err)
	}
	if out["where_we_met"] != "Victory Theater" {
		t.Fatalf("expected trimmed value, got %q", out["where_we_met"])
	}

	if _, err := ValidatePageAnswers(page, map[string]any{"secret_backdoor": "x"}); err == nil {
		t.Fatal("unknown field must be rejected")
	}
	if _, err := ValidatePageAnswers(page, map[string]any{"where_we_met": 42}); err == nil {
		t.Fatal("non-string value must be rejected")
	}
}

func TestValidateCategories(t *testing.T) {
	out, err := ValidateCategories([]Category{
		{CategoryKey: "friend"},
		{CategoryKey: "friend", CustomLabel: "ignored"},
		{CategoryKey: "custom", CustomLabel: "Bandmate"},
	})
	if err != nil {
		t.Fatalf("ValidateCategories: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 deduped categories, got %d: %+v", len(out), out)
	}
	if out[0].CategoryKey != "friend" || out[0].CustomLabel != "" {
		t.Fatalf("non-custom category must drop label: %+v", out[0])
	}
	if out[1].CustomLabel != "Bandmate" {
		t.Fatalf("custom label lost: %+v", out[1])
	}

	if _, err := ValidateCategories([]Category{{CategoryKey: "custom"}}); err == nil {
		t.Fatal("custom category without label must be rejected")
	}
	if _, err := ValidateCategories([]Category{{CategoryKey: "nemesis"}}); err == nil {
		t.Fatal("unknown category key must be rejected")
	}
}

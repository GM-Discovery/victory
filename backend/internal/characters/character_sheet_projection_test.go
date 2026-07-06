package characters

import "testing"

func TestApplyFaceOverridesDefaultsToInferredVisible(t *testing.T) {
	fields := []WorkbookPageField{
		{Key: "tagline", Label: "Featured Quote", PriorityScore: 70},
	}
	out := applyFaceOverrides(fields, map[string]FaceOverride{})
	if out[0].FaceVisibilityMode != faceVisibilityInferred {
		t.Fatalf("visibility mode = %q, want inferred", out[0].FaceVisibilityMode)
	}
	if !out[0].FaceVisible {
		t.Fatalf("expected inferred field to be visible by default")
	}
}

func TestApplyFaceOverridesHidesExplicitlyHidden(t *testing.T) {
	fields := []WorkbookPageField{
		{Key: "tagline", Label: "Featured Quote", PriorityScore: 70},
	}
	overrides := map[string]FaceOverride{
		"tagline": {FactKey: "tagline", VisibilityMode: faceVisibilityHidden},
	}
	out := applyFaceOverrides(fields, overrides)
	if out[0].FaceVisible {
		t.Fatalf("expected explicitly hidden field to be invisible")
	}
	if out[0].FaceVisibilityMode != faceVisibilityHidden {
		t.Fatalf("visibility mode = %q, want hidden", out[0].FaceVisibilityMode)
	}
}

func TestApplyFaceOverridesManualPriorityWins(t *testing.T) {
	fields := []WorkbookPageField{
		{Key: "tagline", Label: "Featured Quote", PriorityMode: "inferred", PriorityScore: 70},
	}
	overrides := map[string]FaceOverride{
		"tagline": {FactKey: "tagline", VisibilityMode: faceVisibilityShown, PriorityMode: facePriorityManual, PriorityScore: 999},
	}
	out := applyFaceOverrides(fields, overrides)
	if out[0].PriorityMode != facePriorityManual {
		t.Fatalf("priority mode = %q, want manual", out[0].PriorityMode)
	}
	if out[0].PriorityScore != 999 {
		t.Fatalf("priority score = %d, want 999", out[0].PriorityScore)
	}
}

func TestApplyFaceOverridesDirectorValueOverrideWins(t *testing.T) {
	fields := []WorkbookPageField{
		{Key: "tagline", Label: "Featured Quote", Value: "Original", SourceKind: "owner_explicit", PriorityScore: 70},
	}
	overrides := map[string]FaceOverride{
		"tagline": {FactKey: "tagline", ValueOverrideActive: true, ValueOverride: "Director value"},
	}
	out := applyFaceOverrides(fields, overrides)
	if out[0].Value != "Director value" {
		t.Fatalf("value = %q, want Director value", out[0].Value)
	}
	if out[0].SourceKind != "director_override" {
		t.Fatalf("source = %q, want director_override", out[0].SourceKind)
	}
}

func TestApplyFaceOverridesUnrelatedFieldUnaffected(t *testing.T) {
	fields := []WorkbookPageField{
		{Key: "name", Label: "Name", PriorityScore: 110},
		{Key: "tagline", Label: "Featured Quote", PriorityScore: 70},
	}
	overrides := map[string]FaceOverride{
		"tagline": {FactKey: "tagline", VisibilityMode: faceVisibilityHidden},
	}
	out := applyFaceOverrides(fields, overrides)
	if !out[0].FaceVisible {
		t.Fatalf("expected unrelated field 'name' to remain visible")
	}
}

func TestSortFaceFieldsOrdersByDescendingPriorityThenLabel(t *testing.T) {
	fields := []WorkbookPageField{
		{Key: "b", Label: "Bravo", PriorityScore: 50},
		{Key: "a", Label: "Alpha", PriorityScore: 90},
		{Key: "c", Label: "Charlie", PriorityScore: 50},
	}
	sortFaceFields(fields)
	got := []string{fields[0].Key, fields[1].Key, fields[2].Key}
	want := []string{"a", "b", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestResolveEffectiveFaceFieldsFiltersHiddenAndSorts(t *testing.T) {
	fields := []WorkbookPageField{
		{Key: "name", Label: "Name", Region: "identity", PriorityScore: 110},
		{Key: "tagline", Label: "Featured Quote", Region: "glance", PriorityScore: 70},
		{Key: "archetype", Label: "Archetype", Region: "identity", PriorityScore: 100},
	}
	overrides := map[string]FaceOverride{
		"tagline": {FactKey: "tagline", VisibilityMode: faceVisibilityHidden},
	}
	out := resolveEffectiveFaceFields(fields, overrides)
	if len(out) != 2 {
		t.Fatalf("expected 2 visible fields, got %d: %+v", len(out), out)
	}
	if out[0].Key != "name" || out[1].Key != "archetype" {
		t.Fatalf("unexpected order: %+v", out)
	}
}

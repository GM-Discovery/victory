package playerprofile

import "testing"

func TestIsReservedFieldKey(t *testing.T) {
	for _, key := range []string{ReservedFieldStageName, ReservedFieldHandle, ReservedFieldEmail, ReservedFieldAccountUUID} {
		if !IsReservedFieldKey(key) {
			t.Fatalf("expected %q to be reserved", key)
		}
	}
	if IsReservedFieldKey("real_name") {
		t.Fatalf("expected real_name to not be reserved")
	}
}

func testFaceCatalogue() Catalogue {
	return Catalogue{
		CatalogueKey:     "player-profile",
		CatalogueVersion: "1.0.0",
		Pages: []CataloguePage{
			{
				PageKey:   "identity_presentation",
				PageTitle: "Identity",
				Fields: []CatalogueField{
					{FieldKey: "real_name", FieldLabel: "Real Name", FieldType: FieldTypeText, FaceEligible: true, FaceRegion: RegionIdentityHeader, DefaultPriority: 10},
					{FieldKey: "about_text", FieldLabel: "About", FieldType: FieldTypeLongText, FaceEligible: true, FaceRegion: RegionAbout, DefaultPriority: 5},
					{FieldKey: "private_note", FieldLabel: "Private Note", FieldType: FieldTypeLongText, FaceEligible: false},
				},
			},
		},
	}
}

func TestBuildProjectedFieldsOnlyFaceEligible(t *testing.T) {
	cat := testFaceCatalogue()
	facts := map[string]ProfileFact{
		"real_name":    {FieldKey: "real_name", DisplayValue: "Alex"},
		"about_text":   {FieldKey: "about_text", DisplayValue: "Hello"},
		"private_note": {FieldKey: "private_note", DisplayValue: "secret"},
	}
	fields := BuildProjectedFields(cat, facts, nil)
	if len(fields) != 2 {
		t.Fatalf("expected only face-eligible facts to project, got %d: %+v", len(fields), fields)
	}
	for _, f := range fields {
		if f.FieldKey == "private_note" {
			t.Fatalf("private_note must never be projected")
		}
	}
}

func TestBuildProjectedFieldsDefaultsToInferredVisible(t *testing.T) {
	cat := testFaceCatalogue()
	facts := map[string]ProfileFact{"real_name": {FieldKey: "real_name", DisplayValue: "Alex"}}
	fields := BuildProjectedFields(cat, facts, nil)
	if len(fields) != 1 || !fields[0].FaceVisible {
		t.Fatalf("expected inferred field to be visible by default, got %+v", fields)
	}
}

func TestBuildProjectedFieldsHiddenOverrideWins(t *testing.T) {
	cat := testFaceCatalogue()
	facts := map[string]ProfileFact{"real_name": {FieldKey: "real_name", DisplayValue: "Alex"}}
	overrides := map[string]FaceOverride{"real_name": {FieldKey: "real_name", VisibilityMode: VisibilityHidden}}
	fields := BuildProjectedFields(cat, facts, overrides)
	if fields[0].FaceVisible {
		t.Fatalf("expected hidden override to suppress visibility")
	}
}

func TestBuildProjectedFieldsManualPriorityWins(t *testing.T) {
	cat := testFaceCatalogue()
	facts := map[string]ProfileFact{"about_text": {FieldKey: "about_text", DisplayValue: "Hello"}}
	overrides := map[string]FaceOverride{"about_text": {FieldKey: "about_text", PriorityMode: PriorityManual, PriorityScore: 999}}
	fields := BuildProjectedFields(cat, facts, overrides)
	if fields[0].PriorityMode != PriorityManual || fields[0].PriorityScore != 999 {
		t.Fatalf("expected manual priority override to win, got %+v", fields[0])
	}
}

func TestVisibleSortedFieldsOrdersByPriorityDescThenLabel(t *testing.T) {
	fields := []ProjectedField{
		{FieldKey: "a", Label: "Zeta", FaceVisible: true, PriorityScore: 10},
		{FieldKey: "b", Label: "Alpha", FaceVisible: true, PriorityScore: 50},
		{FieldKey: "c", Label: "Beta", FaceVisible: false, PriorityScore: 999},
	}
	sorted := VisibleSortedFields(fields)
	if len(sorted) != 2 {
		t.Fatalf("expected hidden field to be excluded, got %+v", sorted)
	}
	if sorted[0].FieldKey != "b" || sorted[1].FieldKey != "a" {
		t.Fatalf("expected descending priority order, got %+v", sorted)
	}
}

func TestGroupByRegionOmitsEmptyRegions(t *testing.T) {
	fields := []ProjectedField{
		{FieldKey: "a", Region: RegionIdentityHeader},
	}
	grouped := GroupByRegion(fields)
	if len(grouped) != 1 {
		t.Fatalf("expected exactly one populated region, got %+v", grouped)
	}
	if _, ok := grouped[RegionAbout]; ok {
		t.Fatalf("expected empty region to be absent from the map")
	}
}

func TestBuildTrailerFaceAlwaysIncludesStageName(t *testing.T) {
	face := BuildTrailerFace("v1", "Straturli", nil)
	if face.StageName != "Straturli" {
		t.Fatalf("expected stage name to always be present, got %+v", face)
	}
}

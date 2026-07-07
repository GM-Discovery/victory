package playerprofile

import "testing"

func testValidationPage() CataloguePage {
	return CataloguePage{
		PageKey:   "ttrpg_identity",
		PageTitle: "TTRPG Identity",
		Fields: []CatalogueField{
			{FieldKey: "dnd_class", FieldLabel: "D&D Class", FieldType: FieldTypeSingleSelectCustom, AllowCustom: true, Options: []string{"Bard", "Wizard"}},
			{FieldKey: "fixed_choice", FieldLabel: "Fixed Choice", FieldType: FieldTypeSingleSelect, Options: []string{"A", "B"}},
			{FieldKey: FavoriteTTRPGsFieldKey, FieldLabel: "Favorite TTRPGs", FieldType: FieldTypeMultiSelectCustom, AllowCustom: true, MaxItems: FavoriteTTRPGsMaxItems},
		},
	}
}

func TestValidatePageAnswersRejectsUnknownField(t *testing.T) {
	page := testValidationPage()
	_, err := ValidatePageAnswers(page, map[string]any{"not_a_field": "x"})
	if err == nil {
		t.Fatalf("expected error for unknown field")
	}
}

func TestValidatePageAnswersAllowsCustomDnDClass(t *testing.T) {
	page := testValidationPage()
	out, err := ValidatePageAnswers(page, map[string]any{"dnd_class": "Homebrew Class"})
	if err != nil {
		t.Fatalf("expected custom class to be accepted, got %v", err)
	}
	if out["dnd_class"] != "Homebrew Class" {
		t.Fatalf("expected trimmed custom value, got %+v", out)
	}
}

func TestValidatePageAnswersRejectsNonOptionWhenCustomDisallowed(t *testing.T) {
	page := testValidationPage()
	_, err := ValidatePageAnswers(page, map[string]any{"fixed_choice": "Not An Option"})
	if err == nil {
		t.Fatalf("expected error for option outside fixed list with no custom allowed")
	}
}

func TestValidatePageAnswersEnforcesFavoriteTTRPGsMax(t *testing.T) {
	page := testValidationPage()
	_, err := ValidatePageAnswers(page, map[string]any{
		FavoriteTTRPGsFieldKey: []any{"System A", "System B", "System C", "System D"},
	})
	if err == nil {
		t.Fatalf("expected error when submitting more than %d favorite TTRPGs", FavoriteTTRPGsMaxItems)
	}
}

func TestValidatePageAnswersAcceptsUpToThreeFavoriteTTRPGs(t *testing.T) {
	page := testValidationPage()
	out, err := ValidatePageAnswers(page, map[string]any{
		FavoriteTTRPGsFieldKey: []any{"System A", "System B", "System C"},
	})
	if err != nil {
		t.Fatalf("expected exactly three favorites to be accepted, got %v", err)
	}
	list, _ := out[FavoriteTTRPGsFieldKey].([]string)
	if len(list) != 3 {
		t.Fatalf("expected 3 favorites, got %+v", list)
	}
}

func TestValidatePageAnswersRequiredFieldMissing(t *testing.T) {
	page := testValidationPage()
	page.Fields = append(page.Fields, CatalogueField{FieldKey: "must_have", FieldLabel: "Must Have", FieldType: FieldTypeText, Required: true})
	_, err := ValidatePageAnswers(page, map[string]any{"dnd_class": "Bard"})
	if err == nil {
		t.Fatalf("expected error for missing required field")
	}
}

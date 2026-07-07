package playerprofile

import "testing"

func TestLoadCatalogueValid(t *testing.T) {
	cat, err := LoadCatalogue()
	if err != nil {
		t.Fatalf("LoadCatalogue() error = %v, want nil", err)
	}
	if len(cat.Pages) == 0 {
		t.Fatalf("expected catalogue to define pages")
	}
	if _, ok := cat.FieldByKey(FavoriteTTRPGsFieldKey); !ok {
		t.Fatalf("expected catalogue to define %q", FavoriteTTRPGsFieldKey)
	}
}

func baseCatalogue() Catalogue {
	return Catalogue{
		CatalogueKey:     "player-profile",
		CatalogueVersion: "1.0.0",
		Pages: []CataloguePage{
			{
				PageKey:   "page_one",
				PageTitle: "Page One",
				Fields: []CatalogueField{
					{FieldKey: FavoriteTTRPGsFieldKey, FieldLabel: "Favorite TTRPGs", FieldType: FieldTypeMultiSelectCustom, AllowCustom: true, MaxItems: FavoriteTTRPGsMaxItems},
				},
			},
		},
	}
}

func TestValidateCatalogueAcceptsBase(t *testing.T) {
	if err := ValidateCatalogue(baseCatalogue()); err != nil {
		t.Fatalf("expected base catalogue to validate, got %v", err)
	}
}

func TestValidateCatalogueRejectsDuplicatePageKeys(t *testing.T) {
	cat := baseCatalogue()
	cat.Pages = append(cat.Pages, cat.Pages[0])
	if err := ValidateCatalogue(cat); err == nil {
		t.Fatalf("expected error for duplicate page_key")
	}
}

func TestValidateCatalogueRejectsDuplicateFieldKeys(t *testing.T) {
	cat := baseCatalogue()
	cat.Pages[0].Fields = append(cat.Pages[0].Fields, cat.Pages[0].Fields[0])
	if err := ValidateCatalogue(cat); err == nil {
		t.Fatalf("expected error for duplicate field_key")
	}
}

func TestValidateCatalogueRejectsInvalidFieldType(t *testing.T) {
	cat := baseCatalogue()
	cat.Pages[0].Fields[0].FieldType = "not_a_real_type"
	if err := ValidateCatalogue(cat); err == nil {
		t.Fatalf("expected error for invalid field_type")
	}
}

func TestValidateCatalogueRejectsFaceEligibleWithoutRegion(t *testing.T) {
	cat := baseCatalogue()
	cat.Pages[0].Fields = append(cat.Pages[0].Fields, CatalogueField{
		FieldKey: "extra_field", FieldLabel: "Extra", FieldType: FieldTypeText, FaceEligible: true,
	})
	if err := ValidateCatalogue(cat); err == nil {
		t.Fatalf("expected error for face-eligible field without a valid region")
	}
}

func TestValidateCatalogueRejectsFavoriteTTRPGsWrongMax(t *testing.T) {
	cat := baseCatalogue()
	cat.Pages[0].Fields[0].MaxItems = 5
	if err := ValidateCatalogue(cat); err == nil {
		t.Fatalf("expected error when %s max_items != %d", FavoriteTTRPGsFieldKey, FavoriteTTRPGsMaxItems)
	}
}

func TestValidateCatalogueRequiresFavoriteTTRPGsField(t *testing.T) {
	cat := baseCatalogue()
	cat.Pages[0].Fields = nil
	if err := ValidateCatalogue(cat); err == nil {
		t.Fatalf("expected error when catalogue omits %s", FavoriteTTRPGsFieldKey)
	}
}

func TestValidateCatalogueRejectsCollectionFieldWithoutMaxItems(t *testing.T) {
	cat := baseCatalogue()
	cat.Pages[0].Fields = append(cat.Pages[0].Fields, CatalogueField{
		FieldKey: "some_multi", FieldLabel: "Some Multi", FieldType: FieldTypeMultiSelectCustom,
	})
	if err := ValidateCatalogue(cat); err == nil {
		t.Fatalf("expected error for collection field missing max_items")
	}
}

package playerrelationships

import "testing"

func TestLoadEmbeddedCatalogue(t *testing.T) {
	cat, err := LoadCatalogue()
	if err != nil {
		t.Fatalf("LoadCatalogue: %v", err)
	}
	if cat.CatalogueKey != "player-relationship" {
		t.Fatalf("catalogue_key = %q", cat.CatalogueKey)
	}

	// The four kernel workbook pages must exist (Kernel 62 §7.1-7.4).
	for _, pageKey := range []string{"connection", "understanding_them", "our_relationship", "shared_work_play"} {
		if _, ok := cat.PageByKey(pageKey); !ok {
			t.Fatalf("missing page %q", pageKey)
		}
	}
}

func TestValidateCatalogueRejectsDuplicates(t *testing.T) {
	base := Catalogue{
		CatalogueKey:     "test",
		CatalogueVersion: "1.0.0",
		Pages: []CataloguePage{
			{PageKey: "a", PageTitle: "A", Fields: []CatalogueField{
				{FieldKey: "x", FieldLabel: "X", FieldType: FieldTypeText},
			}},
			{PageKey: "b", PageTitle: "B", Fields: []CatalogueField{
				{FieldKey: "x", FieldLabel: "X again", FieldType: FieldTypeText},
			}},
		},
	}
	if err := ValidateCatalogue(base); err == nil {
		t.Fatal("expected duplicate field_key rejection")
	}
}

func TestValidateCatalogueRejectsUnknownFieldType(t *testing.T) {
	cat := Catalogue{
		CatalogueKey:     "test",
		CatalogueVersion: "1.0.0",
		Pages: []CataloguePage{
			{PageKey: "a", PageTitle: "A", Fields: []CatalogueField{
				{FieldKey: "x", FieldLabel: "X", FieldType: "spreadsheet"},
			}},
		},
	}
	if err := ValidateCatalogue(cat); err == nil {
		t.Fatal("expected invalid field_type rejection")
	}
}

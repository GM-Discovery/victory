package playerrelationships

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed catalogues/player-relationship-v1.0.0.json
var catalogueFS embed.FS

// CurrentCatalogueFile is the versioned relationship-workbook catalogue this
// build ships. A future revision adds a new file and bumps this constant
// rather than mutating v1.0.0 in place -- same discipline as the Kernel 61
// player-profile catalogue.
const CurrentCatalogueFile = "catalogues/player-relationship-v1.0.0.json"

// Field types the relationship workbook supports. The kernel's workbook
// pages are freeform notes fields (Kernel 62 §7); structured values
// (categories, qualitative dropdowns) are relationship columns, not
// catalogue fields.
const (
	FieldTypeText     = "text"
	FieldTypeLongText = "long_text"
)

var validFieldTypes = map[string]bool{
	FieldTypeText:     true,
	FieldTypeLongText: true,
}

// CatalogueField is one question on a relationship workbook page.
type CatalogueField struct {
	FieldKey   string `json:"field_key"`
	FieldLabel string `json:"field_label"`
	FieldType  string `json:"field_type"`
	HelpText   string `json:"help_text,omitempty"`
}

// CataloguePage is one relationship workbook page (Kernel 62 §7).
type CataloguePage struct {
	PageKey         string           `json:"page_key"`
	PageTitle       string           `json:"page_title"`
	PageDescription string           `json:"page_description"`
	Fields          []CatalogueField `json:"fields"`
}

// Catalogue drives both backend validation and frontend rendering -- the
// frontend must not define its own field list independently.
type Catalogue struct {
	CatalogueKey     string          `json:"catalogue_key"`
	CatalogueVersion string          `json:"catalogue_version"`
	Pages            []CataloguePage `json:"pages"`
}

// LoadCatalogue parses and validates the embedded current catalogue.
// Validation failure is startup-fatal at the caller.
func LoadCatalogue() (Catalogue, error) {
	raw, err := catalogueFS.ReadFile(CurrentCatalogueFile)
	if err != nil {
		return Catalogue{}, err
	}

	var cat Catalogue
	if err := json.Unmarshal(raw, &cat); err != nil {
		return Catalogue{}, fmt.Errorf("relationship catalogue parse error: %w", err)
	}

	if err := ValidateCatalogue(cat); err != nil {
		return Catalogue{}, err
	}
	return cat, nil
}

// FieldByKey finds a field anywhere in the catalogue. Field keys are unique
// catalogue-wide (validated below).
func (c Catalogue) FieldByKey(key string) (CatalogueField, bool) {
	for _, page := range c.Pages {
		for _, field := range page.Fields {
			if field.FieldKey == key {
				return field, true
			}
		}
	}
	return CatalogueField{}, false
}

// PageByKey finds a page by its page_key.
func (c Catalogue) PageByKey(key string) (CataloguePage, bool) {
	for _, page := range c.Pages {
		if page.PageKey == key {
			return page, true
		}
	}
	return CataloguePage{}, false
}

// ValidateCatalogue enforces the structural rules a relationship catalogue
// must satisfy. Pure and side-effect free so it runs at startup and in tests
// without a database.
func ValidateCatalogue(c Catalogue) error {
	if strings.TrimSpace(c.CatalogueKey) == "" {
		return fmt.Errorf("catalogue_key is required")
	}
	if strings.TrimSpace(c.CatalogueVersion) == "" {
		return fmt.Errorf("catalogue_version is required")
	}
	if len(c.Pages) == 0 {
		return fmt.Errorf("catalogue must define at least one page")
	}

	seenPageKeys := map[string]bool{}
	seenFieldKeys := map[string]bool{}

	for _, page := range c.Pages {
		key := strings.TrimSpace(page.PageKey)
		if key == "" {
			return fmt.Errorf("page missing page_key")
		}
		if seenPageKeys[key] {
			return fmt.Errorf("duplicate page_key %q", key)
		}
		seenPageKeys[key] = true

		if strings.TrimSpace(page.PageTitle) == "" {
			return fmt.Errorf("page %q missing page_title", key)
		}

		for _, field := range page.Fields {
			fkey := strings.TrimSpace(field.FieldKey)
			if fkey == "" {
				return fmt.Errorf("page %q has a field missing field_key", key)
			}
			if seenFieldKeys[fkey] {
				return fmt.Errorf("duplicate field_key %q", fkey)
			}
			seenFieldKeys[fkey] = true

			if strings.TrimSpace(field.FieldLabel) == "" {
				return fmt.Errorf("field %q missing field_label", fkey)
			}
			if !validFieldTypes[field.FieldType] {
				return fmt.Errorf("field %q has invalid field_type %q", fkey, field.FieldType)
			}
		}
	}

	return nil
}

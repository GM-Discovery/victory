package playerprofile

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed catalogues/player-profile-v1.0.0.json
var catalogueFS embed.FS

// CurrentCatalogueFile is the versioned catalogue this build ships. A future
// catalogue revision adds a new file and bumps this constant rather than
// mutating v1.0.0 in place (Kernel 61 §6.2).
const CurrentCatalogueFile = "catalogues/player-profile-v1.0.0.json"

// Face regions are fixed for v1 (Kernel 61 §6.8) -- a field's face_region
// must be one of these, and the projector groups strictly into these five.
const (
	RegionIdentityHeader  = "identity_header"
	RegionAtAGlance       = "at_a_glance"
	RegionPlayAndCreate   = "play_and_create"
	RegionAbout           = "about"
	RegionCreditsAndLinks = "credits_and_links"
)

var validFaceRegions = map[string]bool{
	RegionIdentityHeader:  true,
	RegionAtAGlance:       true,
	RegionPlayAndCreate:   true,
	RegionAbout:           true,
	RegionCreditsAndLinks: true,
}

// Field types the backend and frontend both understand. Adding a new type
// requires updating both the validator here and the frontend renderer --
// the catalogue cannot invent a type neither side can render.
const (
	FieldTypeText               = "text"
	FieldTypeLongText           = "long_text"
	FieldTypeURL                = "url"
	FieldTypeSingleSelect       = "single_select"
	FieldTypeSingleSelectCustom = "single_select_custom"
	FieldTypeMultiSelectCustom  = "multi_select_custom"
)

var validFieldTypes = map[string]bool{
	FieldTypeText:               true,
	FieldTypeLongText:           true,
	FieldTypeURL:                true,
	FieldTypeSingleSelect:       true,
	FieldTypeSingleSelectCustom: true,
	FieldTypeMultiSelectCustom:  true,
}

func fieldTypeAllowsOptions(t string) bool {
	return t == FieldTypeSingleSelect || t == FieldTypeSingleSelectCustom || t == FieldTypeMultiSelectCustom
}

func fieldTypeIsCollection(t string) bool {
	return t == FieldTypeMultiSelectCustom
}

// FavoriteTTRPGsFieldKey is the one field the kernel spec pins to a fixed
// maximum (Kernel 61 §3.7, §9.6) -- validated explicitly rather than only by
// the generic max_items>0 rule, so a catalogue author can't quietly raise it.
const FavoriteTTRPGsFieldKey = "favorite_ttrpgs"
const FavoriteTTRPGsMaxItems = 3

// CatalogueField is one question on a Player Workbook page (Kernel 61 §6.2).
type CatalogueField struct {
	FieldKey        string   `json:"field_key"`
	FieldLabel      string   `json:"field_label"`
	FieldType       string   `json:"field_type"`
	Required        bool     `json:"required"`
	Options         []string `json:"options,omitempty"`
	AllowCustom     bool     `json:"allow_custom,omitempty"`
	MaxItems        int      `json:"max_items,omitempty"`
	FaceEligible    bool     `json:"face_eligible"`
	FaceRegion      string   `json:"face_region,omitempty"`
	DefaultPriority int      `json:"default_priority"`
	HelpText        string   `json:"help_text,omitempty"`
}

// CataloguePage is one Player Workbook page (Kernel 61 §6.2, §7).
type CataloguePage struct {
	PageKey         string           `json:"page_key"`
	PageTitle       string           `json:"page_title"`
	PageDescription string           `json:"page_description"`
	Fields          []CatalogueField `json:"fields"`
}

// Catalogue is the full versioned page/question catalogue that drives both
// backend validation and frontend rendering (Kernel 61 §6.2) -- the frontend
// must not define its own field list independently of this document.
type Catalogue struct {
	CatalogueKey     string          `json:"catalogue_key"`
	CatalogueVersion string          `json:"catalogue_version"`
	Pages            []CataloguePage `json:"pages"`
}

// LoadCatalogue parses the embedded current catalogue file and validates it.
// Callers should treat a validation failure here as a startup-fatal error --
// an invalid catalogue must never reach a running server (Kernel 61 §9.6).
func LoadCatalogue() (Catalogue, error) {
	raw, err := catalogueFS.ReadFile(CurrentCatalogueFile)
	if err != nil {
		return Catalogue{}, err
	}

	var cat Catalogue
	if err := json.Unmarshal(raw, &cat); err != nil {
		return Catalogue{}, fmt.Errorf("catalogue parse error: %w", err)
	}

	if err := ValidateCatalogue(cat); err != nil {
		return Catalogue{}, err
	}

	return cat, nil
}

// FieldByKey finds a field anywhere in the catalogue by its field_key.
// Field keys are unique catalogue-wide (validated below), so this is
// unambiguous.
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

// ValidateCatalogue enforces the structural and semantic rules a catalogue
// must satisfy before it can drive the backend (Kernel 61 §9.6 / AC-9
// through AC-12). It is pure and side-effect free so it can run at startup
// and in tests without a database.
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
	sawFavoriteTTRPGs := false

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

			if len(field.Options) > 0 && !fieldTypeAllowsOptions(field.FieldType) {
				return fmt.Errorf("field %q has options but field_type %q does not support options", fkey, field.FieldType)
			}

			if fieldTypeIsCollection(field.FieldType) && field.MaxItems <= 0 {
				return fmt.Errorf("collection field %q must define a positive max_items", fkey)
			}
			if !fieldTypeIsCollection(field.FieldType) && field.MaxItems != 0 {
				return fmt.Errorf("non-collection field %q must not define max_items", fkey)
			}

			if field.FaceEligible {
				if !validFaceRegions[field.FaceRegion] {
					return fmt.Errorf("face-eligible field %q has invalid face_region %q", fkey, field.FaceRegion)
				}
			} else if field.FaceRegion != "" {
				return fmt.Errorf("field %q sets face_region but is not face_eligible", fkey)
			}

			if field.DefaultPriority < 0 {
				return fmt.Errorf("field %q has invalid negative default_priority", fkey)
			}

			if fkey == FavoriteTTRPGsFieldKey {
				sawFavoriteTTRPGs = true
				if field.MaxItems != FavoriteTTRPGsMaxItems {
					return fmt.Errorf("field %q must set max_items to %d, got %d", fkey, FavoriteTTRPGsMaxItems, field.MaxItems)
				}
			}
		}
	}

	if !sawFavoriteTTRPGs {
		return fmt.Errorf("catalogue must define required field %q", FavoriteTTRPGsFieldKey)
	}

	return nil
}

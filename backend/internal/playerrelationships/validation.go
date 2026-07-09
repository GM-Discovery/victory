package playerrelationships

import (
	"fmt"
	"strings"
)

const maxFieldValueLength = 20000

// ValidatePageAnswers checks submitted answers for one workbook page against
// the catalogue and returns a sanitized copy. Unknown fields and non-string
// values are rejected -- every relationship workbook field is freeform text
// (Kernel 62 §7).
func ValidatePageAnswers(page CataloguePage, answers map[string]any) (map[string]any, error) {
	byKey := map[string]CatalogueField{}
	for _, f := range page.Fields {
		byKey[f.FieldKey] = f
	}

	out := map[string]any{}
	for key, value := range answers {
		if _, ok := byKey[key]; !ok {
			return nil, fmt.Errorf("unknown_field:%s", key)
		}
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid_value_type:%s", key)
		}
		s = strings.TrimSpace(s)
		if len(s) > maxFieldValueLength {
			return nil, fmt.Errorf("value_too_long:%s", key)
		}
		out[key] = s
	}

	return out, nil
}

// ValidateCategories sanitizes a submitted category set (Kernel 62 §5.2):
// known keys only, custom requires a non-empty label, non-custom labels are
// dropped, duplicates collapse.
func ValidateCategories(input []Category) ([]Category, error) {
	seen := map[string]bool{}
	out := make([]Category, 0, len(input))
	for _, c := range input {
		key := strings.TrimSpace(c.CategoryKey)
		label := strings.TrimSpace(c.CustomLabel)
		if !ValidateCategoryKey(key) {
			return nil, fmt.Errorf("invalid_category:%s", key)
		}
		if key == CategoryCustom {
			if label == "" {
				return nil, fmt.Errorf("custom_category_label_required")
			}
		} else {
			label = ""
		}
		dedup := key + "\x00" + strings.ToLower(label)
		if seen[dedup] {
			continue
		}
		seen[dedup] = true
		out = append(out, Category{CategoryKey: key, CustomLabel: label})
	}
	return out, nil
}

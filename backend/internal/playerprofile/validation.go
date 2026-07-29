package playerprofile

import (
	"fmt"
	"strings"
)

// ValidatePageAnswers checks submitted answers for one page against the
// catalogue's field contracts and returns a sanitized copy. It rejects
// unknown fields, wrong types, and violations of the fixed contracts
// (favorite TTRPGs max three, collection max_items, options-only choices
// without allow_custom) -- Kernel 61 §6.2, §9.6, AC-10/AC-11/AC-12.
func ValidatePageAnswers(page CataloguePage, answers map[string]any) (map[string]any, error) {
	byKey := map[string]CatalogueField{}
	for _, f := range page.Fields {
		byKey[f.FieldKey] = f
	}

	out := map[string]any{}
	for key, value := range answers {
		field, ok := byKey[key]
		if !ok {
			return nil, fmt.Errorf("unknown_field:%s", key)
		}

		sanitized, err := validateFieldValue(field, value)
		if err != nil {
			return nil, err
		}
		out[key] = sanitized
	}

	for _, field := range page.Fields {
		if !field.Required {
			continue
		}
		if _, present := out[field.FieldKey]; !present {
			return nil, fmt.Errorf("required_field_missing:%s", field.FieldKey)
		}
	}

	return out, nil
}

func validateFieldValue(field CatalogueField, value any) (any, error) {
	switch field.FieldType {
	case FieldTypeText, FieldTypeLongText, FieldTypeURL:
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid_value_type:%s", field.FieldKey)
		}
		return strings.TrimSpace(s), nil

	case FieldTypeSingleSelect, FieldTypeSingleSelectCustom:
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("invalid_value_type:%s", field.FieldKey)
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return "", nil
		}
		if !field.AllowCustom && !containsOption(field.Options, s) {
			return nil, fmt.Errorf("invalid_option:%s", field.FieldKey)
		}
		return s, nil

	case FieldTypeMultiSelectCustom:
		items, err := toStringSlice(value)
		if err != nil {
			return nil, fmt.Errorf("invalid_value_type:%s", field.FieldKey)
		}
		cleaned := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if !field.AllowCustom && !containsOption(field.Options, item) {
				return nil, fmt.Errorf("invalid_option:%s", field.FieldKey)
			}
			cleaned = append(cleaned, item)
		}
		if field.MaxItems > 0 && len(cleaned) > field.MaxItems {
			return nil, fmt.Errorf("too_many_items:%s", field.FieldKey)
		}
		return cleaned, nil

	case FieldTypeTTRPGMatrix:
		input, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid_value_type:%s", field.FieldKey)
		}
		out := map[string]string{}
		allowed := map[string]bool{}
		for _, status := range field.Options {
			allowed[strings.ToLower(status)] = true
		}
		for key, raw := range input {
			status, ok := raw.(string)
			if !ok {
				return nil, fmt.Errorf("invalid_value_type:%s", field.FieldKey)
			}
			status = strings.TrimSpace(status)
			if status != "" && !allowed[strings.ToLower(status)] {
				return nil, fmt.Errorf("invalid_option:%s", field.FieldKey)
			}
			out[strings.TrimSpace(key)] = status
		}
		return out, nil

	default:
		return nil, fmt.Errorf("invalid_field_type:%s", field.FieldKey)
	}
}

func containsOption(options []string, want string) bool {
	for _, o := range options {
		if strings.EqualFold(o, want) {
			return true
		}
	}
	return false
}

func toStringSlice(value any) ([]string, error) {
	switch v := value.(type) {
	case []string:
		return v, nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("non_string_item")
			}
			out = append(out, s)
		}
		return out, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("not_a_list")
	}
}

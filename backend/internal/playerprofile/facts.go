package playerprofile

import (
	"fmt"
	"sort"
	"strings"
)

// DeriveEffectiveFacts folds a list of remaining (non-deleted) profile
// events, oldest first, into the current effective fact per field_key. Later
// events overwrite earlier ones for the same key. Because the caller simply
// omits a deleted event from the input, deleting the newest event for a
// field naturally reveals whatever the previous remaining event set
// (Kernel 61 §6.4, AC-15/AC-16/AC-17) -- there is no separate "undo" path.
//
// events must already be sorted oldest-to-newest by CreatedAt; callers own
// that ordering since it usually comes straight from an ORDER BY query.
func DeriveEffectiveFacts(cat Catalogue, events []ProfileEvent) map[string]ProfileFact {
	facts := map[string]ProfileFact{}

	for _, event := range events {
		for fieldKey, rawValue := range event.Payload {
			field, ok := cat.FieldByKey(fieldKey)
			if !ok {
				// A field the current catalogue no longer defines (e.g. a
				// retired legacy import key) is preserved as a fact but is
				// never Face-eligible -- see resolveProjectedField.
				facts[fieldKey] = ProfileFact{
					FieldKey:         fieldKey,
					ValueJSON:        rawValue,
					DisplayValue:     formatDisplayValue(CatalogueField{}, rawValue),
					SourceEventID:    event.ID,
					SourcePageKey:    event.PageKey,
					CatalogueVersion: event.CatalogueVersion,
					EffectiveAt:      event.CreatedAt,
				}
				continue
			}

			facts[fieldKey] = ProfileFact{
				FieldKey:         fieldKey,
				ValueJSON:        rawValue,
				DisplayValue:     formatDisplayValue(field, rawValue),
				SourceEventID:    event.ID,
				SourcePageKey:    event.PageKey,
				CatalogueVersion: event.CatalogueVersion,
				EffectiveAt:      event.CreatedAt,
			}
		}
	}

	return facts
}

func formatDisplayValue(field CatalogueField, value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, fmt.Sprint(item))
		}
		return strings.Join(parts, ", ")
	case []string:
		return strings.Join(v, ", ")
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

// PageAnswersDifferFromFacts reports whether the submitted answers for a
// page contain any material change versus the current effective facts. A
// commit with no material change must not create a duplicate event
// (Kernel 61 §6.3, AC-14).
func PageAnswersDifferFromFacts(answers map[string]any, currentFacts map[string]ProfileFact) bool {
	for key, newValue := range answers {
		existing, ok := currentFacts[key]
		if !ok {
			if !isEmptyValue(newValue) {
				return true
			}
			continue
		}
		if !valuesEqual(existing.ValueJSON, newValue) {
			return true
		}
	}
	return false
}

func isEmptyValue(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(t) == ""
	case []any:
		return len(t) == 0
	case []string:
		return len(t) == 0
	default:
		return false
	}
}

func valuesEqual(a, b any) bool {
	return formatDisplayValue(CatalogueField{}, a) == formatDisplayValue(CatalogueField{}, b)
}

// BuildPageCommitSummary generates a server-authored human-readable summary
// from catalogue field labels and validated values -- client-supplied
// summaries are never trusted (Kernel 61 §6.3).
func BuildPageCommitSummary(page CataloguePage, answers map[string]any) string {
	labels := make([]string, 0, len(answers))
	byKey := map[string]CatalogueField{}
	for _, field := range page.Fields {
		byKey[field.FieldKey] = field
	}
	for key := range answers {
		if field, ok := byKey[key]; ok {
			labels = append(labels, field.FieldLabel)
		} else {
			labels = append(labels, key)
		}
	}
	sort.Strings(labels)
	if len(labels) == 0 {
		return fmt.Sprintf("Updated %s", page.PageTitle)
	}
	return fmt.Sprintf("Updated %s: %s", page.PageTitle, strings.Join(labels, ", "))
}

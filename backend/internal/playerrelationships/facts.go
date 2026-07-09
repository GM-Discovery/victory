package playerrelationships

import (
	"fmt"
	"sort"
	"strings"
)

// DeriveEffectiveFacts folds remaining (non-deleted) relationship events,
// oldest first, into the current effective fact per field_key. Later events
// win per key; deleting the newest event for a field naturally reveals the
// prior remaining value -- the same deterministic fold as the Kernel 61
// player-profile facts engine (Kernel 62 §5.3, §5.4).
//
// events must already be sorted oldest-to-newest by CreatedAt.
func DeriveEffectiveFacts(events []RelationshipEvent) map[string]RelationshipFact {
	facts := map[string]RelationshipFact{}

	for _, event := range events {
		for fieldKey, rawValue := range event.Payload {
			facts[fieldKey] = RelationshipFact{
				FieldKey:      fieldKey,
				ValueJSON:     rawValue,
				DisplayValue:  formatDisplayValue(rawValue),
				SourceEventID: event.ID,
				UpdatedAt:     event.CreatedAt,
			}
		}
	}

	return facts
}

func formatDisplayValue(value any) string {
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

// PageAnswersDifferFromFacts reports whether submitted answers contain any
// material change versus current facts. A no-op commit must not create a
// duplicate event.
func PageAnswersDifferFromFacts(answers map[string]any, currentFacts map[string]RelationshipFact) bool {
	for key, newValue := range answers {
		existing, ok := currentFacts[key]
		if !ok {
			if !isEmptyValue(newValue) {
				return true
			}
			continue
		}
		if formatDisplayValue(existing.ValueJSON) != formatDisplayValue(newValue) {
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

// BuildPageCommitSummary generates a server-authored human-readable summary
// from catalogue field labels -- client-supplied summaries are never trusted.
func BuildPageCommitSummary(page CataloguePage, answers map[string]any) string {
	byKey := map[string]CatalogueField{}
	for _, field := range page.Fields {
		byKey[field.FieldKey] = field
	}
	labels := make([]string, 0, len(answers))
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

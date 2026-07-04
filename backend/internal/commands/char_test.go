package commands

import (
	"context"
	"testing"
)

func TestCharSettableFieldsOnlyAllowsNarrativeSingletons(t *testing.T) {
	want := map[string]bool{"name": true, "pronouns": true, "aura": true}
	if len(charSettableFields) != len(want) {
		t.Fatalf("expected exactly %d settable fields, got %d", len(want), len(charSettableFields))
	}
	for field := range want {
		if _, ok := charSettableFields[field]; !ok {
			t.Fatalf("expected %q to be settable", field)
		}
	}
}

func TestExecuteCharSetRejectsProtectedFacts(t *testing.T) {
	for _, field := range []string{"archetype", "attributes", "skill_dice", "ruleset", "onboarding_status"} {
		t.Run(field, func(t *testing.T) {
			// nil pool is safe here: the protected-field check happens before
			// any database access in ExecuteCharSet.
			_, err := ExecuteCharSet(context.Background(), nil, "user-1", "card-1", field, "value")
			if err == nil || err.Error() != "protected_fact" {
				t.Fatalf("ExecuteCharSet(%q) err = %v, want protected_fact", field, err)
			}
		})
	}
}

func TestExecuteCharNavigateDefaultsToFace(t *testing.T) {
	page, err := ExecuteCharNavigate("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page != "face" {
		t.Fatalf("expected default page 'face', got %q", page)
	}
}

func TestExecuteCharNavigateRejectsUnknownPage(t *testing.T) {
	if _, err := ExecuteCharNavigate("mechanics-and-more"); err == nil {
		t.Fatalf("expected unknown page to be rejected")
	}
}

func TestExecuteCharNavigateAcceptsKnownPages(t *testing.T) {
	for _, page := range []string{"face", "mechanics", "history", "journal"} {
		got, err := ExecuteCharNavigate(page)
		if err != nil {
			t.Fatalf("ExecuteCharNavigate(%q) unexpected error: %v", page, err)
		}
		if got != page {
			t.Fatalf("ExecuteCharNavigate(%q) = %q, want %q", page, got, page)
		}
	}
}

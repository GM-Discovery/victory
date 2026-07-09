package playerrelationships

import "testing"

func TestQualitativeVocabulariesAcceptSpecKeys(t *testing.T) {
	// Every key fixed by Kernel 62 §6 must validate.
	for _, key := range []string{"unknown", "cautious", "developing", "trusted", "deeply_trusted"} {
		if !ValidateTrustKey(key) {
			t.Fatalf("trust key %q rejected", key)
		}
	}
	for _, key := range []string{"unknown", "distant", "familiar", "friendly", "close", "core_relationship"} {
		if !ValidateClosenessKey(key) {
			t.Fatalf("closeness key %q rejected", key)
		}
	}
	for _, key := range []string{"unknown", "inconsistent", "usually_reliable", "reliable", "highly_reliable"} {
		if !ValidateReliabilityKey(key) {
			t.Fatalf("reliability key %q rejected", key)
		}
	}
	for _, key := range []string{"unknown", "difficult", "uneven", "workable", "easy", "very_easy"} {
		if !ValidateCommunicationEaseKey(key) {
			t.Fatalf("communication ease key %q rejected", key)
		}
	}
}

func TestQualitativeVocabulariesRejectInvalidKeys(t *testing.T) {
	for _, key := range []string{"", "5", "best_friend", "TRUSTED", "trusted "} {
		if ValidateTrustKey(key) {
			t.Fatalf("trust key %q should be rejected", key)
		}
	}
	if ValidateClosenessKey("soulmate") {
		t.Fatal("invalid closeness key accepted")
	}
	if ValidateReliabilityKey("flaky") {
		t.Fatal("invalid reliability key accepted")
	}
	if ValidateCommunicationEaseKey("telepathic") {
		t.Fatal("invalid communication ease key accepted")
	}
}

func TestArchivedStateNotDirectlySettable(t *testing.T) {
	// Archive is owned by the archive/unarchive operations (Kernel 62 §14);
	// the state dropdown may not enter it directly.
	if ValidateSettableStateKey(StateArchived) {
		t.Fatal("archived must not be directly settable")
	}
	for _, key := range []string{"active", "quiet", "strained", "rebuilding"} {
		if !ValidateSettableStateKey(key) {
			t.Fatalf("state key %q rejected", key)
		}
	}
	if ValidateSettableStateKey("complicated") {
		t.Fatal("invalid state key accepted")
	}
}

func TestCategoryKeysMatchSpec(t *testing.T) {
	expected := []string{
		"acquaintance", "friend", "close_friend", "family", "collaborator",
		"coworker", "player", "gm_facilitator", "mentor", "mentee", "client",
		"community_contact", "creative_partner", "professional_contact", "custom",
	}
	if len(CategoryKeys) != len(expected) {
		t.Fatalf("expected %d category keys, got %d", len(expected), len(CategoryKeys))
	}
	for _, key := range expected {
		if !ValidateCategoryKey(key) {
			t.Fatalf("category key %q rejected", key)
		}
		if CategoryLabels[key] == "" {
			t.Fatalf("category key %q missing label", key)
		}
	}
	if ValidateCategoryKey("nemesis") {
		t.Fatal("invalid category key accepted")
	}
}

func TestFollowUpStatuses(t *testing.T) {
	for _, status := range []string{"open", "done", "dismissed"} {
		if !ValidateFollowUpStatus(status) {
			t.Fatalf("status %q rejected", status)
		}
	}
	if ValidateFollowUpStatus("snoozed") {
		t.Fatal("invalid follow-up status accepted")
	}
}

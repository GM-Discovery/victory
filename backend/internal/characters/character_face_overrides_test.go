package characters

import "testing"

func TestSetCharacterFactFaceVisibilityRequiresAuth(t *testing.T) {
	_, err := SetCharacterFactFaceVisibility(nil, nil, "", "card-1", "tagline", "shown")
	if err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("err = %v, want not_authenticated", err)
	}
}

func TestSetCharacterFactFaceVisibilityRequiresCardID(t *testing.T) {
	_, err := SetCharacterFactFaceVisibility(nil, nil, "user-1", "", "tagline", "shown")
	if err == nil || err.Error() != "character_card_id_required" {
		t.Fatalf("err = %v, want character_card_id_required", err)
	}
}

func TestSetCharacterFactFaceVisibilityRequiresFactKey(t *testing.T) {
	_, err := SetCharacterFactFaceVisibility(nil, nil, "user-1", "card-1", "  ", "shown")
	if err == nil || err.Error() != "fact_key_required" {
		t.Fatalf("err = %v, want fact_key_required", err)
	}
}

func TestSetCharacterFactFaceVisibilityRejectsInvalidMode(t *testing.T) {
	_, err := SetCharacterFactFaceVisibility(nil, nil, "user-1", "card-1", "tagline", "sometimes")
	if err == nil || err.Error() != "invalid_visibility_mode" {
		t.Fatalf("err = %v, want invalid_visibility_mode", err)
	}
}

func TestSetCharacterFactFaceVisibilityRejectsInferredAsMode(t *testing.T) {
	// "inferred" is a reset, not a settable mode -- callers must use
	// ReturnCharacterFactFaceVisibilityToInferred instead.
	_, err := SetCharacterFactFaceVisibility(nil, nil, "user-1", "card-1", "tagline", "inferred")
	if err == nil || err.Error() != "invalid_visibility_mode" {
		t.Fatalf("err = %v, want invalid_visibility_mode", err)
	}
}

func TestReturnCharacterFactFaceVisibilityToInferredRequiresAuth(t *testing.T) {
	_, err := ReturnCharacterFactFaceVisibilityToInferred(nil, nil, "", "card-1", "tagline")
	if err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("err = %v, want not_authenticated", err)
	}
}

func TestReturnCharacterFactFaceVisibilityToInferredRequiresCardID(t *testing.T) {
	_, err := ReturnCharacterFactFaceVisibilityToInferred(nil, nil, "user-1", "", "tagline")
	if err == nil || err.Error() != "character_card_id_required" {
		t.Fatalf("err = %v, want character_card_id_required", err)
	}
}

func TestReturnCharacterFactFaceVisibilityToInferredRequiresFactKey(t *testing.T) {
	_, err := ReturnCharacterFactFaceVisibilityToInferred(nil, nil, "user-1", "card-1", "")
	if err == nil || err.Error() != "fact_key_required" {
		t.Fatalf("err = %v, want fact_key_required", err)
	}
}

func TestSetDirectorCharacterFactLockRequiresAuth(t *testing.T) {
	_, err := SetDirectorCharacterFactLock(nil, nil, "", "card-1", "tagline", "visibility", true, "Because")
	if err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("err = %v, want not_authenticated", err)
	}
}

func TestSetDirectorCharacterFactLockRequiresFactKey(t *testing.T) {
	_, err := SetDirectorCharacterFactLock(nil, nil, "user-1", "card-1", "", "visibility", true, "Because")
	if err == nil || err.Error() != "fact_key_required" {
		t.Fatalf("err = %v, want fact_key_required", err)
	}
}

func TestSetDirectorCharacterFactLockRequiresReason(t *testing.T) {
	_, err := SetDirectorCharacterFactLock(nil, nil, "user-1", "card-1", "tagline", "visibility", true, " ")
	if err == nil || err.Error() != "director_reason_required" {
		t.Fatalf("err = %v, want director_reason_required", err)
	}
}

func TestSetDirectorCharacterFactLockRejectsInvalidDimension(t *testing.T) {
	_, err := SetDirectorCharacterFactLock(nil, nil, "user-1", "card-1", "tagline", "layout", true, "Because")
	if err == nil || err.Error() != "invalid_lock_dimension" {
		t.Fatalf("err = %v, want invalid_lock_dimension", err)
	}
}

func TestSetDirectorCharacterFactValueOverrideRequiresValue(t *testing.T) {
	_, err := SetDirectorCharacterFactValueOverride(nil, nil, "user-1", "card-1", "tagline", " ", "Because")
	if err == nil || err.Error() != "override_value_required" {
		t.Fatalf("err = %v, want override_value_required", err)
	}
}

func TestSetDirectorCharacterFactValueOverrideRequiresReason(t *testing.T) {
	_, err := SetDirectorCharacterFactValueOverride(nil, nil, "user-1", "card-1", "tagline", "Override", " ")
	if err == nil || err.Error() != "director_reason_required" {
		t.Fatalf("err = %v, want director_reason_required", err)
	}
}

func TestClearDirectorCharacterFactValueOverrideRequiresReason(t *testing.T) {
	_, err := ClearDirectorCharacterFactValueOverride(nil, nil, "user-1", "card-1", "tagline", " ")
	if err == nil || err.Error() != "director_reason_required" {
		t.Fatalf("err = %v, want director_reason_required", err)
	}
}

func TestSetCharacterFactPriorityRequiresAuth(t *testing.T) {
	_, err := SetCharacterFactPriority(nil, nil, "", "card-1", "tagline", 50)
	if err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("err = %v, want not_authenticated", err)
	}
}

func TestSetCharacterFactPriorityRequiresCardID(t *testing.T) {
	_, err := SetCharacterFactPriority(nil, nil, "user-1", "", "tagline", 50)
	if err == nil || err.Error() != "character_card_id_required" {
		t.Fatalf("err = %v, want character_card_id_required", err)
	}
}

func TestSetCharacterFactPriorityRequiresFactKey(t *testing.T) {
	_, err := SetCharacterFactPriority(nil, nil, "user-1", "card-1", "", 50)
	if err == nil || err.Error() != "fact_key_required" {
		t.Fatalf("err = %v, want fact_key_required", err)
	}
}

func TestReturnCharacterFactPriorityToInferredRequiresAuth(t *testing.T) {
	_, err := ReturnCharacterFactPriorityToInferred(nil, nil, "", "card-1", "tagline")
	if err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("err = %v, want not_authenticated", err)
	}
}

func TestReturnCharacterFactPriorityToInferredRequiresCardID(t *testing.T) {
	_, err := ReturnCharacterFactPriorityToInferred(nil, nil, "user-1", "", "tagline")
	if err == nil || err.Error() != "character_card_id_required" {
		t.Fatalf("err = %v, want character_card_id_required", err)
	}
}

func TestReturnCharacterFactPriorityToInferredRequiresFactKey(t *testing.T) {
	_, err := ReturnCharacterFactPriorityToInferred(nil, nil, "user-1", "card-1", "")
	if err == nil || err.Error() != "fact_key_required" {
		t.Fatalf("err = %v, want fact_key_required", err)
	}
}

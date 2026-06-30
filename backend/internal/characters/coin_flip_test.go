package characters

import "testing"

func TestRequestCatharsisCoinFlipRequiresAuth(t *testing.T) {
	_, err := RequestCatharsisCoinFlip(nil, nil, "", "card-1", 1)
	if err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("err = %v, want not_authenticated", err)
	}
}

func TestRequestCatharsisCoinFlipRequiresCardID(t *testing.T) {
	_, err := RequestCatharsisCoinFlip(nil, nil, "user-1", "", 1)
	if err == nil || err.Error() != "character_card_id_required" {
		t.Fatalf("err = %v, want character_card_id_required", err)
	}
}

func TestRequestCatharsisCoinFlipRequiresValidParentIndex(t *testing.T) {
	_, err := RequestCatharsisCoinFlip(nil, nil, "user-1", "card-1", 0)
	if err == nil || err.Error() != "invalid_parent_index" {
		t.Fatalf("err = %v, want invalid_parent_index", err)
	}
}

func TestCoinFlipEventKeyIsStableAndUnique(t *testing.T) {
	if coinFlipEventKey(1) == coinFlipEventKey(2) {
		t.Fatalf("expected distinct event keys per parent index")
	}
	if coinFlipEventKey(1) != coinFlipEventKey(1) {
		t.Fatalf("expected stable event key for the same parent index")
	}
}

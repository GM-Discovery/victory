package playerprofile

import "testing"

func TestValidateStageNameCandidateRejectsBlank(t *testing.T) {
	if _, err := ValidateStageNameCandidate("   "); err == nil {
		t.Fatalf("expected whitespace-only stage name to be rejected")
	}
	if _, err := ValidateStageNameCandidate(""); err == nil {
		t.Fatalf("expected empty stage name to be rejected")
	}
}

func TestValidateStageNameCandidateTrims(t *testing.T) {
	got, err := ValidateStageNameCandidate("  Straturli  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Straturli" {
		t.Fatalf("expected trimmed name, got %q", got)
	}
}

func TestValidateStageNameCandidateRejectsTooLong(t *testing.T) {
	long := ""
	for i := 0; i < 200; i++ {
		long += "a"
	}
	if _, err := ValidateStageNameCandidate(long); err == nil {
		t.Fatalf("expected overly long stage name to be rejected")
	}
}

func TestNormalizeStageNameCaseAndWhitespaceInsensitive(t *testing.T) {
	a := NormalizeStageName("  Straturli  ")
	b := NormalizeStageName("STRATURLI")
	if a != b {
		t.Fatalf("expected normalized names to match, got %q vs %q", a, b)
	}
	if NormalizeStageName("Grant   Murray") != "grant murray" {
		t.Fatalf("expected internal whitespace to collapse, got %q", NormalizeStageName("Grant   Murray"))
	}
}

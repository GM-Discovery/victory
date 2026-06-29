package characters

import "testing"

func TestRomanNumeralRoundTrip(t *testing.T) {
	cases := map[int]string{1: "I", 2: "II", 4: "IV", 9: "IX", 12: "XII", 50: "L", 51: "LI"}
	for n, want := range cases {
		if got := romanNumeral(n); got != want {
			t.Fatalf("romanNumeral(%d) = %q, want %q", n, got, want)
		}
		if got, ok := parseRomanNumeral(want); !ok || got != n {
			t.Fatalf("parseRomanNumeral(%q) = (%d, %v), want (%d, true)", want, got, ok, n)
		}
	}
}

func TestIsGeneratedRomanNameRejectsNonNumerals(t *testing.T) {
	for _, name := range []string{"", "Aurora", "I am not roman", "IIII-ish"} {
		if isGeneratedRomanName(name) {
			t.Fatalf("isGeneratedRomanName(%q) = true, want false", name)
		}
	}
	for _, name := range []string{"I", "IV", "XII", "L"} {
		if !isGeneratedRomanName(name) {
			t.Fatalf("isGeneratedRomanName(%q) = false, want true", name)
		}
	}
}

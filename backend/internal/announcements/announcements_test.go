package announcements

import (
	"strings"
	"testing"
)

func TestPaletteIsBoundedAndWellFormed(t *testing.T) {
	if len(Palette) < 8 {
		t.Fatalf("palette should offer a rich preset set, got %d", len(Palette))
	}
	seen := map[string]bool{}
	for _, s := range Palette {
		if seen[s.Key] {
			t.Fatalf("duplicate style key %q", s.Key)
		}
		seen[s.Key] = true
		if strings.TrimSpace(s.Label) == "" {
			t.Fatalf("style %q has no label", s.Key)
		}
		if strings.TrimSpace(s.Glyph) == "" {
			t.Fatalf("style %q has no glyph", s.Key)
		}
		if strings.TrimSpace(s.Motion) == "" {
			t.Fatalf("style %q has no motion", s.Key)
		}
		if strings.TrimSpace(s.Shape) == "" {
			t.Fatalf("style %q has no shape", s.Key)
		}
	}
	if !seen["custom"] {
		t.Fatal("palette must include the open-text custom style (kernel 89 §10.3)")
	}
}

// Kernel 89 §10.2: distinctions must not rely on color alone. This asserts
// the property structurally -- if two styles ever collapse onto the same
// glyph AND the same motion AND the same shape, they are distinguishable
// only by their accent, which is exactly what the spec forbids.
func TestStylesAreDistinguishableWithoutColor(t *testing.T) {
	seen := map[string]string{}
	for _, s := range Palette {
		fingerprint := s.Glyph + "|" + s.Motion + "|" + s.Shape + "|" + s.Emphasis
		if other, ok := seen[fingerprint]; ok {
			t.Fatalf("styles %q and %q are distinguished by color alone (%s)", other, s.Key, fingerprint)
		}
		seen[fingerprint] = s.Key
	}
}

func TestPresetFallsBackToItsOwnDefaultText(t *testing.T) {
	style, text, err := Compose("explosion", "")
	if err != nil {
		t.Fatalf("preset with no text should compose: %v", err)
	}
	if style.Key != "explosion" {
		t.Fatalf("wrong style: %q", style.Key)
	}
	if text != "EXPLOSION!" {
		t.Fatalf("expected the preset's own default text, got %q", text)
	}
}

func TestCustomStyleRequiresWords(t *testing.T) {
	if _, _, err := Compose("custom", "   "); err == nil {
		t.Fatal("an empty custom announcement should be refused, not projected blank")
	}
}

func TestUnknownStyleIsRefusedNotSilentlyDefaulted(t *testing.T) {
	if _, _, err := Compose("kaboom", "hi"); err == nil {
		t.Fatal("unknown style must error rather than fall back to a plain caption")
	}
	if _, _, err := Compose("", "hi"); err == nil {
		t.Fatal("missing style must error")
	}
}

func TestStyleLookupIsCaseInsensitive(t *testing.T) {
	if _, err := Lookup("  SUCCESS "); err != nil {
		t.Fatalf("lookup should trim and lowercase: %v", err)
	}
}

func TestTextIsBoundedAndSingleLine(t *testing.T) {
	if _, _, err := Compose("custom", strings.Repeat("a", MaxTextLength+1)); err == nil {
		t.Fatal("over-long announcement should be refused")
	}
	_, text, err := Compose("custom", "the wall\n\ncracks   open")
	if err != nil {
		t.Fatalf("newlines should collapse, not fail: %v", err)
	}
	if strings.ContainsAny(text, "\n\r") || strings.Contains(text, "  ") {
		t.Fatalf("expected a single collapsed line, got %q", text)
	}
}

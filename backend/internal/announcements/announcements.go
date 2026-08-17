// Package announcements is Kernel 89's bounded, theatrical Director
// announcement palette.
//
// It is a pure leaf package: no database, no authority, no delivery. It
// answers exactly one question -- "is this a real announcement style, and
// what does it look like?" -- so that the palette has ONE definition
// (kernel 89 §10.1's "bounded and polished") rather than a Go validator
// that drifts from a hand-maintained CSS table in the browser. The
// frontend renders from the styles this package serves; it does not carry
// its own copy.
//
// What this package deliberately does not do (kernel 89 §10.4): infer
// anything. Nothing here reads a die, a total, or a target complexity.
// The Director chooses the style and the words; Victory only renders them.
// There is no mapping anywhere in this package from a roll result to a
// style, and adding one would be a product decision that contradicts the
// kernel, not a refactor.
package announcements

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// MaxTextLength bounds a custom announcement. An announcement is a stage
// caption, not a monologue -- the projection renders it large, over live
// play, and a long one would cover the stage it is meant to punctuate.
const MaxTextLength = 160

// Motion is the entrance treatment. Kept as a small closed vocabulary
// rather than free CSS so a Director cannot author something that hides
// the stage or strobes.
const (
	MotionSlam    = "slam"    // hard scale-in, settles fast
	MotionRise    = "rise"    // drifts upward into place
	MotionShake   = "shake"   // enters then jitters briefly
	MotionUnveil  = "unveil"  // fades in wide, unhurried
	MotionFlicker = "flicker" // two quick alpha beats, then steady
)

// Style is one preset in the palette.
//
// Accent/Background/Ink are presentation hints, not the whole distinction:
// Glyph, Motion, and Shape carry the same information non-chromatically, so
// the palette satisfies kernel 89 §10.2's "do not rely on color alone"
// structurally rather than by convention. A viewer who cannot distinguish
// the accent still gets a different symbol, a different entrance, and a
// different frame.
type Style struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	DefaultText string `json:"default_text"`
	Accent     string `json:"accent"`
	Background string `json:"background"`
	Ink        string `json:"ink"`
	Glyph      string `json:"glyph"`
	Motion     string `json:"motion"`
	Shape      string `json:"shape"`
	// Emphasis drives typography weight/tracking on the client. Three
	// steps only: "loud", "firm", "soft".
	Emphasis string `json:"emphasis"`
}

// Palette is the whole bounded set (kernel 89 §10.1). "custom" is included
// deliberately: §10.3's open-text option is a style like any other rather
// than a separate unstyled code path, so a custom announcement still
// arrives on stage looking like it belongs there.
var Palette = []Style{
	{
		Key: "success", Label: "Success", DefaultText: "SUCCESS!!!",
		Accent: "#5ad18c", Background: "#0f2a1d", Ink: "#eafff2",
		Glyph: "✦", Motion: MotionSlam, Shape: "burst", Emphasis: "loud",
	},
	{
		Key: "triumph", Label: "Triumph", DefaultText: "TRIUMPH",
		Accent: "#e8c46a", Background: "#2a2110", Ink: "#fff6e0",
		Glyph: "❖", Motion: MotionRise, Shape: "crown", Emphasis: "loud",
	},
	{
		Key: "explosion", Label: "Explosion", DefaultText: "EXPLOSION!",
		Accent: "#ff8a3d", Background: "#2e1408", Ink: "#fff1e4",
		Glyph: "✸", Motion: MotionShake, Shape: "burst", Emphasis: "loud",
	},
	{
		Key: "consequences", Label: "Consequences", DefaultText: "OH NO! CONSEQUENCES!",
		Accent: "#e3585f", Background: "#2c0f13", Ink: "#ffe8ea",
		Glyph: "▲", Motion: MotionShake, Shape: "jagged", Emphasis: "loud",
	},
	{
		Key: "failure", Label: "Failure", DefaultText: "FAILURE",
		Accent: "#9aa3b2", Background: "#171a20", Ink: "#eef1f6",
		Glyph: "✕", Motion: MotionFlicker, Shape: "slab", Emphasis: "firm",
	},
	{
		Key: "danger", Label: "Danger", DefaultText: "DANGER",
		Accent: "#d94a4a", Background: "#2a0e0e", Ink: "#ffeaea",
		Glyph: "⚠", Motion: MotionFlicker, Shape: "jagged", Emphasis: "firm",
	},
	{
		Key: "warning", Label: "Warning", DefaultText: "WARNING",
		Accent: "#d8a63c", Background: "#291f0a", Ink: "#fff5df",
		Glyph: "!", Motion: MotionFlicker, Shape: "slab", Emphasis: "firm",
	},
	{
		Key: "revelation", Label: "Revelation", DefaultText: "REVELATION",
		Accent: "#9d7bf0", Background: "#1c1430", Ink: "#f2ecff",
		Glyph: "◈", Motion: MotionUnveil, Shape: "halo", Emphasis: "firm",
	},
	{
		Key: "discovery", Label: "Discovery", DefaultText: "DISCOVERY",
		Accent: "#4fb8d8", Background: "#0c2029", Ink: "#e6f8ff",
		Glyph: "◇", Motion: MotionUnveil, Shape: "halo", Emphasis: "soft",
	},
	{
		Key: "custom", Label: "Custom", DefaultText: "",
		Accent: "#c8ccd6", Background: "#15181e", Ink: "#f2f4f8",
		Glyph: "▪", Motion: MotionRise, Shape: "slab", Emphasis: "soft",
	},
}

// Lookup returns the Style for key. The lookup is exact and
// case-insensitive; an unknown key is an error, never a silent fallback to
// a default style -- a Director who asked for "explosion" and got a plain
// caption would have no way to tell the palette rejected them.
func Lookup(key string) (Style, error) {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return Style{}, errors.New("announcement_style_required")
	}
	for _, s := range Palette {
		if s.Key == key {
			return s, nil
		}
	}
	return Style{}, errors.New("unknown_announcement_style")
}

// Compose validates a Director's announcement request and returns the
// resolved style plus the exact text to project.
//
// Empty text falls back to the style's own DefaultText, which is what makes
// the preset palette one click: choosing "Explosion" and pressing send is a
// complete action. The "custom" style has no DefaultText, so it requires
// words -- an empty custom announcement is a mistake, not a blank banner.
func Compose(styleKey, text string) (Style, string, error) {
	style, err := Lookup(styleKey)
	if err != nil {
		return Style{}, "", err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		text = style.DefaultText
	}
	if text == "" {
		return Style{}, "", errors.New("announcement_text_required")
	}
	if utf8.RuneCountInString(text) > MaxTextLength {
		return Style{}, "", errors.New("announcement_text_too_long")
	}
	// Newlines would let one announcement grow to any height regardless of
	// the rune cap, so they collapse to spaces rather than being rejected --
	// a pasted line break should not fail a Director's send mid-scene.
	text = strings.Join(strings.Fields(text), " ")
	return style, text, nil
}

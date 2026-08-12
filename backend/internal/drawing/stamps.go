package drawing

// ValidStamps is the small production-provided stamp palette (kernel §1
// "Include a small production-provided stamp palette. Do not build a
// generalized asset marketplace/library."). Each key is a symbol
// reference the frontend renders as a small vector glyph -- a stamp
// remains a reusable symbol reference, not destructive pixels or an
// uploaded asset, so this list is a fixed Go constant, not a table.
var ValidStamps = map[string]bool{
	"settlement": true,
	"ruin":       true,
	"mountain":   true,
	"forest":     true,
	"river-mark": true,
	"camp":       true,
	"danger":     true,
	"treasure":   true,
	"waypoint":   true,
	"skull":      true,
}

// StampLabels is a display-name lookup for the palette, used by the HTTP
// listing endpoint so the frontend doesn't have to hardcode labels.
var StampLabels = map[string]string{
	"settlement": "Settlement",
	"ruin":       "Ruin",
	"mountain":   "Mountain",
	"forest":     "Forest",
	"river-mark": "River Mark",
	"camp":       "Camp",
	"danger":     "Danger",
	"treasure":   "Treasure",
	"waypoint":   "Waypoint",
	"skull":      "Skull",
}

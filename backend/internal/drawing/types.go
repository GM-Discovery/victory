// Package drawing implements Kernel 87's canonical collaborative drawing
// objects: server-authoritative create/edit/delete/z-order/lock over a
// persistent, map-relative (not browser-pixel) geometry model, plus the
// small per-Show drawing-authority-mode and measured-tabletop settings
// that gate and configure it.
//
// Scope resolution (Cohort vs Show) deliberately reuses
// backend/internal/rollaudience rather than re-deriving cohort membership
// -- the same package Kernel 86 built for exactly this Show/Cohort
// audience question, already leaf-package-safe (pgx only) so it can be
// imported here without an import cycle.
package drawing

import "time"

const (
	TypeFreehand  = "freehand"
	TypeLine      = "line"
	TypePolyline  = "polyline"
	TypeRectangle = "rectangle"
	TypeEllipse   = "ellipse"
	TypePolygon   = "polygon"
	TypeText      = "text"
	TypeStamp     = "stamp"
)

var ValidObjectTypes = map[string]bool{
	TypeFreehand:  true,
	TypeLine:      true,
	TypePolyline:  true,
	TypeRectangle: true,
	TypeEllipse:   true,
	TypePolygon:   true,
	TypeText:      true,
	TypeStamp:     true,
}

const (
	ModeDirectorOnly = "director_only"
	ModeTurnLeader   = "turn_leader"
	ModeFreeform     = "freeform"
)

var ValidDrawingModes = map[string]bool{
	ModeDirectorOnly: true,
	ModeTurnLeader:   true,
	ModeFreeform:     true,
}

const (
	LineSolid  = "solid"
	LineDashed = "dashed"
	LineDotted = "dotted"
)

var ValidLineStyles = map[string]bool{
	LineSolid:  true,
	LineDashed: true,
	LineDotted: true,
}

const (
	DiagonalAlternating = "alternating_1_2"  // 5e-style 5/10/5 -- every other diagonal costs double.
	DiagonalEveryOne    = "every_diagonal_1" // D&D4e/simplified -- every diagonal costs the same as orthogonal.
	DiagonalEuclidean   = "euclidean"        // true geometric distance, ignoring grid step counting entirely.
)

var ValidDiagonalPolicies = map[string]bool{
	DiagonalAlternating: true,
	DiagonalEveryOne:    true,
	DiagonalEuclidean:   true,
}

// Security/abuse bounds (kernel §16). Conservative but generous enough for
// real drawing-as-play; a stroke or polygon this large is already well
// past what a mouse/touch gesture produces.
const (
	MaxPointsPerStroke = 2000
	MaxVertices        = 500
	MaxTextLength      = 500
	MaxStampKeyLength  = 64
	MaxObjectsPerShow  = 5000
)

// Object is one canonical drawing object (kernel §2).
type Object struct {
	ID                 string         `json:"id"`
	ShowID             string         `json:"show_id"`
	CohortID           string         `json:"cohort_id,omitempty"`
	CreatorUserID      string         `json:"creator_user_id"`
	CreatorCharacterID string         `json:"creator_character_id,omitempty"`
	ObjectType         string         `json:"object_type"`
	Geometry           map[string]any `json:"geometry"`
	StrokeColor        string         `json:"stroke_color"`
	FillColor          string         `json:"fill_color,omitempty"`
	StrokeWidth        float64        `json:"stroke_width"`
	Opacity            float64        `json:"opacity"`
	LineStyle          string         `json:"line_style"`
	Rotation           float64        `json:"rotation"`
	ZOrder             int            `json:"z_order"`
	Locked             bool           `json:"locked"`
	TextContent        string         `json:"text_content,omitempty"`
	StampKey           string         `json:"stamp_key,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`

	// HiddenBackstageOnly reports that this object is hidden from ordinary
	// viewers by Kernel 90 canonical state, and is present in this response
	// only because the reader is backstage (§14). Not stored on the row --
	// visibility lives in stage_object_states, keyed by Show, because Kernel
	// 87 drawings are Show-scoped while their visibility may differ per Show
	// viewer. Never true in a response to a viewer who is not permitted to
	// perceive the object, because such a viewer does not receive the object.
	HiddenBackstageOnly bool `json:"hidden_backstage_only,omitempty"`
}

// CreateRequest is the payload for creating one new drawing object.
type CreateRequest struct {
	SessionID          string         `json:"session_id"`
	Scope              string         `json:"scope"` // "cohort" (default) or "show"
	ObjectType         string         `json:"object_type"`
	Geometry           map[string]any `json:"geometry"`
	StrokeColor        string         `json:"stroke_color"`
	FillColor          string         `json:"fill_color"`
	StrokeWidth        float64        `json:"stroke_width"`
	Opacity            float64        `json:"opacity"`
	LineStyle          string         `json:"line_style"`
	Rotation           float64        `json:"rotation"`
	TextContent        string         `json:"text_content"`
	StampKey           string         `json:"stamp_key"`
	CreatorCharacterID string         `json:"creator_character_id"`
}

// UpdateRequest is a partial-update payload; nil pointer fields are left
// unchanged. Geometry, when present, fully replaces the stored geometry.
type UpdateRequest struct {
	SessionID   string         `json:"session_id"`
	Geometry    map[string]any `json:"geometry,omitempty"`
	StrokeColor *string        `json:"stroke_color,omitempty"`
	FillColor   *string        `json:"fill_color,omitempty"`
	StrokeWidth *float64       `json:"stroke_width,omitempty"`
	Opacity     *float64       `json:"opacity,omitempty"`
	LineStyle   *string        `json:"line_style,omitempty"`
	Rotation    *float64       `json:"rotation,omitempty"`
	TextContent *string        `json:"text_content,omitempty"`
}

// Settings is the per-Show drawing-mode + measured-tabletop configuration
// (kernel §7, §9).
type Settings struct {
	ShowID              string         `json:"show_id"`
	DrawingMode         string         `json:"drawing_mode"`
	ScaleGridUnits      float64        `json:"scale_grid_units"`
	ScaleRealUnits      float64        `json:"scale_real_units"`
	ScaleUnitLabel      string         `json:"scale_unit_label"`
	DiagonalPolicy      string         `json:"diagonal_policy"`
	GridlessCalibration map[string]any `json:"gridless_calibration,omitempty"`
}

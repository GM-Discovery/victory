package drawing

import "math"

// Point is one map-relative "world" coordinate, matching the same space
// drawing geometry and tokens already use.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// GridGeometry is the subset of venue_grid_configs the measurement tool
// needs (kernel §7 "Respect current hex orientation/config" -- this
// package does not own that table, callers pass in what they already
// loaded from backend/internal/venues' grid config).
type GridGeometry struct {
	GridType       string // "square", "hex", "none"
	CellSize       float64
	HexOrientation string // "flat-top" or "pointy-top"
}

// SquareGridDistance converts a straight-line world-space distance between
// two points on a square grid into a cell count, applying the Show's
// configured diagonal policy (kernel §7's required explicit setting).
// worldDelta is expressed in the same "world" pixels grid cellSize is
// measured in.
func SquareGridDistance(dx, dy, cellSize float64, policy string) float64 {
	if cellSize <= 0 {
		return 0
	}
	cellsX := math.Abs(dx) / cellSize
	cellsY := math.Abs(dy) / cellSize

	switch policy {
	case DiagonalEuclidean:
		return math.Hypot(cellsX, cellsY)
	case DiagonalEveryOne:
		// Chebyshev distance: diagonal step costs the same as orthogonal.
		return math.Max(cellsX, cellsY)
	case DiagonalAlternating:
		fallthrough
	default:
		// 5e-style "5/10/5": the 1st diagonal cell costs 1, the 2nd
		// costs 2 (running total 3 across 2 diagonal moves), the 3rd
		// costs 1 again, etc. -- closed form diagonal + floor(diagonal/2).
		straight := math.Abs(cellsX - cellsY)
		diagonal := math.Min(cellsX, cellsY)
		diagonalCost := diagonal + math.Floor(diagonal/2)
		return straight + diagonalCost
	}
}

// HexGridDistance returns the cell count between two axial-ish world
// points on a hex grid, respecting the configured orientation. cellSize
// matches frontend/lib/stage-runtime/geometry.js's convention exactly
// (venue_grid_configs.cell_size, the hex's corner-to-corner diameter --
// that file derives circumradius as size = cellSize / 2 and horizontal/
// vertical center-to-center spacing from there), so this stays pixel-
// consistent with how the grid is actually drawn rather than introducing
// a second "hex size" definition.
func HexGridDistance(dx, dy, cellSize float64, orientation string) float64 {
	if cellSize <= 0 {
		return 0
	}
	s := cellSize / 2
	var q, r float64
	if orientation == "pointy-top" {
		q = (math.Sqrt(3)/3*dx - 1.0/3*dy) / s
		r = (2.0 / 3 * dy) / s
	} else {
		q = (2.0 / 3 * dx) / s
		r = (-1.0/3*dx + math.Sqrt(3)/3*dy) / s
	}
	x := q
	z := r
	y := -x - z
	return (math.Abs(x) + math.Abs(y) + math.Abs(z)) / 2
}

// PathDistanceCells sums the grid-cell distance of consecutive segments
// in a multi-point path (kernel §7 "multi-segment/path").
func PathDistanceCells(points []Point, grid GridGeometry, diagonalPolicy string) float64 {
	if len(points) < 2 {
		return 0
	}
	total := 0.0
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		if grid.GridType == "hex" {
			total += HexGridDistance(dx, dy, grid.CellSize, grid.HexOrientation)
		} else if grid.GridType == "square" {
			total += SquareGridDistance(dx, dy, grid.CellSize, diagonalPolicy)
		} else {
			// Gridless: raw world-distance, converted via calibration by
			// the caller (CellsToRealUnits with grid_units=calibrated
			// world-distance-per-unit).
			total += math.Hypot(dx, dy)
		}
	}
	return total
}

// CellsToRealUnits converts a cell (or gridless calibrated-distance)
// count into the Show's configured physical scale, e.g. cells=3,
// scaleGridUnits=1, scaleRealUnits=5 -> 15 (ft).
func CellsToRealUnits(cells, scaleGridUnits, scaleRealUnits float64) float64 {
	if scaleGridUnits <= 0 {
		return 0
	}
	return cells / scaleGridUnits * scaleRealUnits
}

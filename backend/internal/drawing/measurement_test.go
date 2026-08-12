package drawing

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestSquareGridDistanceDiagonalPolicies(t *testing.T) {
	cellSize := 50.0
	// 3 cells right, 3 cells down -- a pure diagonal.
	dx, dy := 150.0, 150.0

	if got := SquareGridDistance(dx, dy, cellSize, DiagonalEveryOne); !almostEqual(got, 3) {
		t.Fatalf("every_diagonal_1: got %v, want 3", got)
	}
	if got := SquareGridDistance(dx, dy, cellSize, DiagonalAlternating); !almostEqual(got, 4) {
		// 3 diagonal cells: 1st costs 1, 2nd costs 2 (running total 3),
		// 3rd costs 1 again (running total 4) -- classic 5/10/5 pacing.
		t.Fatalf("alternating_1_2: got %v, want 4", got)
	}
	if got := SquareGridDistance(dx, dy, cellSize, DiagonalEuclidean); !almostEqual(got, 3*math.Sqrt2) {
		t.Fatalf("euclidean: got %v, want %v", got, 3*math.Sqrt2)
	}
}

func TestSquareGridDistanceOrthogonalUnaffectedByPolicy(t *testing.T) {
	cellSize := 50.0
	dx, dy := 200.0, 0.0
	for _, policy := range []string{DiagonalAlternating, DiagonalEveryOne, DiagonalEuclidean} {
		if got := SquareGridDistance(dx, dy, cellSize, policy); !almostEqual(got, 4) {
			t.Fatalf("policy %s: orthogonal got %v, want 4", policy, got)
		}
	}
}

func TestHexGridDistanceAdjacentCell(t *testing.T) {
	cellSize := 40.0
	// One flat-top hex step directly "east" (q+1,r0) is 1.5*size = 0.75*
	// cellSize on X, 0 on Y -- matches geometry.js's horizSpacing for
	// flat-top hexes exactly (hexWidth * 0.75, size = cellSize / 2).
	dx, dy := 0.75*cellSize, 0.0
	got := HexGridDistance(dx, dy, cellSize, "flat-top")
	if !almostEqual(math.Round(got), 1) {
		t.Fatalf("adjacent hex cell distance = %v, want ~1", got)
	}
}

func TestPathDistanceCellsSumsSegments(t *testing.T) {
	grid := GridGeometry{GridType: "square", CellSize: 50}
	points := []Point{{X: 0, Y: 0}, {X: 100, Y: 0}, {X: 100, Y: 100}}
	got := PathDistanceCells(points, grid, DiagonalEveryOne)
	if !almostEqual(got, 4) { // 2 cells + 2 cells
		t.Fatalf("path distance = %v, want 4", got)
	}
}

func TestPathDistanceCellsGridless(t *testing.T) {
	grid := GridGeometry{GridType: "none"}
	points := []Point{{X: 0, Y: 0}, {X: 30, Y: 40}}
	got := PathDistanceCells(points, grid, DiagonalEveryOne)
	if !almostEqual(got, 50) { // 3-4-5 triangle
		t.Fatalf("gridless path distance = %v, want 50", got)
	}
}

func TestCellsToRealUnits(t *testing.T) {
	// 1 square = 5 ft, 3 cells -> 15 ft.
	got := CellsToRealUnits(3, 1, 5)
	if !almostEqual(got, 15) {
		t.Fatalf("CellsToRealUnits = %v, want 15", got)
	}
	// 1 hex = 2 miles, 4 cells -> 8 miles.
	got = CellsToRealUnits(4, 1, 2)
	if !almostEqual(got, 8) {
		t.Fatalf("CellsToRealUnits = %v, want 8", got)
	}
}

package network

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
)

// Kernel 72A: the former hardcoded allowlist (normalizeSessionControlVenueSlug)
// became the session_control_enabled venues.config flag, seeded by migration
// 055 for exactly the same four venues — asserted here against the real
// migrated test database, plus fail-closed behavior for unknown/blank slugs.
func TestVenueSessionControlEnabled(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()
	ctx := context.Background()

	cases := map[string]bool{
		"the-cave":            true,
		"first-theater":       true,
		"catharsis":           true,
		"middle-school-stage": true,
		"greenroom":           false,
		"unknown-venue":       false,
		"":                    false,
	}

	for slug, want := range cases {
		got, err := VenueSessionControlEnabled(ctx, pool, slug)
		if err != nil {
			t.Fatalf("VenueSessionControlEnabled(%q): %v", slug, err)
		}
		if got != want {
			t.Fatalf("VenueSessionControlEnabled(%q) = %v, want %v", slug, got, want)
		}
	}
}

func TestNormalizeSessionControlCommand(t *testing.T) {
	cases := map[string]string{
		"":       "status",
		"status": "status",
		"start":  "start",
		"end":    "end",
		"banana": "",
	}

	for input, want := range cases {
		if got := normalizeSessionControlCommand(input); got != want {
			t.Fatalf("normalizeSessionControlCommand(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSessionControlStateAndMessage(t *testing.T) {
	row := sessionControlRow{}
	if got := sessionControlState(row); got != "closed" {
		t.Fatalf("expected closed for empty row, got %q", got)
	}
	if got := sessionControlMessage(row); got != "Session closed. Use /session start to open it." {
		t.Fatalf("unexpected closed message: %q", got)
	}

	row.Found = true
	row.Showing.ID = "showing-1"
	row.Showing.Status = "rehearsal"
	if got := sessionControlState(row); got != "ready" {
		t.Fatalf("expected ready for rehearsal, got %q", got)
	}

	row.Showing.Status = "live"
	if got := sessionControlState(row); got != "active" {
		t.Fatalf("expected active for live, got %q", got)
	}
	if got := sessionControlMessage(row); got != "Session active. Showing open." {
		t.Fatalf("unexpected active message: %q", got)
	}
}

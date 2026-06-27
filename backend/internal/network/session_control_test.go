package network

import (
	"testing"
)

func TestNormalizeSessionControlVenueSlug(t *testing.T) {
	cases := map[string]string{
		"the-cave":            "the-cave",
		"first-theater":       "first-theater",
		"catharsis":           "catharsis",
		"middle-school-stage": "middle-school-stage",
		"unknown":             "",
	}

	for input, want := range cases {
		if got := normalizeSessionControlVenueSlug(input); got != want {
			t.Fatalf("normalizeSessionControlVenueSlug(%q) = %q, want %q", input, got, want)
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

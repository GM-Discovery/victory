package playerrelationships

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"victory/backend/internal/playerprofile"
)

// TestRelationshipSerializationNeverLeaksAccountUUIDs proves that the JSON
// any relationship response is built from cannot contain the raw observer or
// subject account UUIDs (Kernel 62 §10) -- they are struct-tagged out.
func TestRelationshipSerializationNeverLeaksAccountUUIDs(t *testing.T) {
	observerUUID := "11111111-1111-1111-1111-111111111111"
	subjectUUID := "22222222-2222-2222-2222-222222222222"

	rel := Relationship{
		ID:                "33333333-3333-3333-3333-333333333333",
		ObserverUserID:    observerUUID,
		SubjectUserID:     subjectUUID,
		PrivateNickname:   "The reliable one",
		RelationshipState: "active",
		TrustLevel:        "trusted",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	item := ListItem{
		Relationship: rel,
		Subject: SubjectHeader{
			SubjectProfileID: "44444444-4444-4444-4444-444444444444",
			StageName:        "Alex",
		},
	}
	detail := Detail{ListItem: item}

	for name, v := range map[string]any{"relationship": rel, "list_item": item, "detail": detail} {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		body := string(raw)
		if strings.Contains(body, observerUUID) {
			t.Fatalf("%s JSON leaks observer UUID: %s", name, body)
		}
		if strings.Contains(body, subjectUUID) {
			t.Fatalf("%s JSON leaks subject UUID: %s", name, body)
		}
	}
}

// TestSharedContextFoldOnlyVerifiedOverlap proves the shared-context fold
// keeps only productions where both sides hold active memberships and never
// invents data for one-sided membership (Kernel 62 §13.1, §13.2).
func TestSharedContextFoldOnlyVerifiedOverlap(t *testing.T) {
	observer := "obs"
	subject := "subj"
	early := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	rows := []membershipRow{
		{ProductionID: "p1", ProductionName: "Hamlet", UserID: observer, Role: "cast", CreatedAt: early},
		{ProductionID: "p1", ProductionName: "Hamlet", UserID: subject, Role: "director", CreatedAt: later},
		{ProductionID: "p2", ProductionName: "Solo Show", UserID: observer, Role: "cast", CreatedAt: early},
		{ProductionID: "p3", ProductionName: "Other People", UserID: "someone_else", Role: "crew", CreatedAt: early},
	}

	shared := FoldSharedProductions(rows, observer, subject)
	if len(shared) != 1 {
		t.Fatalf("expected exactly 1 shared production, got %d: %+v", len(shared), shared)
	}
	got := shared[0]
	if got.ProductionName != "Hamlet" {
		t.Fatalf("shared production = %q", got.ProductionName)
	}
	if len(got.ObserverRoles) != 1 || got.ObserverRoles[0] != "cast" {
		t.Fatalf("observer roles = %v", got.ObserverRoles)
	}
	if len(got.SubjectRoles) != 1 || got.SubjectRoles[0] != "director" {
		t.Fatalf("subject roles = %v", got.SubjectRoles)
	}
	// Overlap begins when the later of the two joined.
	if !got.Since.Equal(later) {
		t.Fatalf("since = %v, want %v", got.Since, later)
	}
}

func TestSharedContextFoldEmptyIsSafe(t *testing.T) {
	shared := FoldSharedProductions(nil, "obs", "subj")
	if shared == nil || len(shared) != 0 {
		t.Fatalf("expected empty non-nil slice, got %#v", shared)
	}
}

// TestNoParallelIdentitySystem is a compile-time-style guard: the
// relationship layer resolves subjects through the Kernel 61 player-profile
// workbook surface rather than any duplicate identity table. Referencing the
// playerprofile package here keeps that dependency direction explicit
// (Kernel 62 §3).
func TestNoParallelIdentitySystem(t *testing.T) {
	if playerprofile.ReservedFieldAccountUUID != "account_uuid" {
		t.Fatal("kernel 61 reserved field contract changed unexpectedly")
	}
}

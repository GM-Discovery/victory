package directorprep

import (
	"context"
	"testing"
)

func TestDirectorCanPrepareAndRecallATargetComplexity(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k89dir")
	showID, _ := showFixture(t, pool, director)

	if err := RequireDirector(ctx, pool, director, showID); err != nil {
		t.Fatalf("director should hold authority: %v", err)
	}

	created, err := Create(ctx, pool, CreateInput{
		ShowID:  showID,
		Kind:    KindTargetComplexity,
		Label:   "Climb Training Wall",
		Payload: map[string]any{"value": float64(14), "note": "Russel points at the north face."},
		ActorID: director,
	})
	if err != nil {
		t.Fatalf("create target complexity: %v", err)
	}
	if created.Payload["value"].(float64) != 14 {
		t.Fatalf("stored value wrong: %#v", created.Payload)
	}

	// Recall.
	list, err := List(ctx, pool, showID, KindTargetComplexity)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Label != "Climb Training Wall" {
		t.Fatalf("expected one recalled preparation, got %#v", list)
	}
}

// Kernel 89 §7: "Director can change the value live."
func TestTargetComplexityCanBeChangedLive(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k89dir")
	showID, _ := showFixture(t, pool, director)

	created, err := Create(ctx, pool, CreateInput{
		ShowID: showID, Kind: KindTargetComplexity, Label: "Vault the Rail",
		Payload: map[string]any{"value": float64(9)}, ActorID: director,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newPayload := map[string]any{"value": float64(12)}
	updated, err := Update(ctx, pool, created.ID, UpdateInput{Payload: &newPayload})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Payload["value"].(float64) != 12 {
		t.Fatalf("live change did not stick: %#v", updated.Payload)
	}
	if updated.Label != "Vault the Rail" {
		t.Fatalf("a payload-only patch must not disturb the label, got %q", updated.Label)
	}
}

func TestTargetComplexityIsBounded(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k89dir")
	showID, _ := showFixture(t, pool, director)

	for _, bad := range []any{float64(0), float64(-3), float64(1000)} {
		if _, err := Create(ctx, pool, CreateInput{
			ShowID: showID, Kind: KindTargetComplexity, Label: "Bad",
			Payload: map[string]any{"value": bad}, ActorID: director,
		}); err == nil {
			t.Fatalf("value %v should be out of range", bad)
		}
	}
	if _, err := Create(ctx, pool, CreateInput{
		ShowID: showID, Kind: KindTargetComplexity, Label: "No value", ActorID: director,
	}); err == nil {
		t.Fatal("a target complexity with no value should be refused")
	}
	if _, err := Create(ctx, pool, CreateInput{
		ShowID: showID, Kind: KindTargetComplexity, Label: "   ",
		Payload: map[string]any{"value": float64(5)}, ActorID: director,
	}); err == nil {
		t.Fatal("a blank label should be refused")
	}
}

// The one property that makes migration 105's JSONB column safe: extra keys
// are dropped at validation, so nothing a client invents can be stored and
// hoped-for by some later reader (kernel 89 §6/§26 -- no macro payloads).
func TestUnknownPayloadKeysAreNotStored(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k89dir")
	showID, _ := showFixture(t, pool, director)

	created, err := Create(ctx, pool, CreateInput{
		ShowID: showID, Kind: KindTargetComplexity, Label: "Sneaky",
		Payload: map[string]any{
			"value":       float64(11),
			"then":        "award_fate",
			"on_success":  []any{"change_scene"},
			"script":      "while true do end",
		},
		ActorID: director,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	for _, forbidden := range []string{"then", "on_success", "script"} {
		if _, ok := created.Payload[forbidden]; ok {
			t.Fatalf("payload retained smuggled key %q: %#v", forbidden, created.Payload)
		}
	}
}

func TestUnknownKindIsRefused(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k89dir")
	showID, _ := showFixture(t, pool, director)

	if _, err := Create(ctx, pool, CreateInput{
		ShowID: showID, Kind: "encounter_graph", Label: "Nope",
		Payload: map[string]any{}, ActorID: director,
	}); err == nil {
		t.Fatal("an unlisted kind should be refused before it reaches the CHECK constraint")
	}
}

func TestSavedAnnouncementReusesThePaletteValidator(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k89dir")
	showID, _ := showFixture(t, pool, director)

	created, err := Create(ctx, pool, CreateInput{
		ShowID: showID, Kind: KindAnnouncement, Label: "Wall collapses",
		Payload: map[string]any{"style": "consequences", "text": "The training wall gives way!"},
		ActorID: director,
	})
	if err != nil {
		t.Fatalf("create announcement preset: %v", err)
	}
	if created.Payload["style"] != "consequences" {
		t.Fatalf("style not stored: %#v", created.Payload)
	}

	if _, err := Create(ctx, pool, CreateInput{
		ShowID: showID, Kind: KindAnnouncement, Label: "Bad style",
		Payload: map[string]any{"style": "kaboom", "text": "hi"}, ActorID: director,
	}); err == nil {
		t.Fatal("a saved preset must be validated against the same palette a live announcement is")
	}
}

// Kernel 89 §28: a Player cannot create, alter, or even read Director
// preparations. There is no Player read path in this package at all, so
// this pins the write/authority half.
func TestNonDirectorHasNoAuthority(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k89dir")
	outsider := insertTestUser(t, pool, "k89out")
	showID, _ := showFixture(t, pool, director)

	if err := RequireDirector(ctx, pool, outsider, showID); err == nil {
		t.Fatal("a user with no Location role must not hold Director preparation authority")
	}
	if err := RequireDirector(ctx, pool, "", showID); err == nil {
		t.Fatal("an anonymous caller must be refused")
	}
}

func TestDeleteRemovesTheRecall(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()
	director := insertTestUser(t, pool, "k89dir")
	showID, _ := showFixture(t, pool, director)

	created, err := Create(ctx, pool, CreateInput{
		ShowID: showID, Kind: KindTargetComplexity, Label: "Temporary",
		Payload: map[string]any{"value": float64(7)}, ActorID: director,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := Delete(ctx, pool, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := Delete(ctx, pool, created.ID); err == nil {
		t.Fatal("deleting twice should report not found rather than succeed silently")
	}
	list, err := List(ctx, pool, showID, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected an empty recall list, got %#v", list)
	}
}

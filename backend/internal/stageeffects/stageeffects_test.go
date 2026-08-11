package stageeffects

import (
	"testing"
	"time"
)

func TestCreateGetPinDismiss(t *testing.T) {
	r := NewRegistry()

	e := r.Create("session-1", Effect{
		Type:           "dice_roll",
		SourceActionID: "action-1",
		Audience:       "cohort",
		CohortID:       "cohort-1",
		ActorID:        "user-1",
		DurationMs:     3500,
	})
	if e.ID == "" {
		t.Fatal("expected a generated effect ID")
	}
	if e.SessionID != "session-1" {
		t.Fatalf("SessionID = %q, want session-1", e.SessionID)
	}
	if e.Pinned {
		t.Fatal("a freshly created effect must not start pinned")
	}

	got, ok := r.Get("session-1", e.ID)
	if !ok {
		t.Fatal("expected to find the just-created effect")
	}
	if got.ActorID != "user-1" {
		t.Fatalf("ActorID = %q, want user-1", got.ActorID)
	}

	if _, ok := r.Get("session-2", e.ID); ok {
		t.Fatal("an effect must not be visible under a different session ID")
	}

	pinned, ok := r.Pin("session-1", e.ID)
	if !ok || !pinned.Pinned {
		t.Fatalf("expected Pin to succeed and mark Pinned=true, ok=%v pinned=%+v", ok, pinned)
	}

	if !r.Dismiss("session-1", e.ID) {
		t.Fatal("expected Dismiss to succeed on an existing (pinned) effect")
	}
	if _, ok := r.Get("session-1", e.ID); ok {
		t.Fatal("expected the effect to be gone after Dismiss")
	}
	if r.Dismiss("session-1", e.ID) {
		t.Fatal("expected a second Dismiss of the same effect to report false")
	}
}

func TestPinDismissUnknownEffectReportFalse(t *testing.T) {
	r := NewRegistry()
	if _, ok := r.Pin("session-1", "does-not-exist"); ok {
		t.Fatal("expected Pin on an unknown effect to report false")
	}
	if r.Dismiss("session-1", "does-not-exist") {
		t.Fatal("expected Dismiss on an unknown effect to report false")
	}
}

func TestTransientEffectExpiresButPinnedSurvives(t *testing.T) {
	r := NewRegistry()

	transient := r.Create("session-1", Effect{Type: "dice_roll", ActorID: "user-1", DurationMs: 1})
	pinned := r.Create("session-1", Effect{Type: "dice_roll", ActorID: "user-1", DurationMs: 1})
	if _, ok := r.Pin("session-1", pinned.ID); !ok {
		t.Fatal("expected to pin the second effect")
	}

	// Force both CreatedAt timestamps into the past so the next sweep
	// (triggered by any registry call for this session) treats the
	// transient one as expired, well past DurationMs+expiryGrace.
	forceEffectAge(r, "session-1", transient.ID, 10*time.Minute)
	forceEffectAge(r, "session-1", pinned.ID, 10*time.Minute)

	if _, ok := r.Get("session-1", transient.ID); ok {
		t.Fatal("expected the transient effect to have been swept after expiry")
	}
	if _, ok := r.Get("session-1", pinned.ID); !ok {
		t.Fatal("expected the pinned effect to survive the same sweep")
	}

	pins := r.ListPinned("session-1")
	if len(pins) != 1 || pins[0].ID != pinned.ID {
		t.Fatalf("ListPinned = %+v, want exactly [pinned]", pins)
	}
}

// forceEffectAge reaches directly into the registry's internal map to
// backdate CreatedAt -- there is no public API for this (deliberately: no
// caller should ever need to lie about when an effect was created), so the
// test uses the same package-internal access its own sweep logic does.
func forceEffectAge(r *Registry, sessionID, effectID string, age time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if effects, ok := r.sessions[sessionID]; ok {
		if e, ok := effects[effectID]; ok {
			e.CreatedAt = time.Now().Add(-age)
		}
	}
}

func TestEndSessionClearsEverything(t *testing.T) {
	r := NewRegistry()
	e := r.Create("session-1", Effect{Type: "dice_roll", ActorID: "user-1", DurationMs: 3500})
	if _, ok := r.Pin("session-1", e.ID); !ok {
		t.Fatal("expected to pin the effect")
	}

	r.EndSession("session-1")

	if _, ok := r.Get("session-1", e.ID); ok {
		t.Fatal("expected EndSession to clear even a pinned effect")
	}
	if pins := r.ListPinned("session-1"); len(pins) != 0 {
		t.Fatalf("expected no pinned effects after EndSession, got %+v", pins)
	}
}

func TestListPinnedIsScopedPerSession(t *testing.T) {
	r := NewRegistry()
	a := r.Create("session-a", Effect{Type: "dice_roll", ActorID: "user-1", DurationMs: 3500})
	b := r.Create("session-b", Effect{Type: "dice_roll", ActorID: "user-1", DurationMs: 3500})
	if _, ok := r.Pin("session-a", a.ID); !ok {
		t.Fatal("expected to pin effect a")
	}
	if _, ok := r.Pin("session-b", b.ID); !ok {
		t.Fatal("expected to pin effect b")
	}

	pinsA := r.ListPinned("session-a")
	if len(pinsA) != 1 || pinsA[0].ID != a.ID {
		t.Fatalf("session-a ListPinned = %+v, want exactly [a]", pinsA)
	}
	pinsB := r.ListPinned("session-b")
	if len(pinsB) != 1 || pinsB[0].ID != b.ID {
		t.Fatalf("session-b ListPinned = %+v, want exactly [b]", pinsB)
	}
}

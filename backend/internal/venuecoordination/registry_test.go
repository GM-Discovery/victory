package venuecoordination

import "testing"

func TestEnsureSessionInitializesOnce(t *testing.T) {
	r := NewRegistry()
	s := r.EnsureSession("sess1", "storyboards", "board1", "owner-1")
	if s.GroupLeaderUserID != "owner-1" {
		t.Fatalf("expected leader owner-1, got %q", s.GroupLeaderUserID)
	}
	if s.CurrentTurnUserID != "" {
		t.Fatalf("expected current turn unset, got %q", s.CurrentTurnUserID)
	}

	// A second EnsureSession call for the same ID (a later joiner) must not
	// re-initialize or change the already-active session.
	s2 := r.EnsureSession("sess1", "storyboards", "board1", "someone-else")
	if s2.GroupLeaderUserID != "owner-1" {
		t.Fatalf("later EnsureSession must not overwrite leader, got %q", s2.GroupLeaderUserID)
	}
}

func TestEnsureSessionOwnerNotPresentLeavesLeaderUnset(t *testing.T) {
	r := NewRegistry()
	s := r.EnsureSession("sess2", "storyboards", "board2", "")
	if s.GroupLeaderUserID != "" {
		t.Fatalf("expected leader unset when owner not present, got %q", s.GroupLeaderUserID)
	}
}

func TestSetGroupLeaderRequiresActiveSession(t *testing.T) {
	r := NewRegistry()
	if _, ok := r.SetGroupLeader("no-session", "u1"); ok {
		t.Fatalf("expected false for inactive session")
	}
}

func TestSetGroupLeaderAndCurrentTurn(t *testing.T) {
	r := NewRegistry()
	r.EnsureSession("sess3", "storyboards", "board3", "owner")

	s, ok := r.SetGroupLeader("sess3", "alice")
	if !ok || s.GroupLeaderUserID != "alice" {
		t.Fatalf("expected leader alice, got %+v ok=%v", s, ok)
	}

	s, ok = r.SetCurrentTurn("sess3", "bob")
	if !ok || s.CurrentTurnUserID != "bob" {
		t.Fatalf("expected turn bob, got %+v ok=%v", s, ok)
	}

	// Idempotent repeat assignment.
	s, ok = r.SetCurrentTurn("sess3", "bob")
	if !ok || s.CurrentTurnUserID != "bob" {
		t.Fatalf("expected idempotent repeat to remain bob, got %+v ok=%v", s, ok)
	}
}

func TestEndSessionClearsAndLaterSessionStartsFresh(t *testing.T) {
	r := NewRegistry()
	r.EnsureSession("sess4", "storyboards", "board4", "owner")
	r.SetGroupLeader("sess4", "alice")
	r.SetCurrentTurn("sess4", "bob")

	r.EndSession("sess4")
	if _, ok := r.Get("sess4"); ok {
		t.Fatalf("expected no session after EndSession")
	}

	fresh := r.EnsureSession("sess4", "storyboards", "board4", "owner")
	if fresh.GroupLeaderUserID != "owner" {
		t.Fatalf("expected fresh session to re-init leader to owner, got %q", fresh.GroupLeaderUserID)
	}
	if fresh.CurrentTurnUserID != "" {
		t.Fatalf("expected fresh session current turn unset, got %q", fresh.CurrentTurnUserID)
	}
}

func TestDisconnectDoesNotAutoReassign(t *testing.T) {
	// This package has no concept of "disconnect" at all -- state simply
	// stays whatever it was set to until an explicit Set call or
	// EndSession. This test documents that guarantee directly: nothing in
	// Registry ever mutates state on its own.
	r := NewRegistry()
	r.EnsureSession("sess5", "storyboards", "board5", "owner")
	r.SetCurrentTurn("sess5", "alice")

	s, _ := r.Get("sess5")
	if s.CurrentTurnUserID != "alice" {
		t.Fatalf("expected turn to remain alice absent any explicit mutation, got %q", s.CurrentTurnUserID)
	}
}

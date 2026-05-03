package messages

import (
	"strings"
	"testing"
)

func TestSanitizeCreateMessageRequest(t *testing.T) {
	req := createMessageRequest{
		ToUserID: "  user-1  ",
		Subject:  "  Hello  ",
		Body:     "  " + strings.Repeat("x", 260) + "  ",
	}

	got := sanitizeCreateMessageRequest(req)
	if got.ToUserID != "user-1" {
		t.Fatalf("expected trimmed to_user_id, got %q", got.ToUserID)
	}
	if got.Subject != "Hello" {
		t.Fatalf("expected trimmed subject, got %q", got.Subject)
	}
	if len([]rune(got.Body)) > 250 {
		t.Fatalf("expected body to be capped at 250 runes, got %d", len([]rune(got.Body)))
	}
}

func TestTruncateRunes(t *testing.T) {
	got := truncateRunes("Victory", 10)
	if got != "Victory" {
		t.Fatalf("unexpected truncate result: %q", got)
	}

	got = truncateRunes("A long mailbox body", 5)
	if got != "A lon" {
		t.Fatalf("expected truncated prefix, got %q", got)
	}
}

func TestSanitizeNoteCardRequest(t *testing.T) {
	req := noteCardRequest{
		ToUserID:        "  user-1  ",
		ToParticipantID: "  participant-1  ",
		Subject:         "  Hello  ",
		Body:            "  body  ",
	}

	got := sanitizeNoteCardRequest(req)
	if got.ToUserID != "user-1" || got.ToParticipantID != "participant-1" {
		t.Fatalf("expected note card recipient fields to be trimmed, got %+v", got)
	}
	if got.Subject != "Hello" || got.Body != "body" {
		t.Fatalf("expected note card text fields to be trimmed, got %+v", got)
	}
}

func TestIsNoteCardRecipientRole(t *testing.T) {
	cases := map[string]bool{
		"director": true,
		"cast":     true,
		"crew":     true,
		"audience": false,
		"producer": false,
		"":         false,
	}

	for input, want := range cases {
		if got := isNoteCardRecipientRole(input); got != want {
			t.Fatalf("role %q: want %v, got %v", input, want, got)
		}
	}
}

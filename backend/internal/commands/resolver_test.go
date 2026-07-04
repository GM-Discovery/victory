package commands

import (
	"context"
	"testing"
)

func TestResolveActiveCharacterNoUserID(t *testing.T) {
	persona, source, err := ResolveActiveCharacter(context.Background(), nil, "", "session-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if persona != nil {
		t.Fatalf("expected nil persona for empty userID, got %+v", persona)
	}
	if source != "" {
		t.Fatalf("expected empty source for empty userID, got %q", source)
	}
}

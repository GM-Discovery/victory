package commands

import "testing"

func TestHelpAllCommandsMatchesRegistry(t *testing.T) {
	entries, err := Help("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != len(AllCommands()) {
		t.Fatalf("Help(\"\") returned %d entries, want %d", len(entries), len(AllCommands()))
	}
}

func TestHelpSingleCommand(t *testing.T) {
	entries, err := Help("char")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 || entries[0].Path != "char" {
		t.Fatalf("Help(\"char\") = %+v, want single char entry", entries)
	}
}

func TestHelpUnknownCommand(t *testing.T) {
	if _, err := Help("nonexistent"); err == nil {
		t.Fatalf("expected error for unknown command")
	}
}

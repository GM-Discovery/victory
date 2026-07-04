package commands

import "testing"

func TestAllCommandsNoDuplicatePaths(t *testing.T) {
	seen := map[string]bool{}
	for _, cmd := range AllCommands() {
		path := normalizePath(cmd.Path)
		if seen[path] {
			t.Fatalf("duplicate command path: %s", cmd.Path)
		}
		seen[path] = true
	}
}

func TestFindResolvesPathCaseInsensitively(t *testing.T) {
	cmd, ok := Find("Char")
	if !ok {
		t.Fatalf("expected to find char command")
	}
	if cmd.Path != "char" {
		t.Fatalf("expected path 'char', got %q", cmd.Path)
	}
}

func TestFindUnknownCommand(t *testing.T) {
	if _, ok := Find("nonexistent"); ok {
		t.Fatalf("expected unknown command to not be found")
	}
}

func TestMicAndSessionAreLegacy(t *testing.T) {
	for _, path := range []string{"mic", "session"} {
		cmd, ok := Find(path)
		if !ok {
			t.Fatalf("expected to find %s command", path)
		}
		if !cmd.Legacy {
			t.Fatalf("expected %s to be marked legacy", path)
		}
		if cmd.LegacyEndpoint == "" {
			t.Fatalf("expected %s to have a legacy endpoint", path)
		}
	}
}

func TestNonLegacyCommandsAreNotLegacy(t *testing.T) {
	for _, path := range []string{"char", "bio", "quote", "journal", "ooc", "help"} {
		cmd, ok := Find(path)
		if !ok {
			t.Fatalf("expected to find %s command", path)
		}
		if cmd.Legacy {
			t.Fatalf("expected %s to not be marked legacy", path)
		}
	}
}

func TestAvailableWithoutVenueFilterReturnsAll(t *testing.T) {
	if got, want := len(Available("")), len(AllCommands()); got != want {
		t.Fatalf("Available(\"\") returned %d commands, want %d", got, want)
	}
	if got, want := len(Available("first-theater")), len(AllCommands()); got != want {
		t.Fatalf("Available(\"first-theater\") returned %d commands, want %d (no command in this pass restricts venues)", got, want)
	}
}

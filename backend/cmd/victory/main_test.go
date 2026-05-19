package main

import "testing"

func TestDiscordOAuthConfigFromEnvScoping(t *testing.T) {
	t.Setenv("DISCORD_OAUTH_SCOPES", "identify,email")
	got := parseDiscordOAuthScopes("identify,email")
	if len(got) != 2 || got[0] != "identify" || got[1] != "email" {
		t.Fatalf("unexpected scopes: %#v", got)
	}
}

func TestParseBoolish(t *testing.T) {
	for _, input := range []string{"true", "1", "yes", "on", "T"} {
		if !parseBoolish(input) {
			t.Fatalf("expected %q to be truthy", input)
		}
	}

	for _, input := range []string{"false", "0", "no", ""} {
		if parseBoolish(input) {
			t.Fatalf("expected %q to be falsey", input)
		}
	}
}

func TestDiscordOAuthConfigFromEnvDisabledWithoutVars(t *testing.T) {
	t.Setenv("DISCORD_OAUTH_ENABLED", "")
	t.Setenv("DISCORD_CLIENT_ID", "")
	t.Setenv("DISCORD_CLIENT_SECRET", "")
	t.Setenv("DISCORD_REDIRECT_URL", "")

	cfg := discordOAuthConfigFromEnv()
	if cfg.Enabled {
		t.Fatalf("expected config to be disabled without required vars")
	}
}

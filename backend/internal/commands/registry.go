// Package commands is the single server-authoritative registry, resolver,
// and execution surface for venue slash commands (Kernel 59, descoped pass).
//
// Scope note: this pass only implements the commands that can be built on
// top of primitives that genuinely exist today (characters.UpdateCard's
// whole-row overwrite, the character_journals table, the actions/chat
// pipeline). It deliberately excludes /value, /override, skill-add, and
// collection-item commands (relationships/quotes-list/custom-fields), since
// none of the domain layer those would need (a mechanical value registry,
// persisted priority/lock state, a collection-item store, a real
// Director-authority gate) exists in the codebase yet. See the Kernel 59
// plan for the full rationale.
package commands

import "strings"

// Command is the metadata shape shared by parsing, the /help output, and the
// frontend command palette. Everything that needs to know "what commands
// exist and what do they look like" reads from AllCommands()/Find()/
// Available() rather than maintaining a separate hardcoded list.
type Command struct {
	Path            string   `json:"path"`
	Aliases         []string `json:"aliases,omitempty"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Usage           string   `json:"usage"`
	Category        string   `json:"category"`
	Subcommands     []string `json:"subcommands,omitempty"`
	VenueSlugs      []string `json:"venue_slugs,omitempty"` // empty/nil = all venues
	Legacy          bool     `json:"legacy"`
	LegacyEndpoint  string   `json:"legacy_endpoint,omitempty"`
	RequiresPreview bool     `json:"requires_preview"`
	RequiresActive  bool     `json:"requires_active_character"`
}

func AllCommands() []Command {
	return []Command{
		{
			Path:           "mic",
			Title:          "Microphone control",
			Description:    "Turn the Discord mic bridge on or off, or check its status.",
			Usage:          "/mic hot|off|status",
			Category:       "system",
			Subcommands:    []string{"hot", "on", "start", "off", "status"},
			Legacy:         true,
			LegacyEndpoint: "/api/discord/mic/control",
		},
		{
			Path:           "session",
			Title:          "Session control",
			Description:    "Start, end, or check the status of the current venue session.",
			Usage:          "/session start|end|status",
			Category:       "system",
			Subcommands:    []string{"start", "end", "status"},
			Legacy:         true,
			LegacyEndpoint: "/api/session/control",
		},
		{
			// Rolls execute through the venue dice tray/websocket, not the
			// generic command executor -- registered here so the palette,
			// /help, and the Guide tab list it.
			Path:        "roll",
			Aliases:     []string{"r"},
			Title:       "Roll dice",
			Description: "Roll dice publicly. Add ! for exploding dice (each max face rolls a bonus die).",
			Usage:       "/roll XdY[!][+Z]",
			Category:    "dice",
			Legacy:      true,
		},
		{
			Path:           "char",
			Title:          "Character",
			Description:    "Edit your active character's name, pronouns, or Token Aura, or jump to a workbook page.",
			Usage:          "/char set name|pronouns|aura <value>",
			Category:       "character",
			Subcommands:    []string{"set name", "set pronouns", "set aura", "face", "mechanics", "history", "journal"},
			RequiresActive: true,
		},
		{
			Path:           "bio",
			Title:          "Biography",
			Description:    "Replace your active character's biography.",
			Usage:          "/bio set <text>",
			Category:       "character",
			Subcommands:    []string{"set"},
			RequiresActive: true,
		},
		{
			Path:           "quote",
			Title:          "Featured quote",
			Description:    "Replace your active character's featured quote.",
			Usage:          "/quote set <text>",
			Category:       "character",
			Subcommands:    []string{"set"},
			RequiresActive: true,
		},
		{
			Path:           "journal",
			Title:          "Journal",
			Description:    "Add a private journal entry for your active character, or view recent entries. Never public.",
			Usage:          "/journal add <text>",
			Category:       "character",
			Subcommands:    []string{"add", "recent"},
			RequiresActive: true,
		},
		{
			Path:        "ooc",
			Title:       "Out of character",
			Description: "Send an out-of-character message to the venue chat.",
			Usage:       "/ooc <message>",
			Category:    "ooc",
		},
		{
			Path:        "help",
			Title:       "Help",
			Description: "List available commands, or show usage for one command.",
			Usage:       "/help [command]",
			Category:    "system",
		},
	}
}

// Find resolves path (case-insensitively) against a command's Path or any of
// its Aliases.
func Find(path string) (Command, bool) {
	needle := normalizePath(path)
	for _, cmd := range AllCommands() {
		if normalizePath(cmd.Path) == needle {
			return cmd, true
		}
		for _, alias := range cmd.Aliases {
			if normalizePath(alias) == needle {
				return cmd, true
			}
		}
	}
	return Command{}, false
}

// Available filters AllCommands() by venue. A command with an empty
// VenueSlugs list is available everywhere.
func Available(venueSlug string) []Command {
	venueSlug = normalizePath(venueSlug)
	out := make([]Command, 0, len(AllCommands()))
	for _, cmd := range AllCommands() {
		if len(cmd.VenueSlugs) == 0 || venueSlug == "" {
			out = append(out, cmd)
			continue
		}
		for _, slug := range cmd.VenueSlugs {
			if normalizePath(slug) == venueSlug {
				out = append(out, cmd)
				break
			}
		}
	}
	return out
}

func normalizePath(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

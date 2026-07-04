package commands

import "strings"

// HelpEntry is the shape rendered by /help.
type HelpEntry struct {
	Path        string `json:"path"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Usage       string `json:"usage"`
}

// Help renders /help (all commands) or /help <command> (usage for one).
// Pure function over the registry -- no DB access.
func Help(path string) ([]HelpEntry, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		out := make([]HelpEntry, 0, len(AllCommands()))
		for _, cmd := range AllCommands() {
			out = append(out, HelpEntry{Path: cmd.Path, Title: cmd.Title, Description: cmd.Description, Usage: cmd.Usage})
		}
		return out, nil
	}

	cmd, ok := Find(path)
	if !ok {
		return nil, errUnknownCommand
	}
	return []HelpEntry{{Path: cmd.Path, Title: cmd.Title, Description: cmd.Description, Usage: cmd.Usage}}, nil
}

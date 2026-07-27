package aftercare

import (
	"encoding/csv"
	"io"
	"strings"
	"time"
)

// CSV export (kernel-75 S9.4).
//
// This is the repo's FIRST file download. Every other handler returns the
// {ok, data} JSON envelope, so the conventions established here are worth
// stating explicitly for whoever writes the second one:
//
//   - The route ends in `.csv` rather than taking ?format=csv. It is
//     greppable, and the browser's Save-As default filename comes out right.
//   - Authority is resolved and every row is buffered BEFORE any byte is
//     written. Streaming pgx.Rows straight into a csv.Writer makes a
//     mid-stream failure unreportable, because the 200 and the headers have
//     already gone. Row counts here are bounded by roster size.
//   - On any error the handler falls back to the ordinary JSON error
//     envelope -- legal precisely because nothing has been written yet.
//
// S9.4 sketches a folder tree (Show Run / Show / Session / aftercare.csv).
// A web download cannot create folders, so the equivalent metadata is
// preserved in the filename and in the columns, which S9.4 explicitly
// permits.

// sanitizeCell defends against CSV injection.
//
// NOT optional, and not a nicety. These cells are free text a Player wrote,
// and a Director will open the file in Excel or Sheets. A response beginning
// with =, +, -, @, TAB or CR is interpreted as a FORMULA by every major
// spreadsheet, which turns "export a reflection" into "execute what a
// participant typed on the Director's machine". Prefixing a single quote
// makes the cell inert while leaving the text readable.
func sanitizeCell(v string) string {
	if v == "" {
		return v
	}
	switch v[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + v
	}
	return v
}

// ExportFilename builds a predictable, safe download name.
//
// Built only from a sanitized short code and a date -- NEVER from Player
// text, a Character name, or a Show title, any of which could carry quotes,
// newlines, or path separators into a Content-Disposition header.
func ExportFilename(shortCode string, at time.Time) string {
	safe := make([]rune, 0, len(shortCode))
	for _, r := range strings.ToLower(strings.TrimSpace(shortCode)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			safe = append(safe, r)
		}
	}
	code := string(safe)
	if code == "" {
		code = "show"
	}
	return "aftercare-" + code + "-" + at.UTC().Format("20060102") + ".csv"
}

// WriteCSV renders the review rows in LONG format: one row per answer
// rather than one column per prompt.
//
// Long format survives a prompt-set change without a column explosion, and
// it makes prompt_set_version data rather than schema -- a Director reading
// an old export can see which wording an answer was written against.
//
// Rows with no answers (state none/skipped/draft) still appear exactly once,
// with an empty prompt and response, so a reader can tell "this Player did
// not answer" apart from "this Player is not in the export".
func WriteCSV(w io.Writer, showRunTitle, showTitle string, rows []ReviewRow) error {
	cw := csv.NewWriter(w)
	header := []string{
		"show_run", "show", "player_handle", "player_name", "character",
		"aftercare_state", "submitted_at", "consecutive_skips",
		"prompt_set_version", "prompt_key", "prompt", "response",
		"door_intention", "tutorial_status",
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, r := range rows {
		submitted := ""
		if r.SubmittedAt != nil {
			submitted = r.SubmittedAt.UTC().Format(time.RFC3339)
		}
		base := []string{
			sanitizeCell(showRunTitle),
			sanitizeCell(showTitle),
			sanitizeCell(r.PlayerHandle),
			sanitizeCell(r.PlayerName),
			sanitizeCell(r.CharacterName),
			r.State,
			submitted,
			itoa(r.ConsecutiveSkips),
		}

		// Emit answers in the authored prompt order, not map order, so two
		// exports of the same data are byte-identical.
		emitted := false
		for _, prompt := range PromptSetV1 {
			answer, ok := r.Responses[prompt.Key]
			if !ok || strings.TrimSpace(answer) == "" {
				continue
			}
			record := append(append([]string{}, base...),
				itoa(r.PromptSetVersion),
				prompt.Key,
				sanitizeCell(prompt.Label),
				sanitizeCell(answer),
				sanitizeCell(r.DoorIntention),
				sanitizeCell(r.TutorialStatus),
			)
			if err := cw.Write(record); err != nil {
				return err
			}
			emitted = true
		}

		if !emitted {
			record := append(append([]string{}, base...),
				"", "", "", "",
				sanitizeCell(r.DoorIntention),
				sanitizeCell(r.TutorialStatus),
			)
			if err := cw.Write(record); err != nil {
				return err
			}
		}
	}

	cw.Flush()
	return cw.Error()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

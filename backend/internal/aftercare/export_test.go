package aftercare

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"
)

// TestSanitizeCellNeutralizesFormulas is a security test, not a formatting
// one. These cells hold text a participant typed, and the file is opened in
// a spreadsheet on a Director's machine.
func TestSanitizeCellNeutralizesFormulas(t *testing.T) {
	dangerous := []string{
		`=cmd|' /C calc'!A0`,
		`+1+1`,
		`-1+1`,
		`@SUM(A1:A9)`,
		"\tleading tab",
		"\rleading cr",
	}
	for _, in := range dangerous {
		got := sanitizeCell(in)
		if !strings.HasPrefix(got, "'") {
			t.Errorf("sanitizeCell(%q) = %q, want a leading quote to make it inert", in, got)
		}
		if !strings.Contains(got, strings.TrimSpace(in)) && !strings.Contains(got, in) {
			t.Errorf("sanitizeCell(%q) = %q, the original text must remain readable", in, got)
		}
	}

	safe := []string{"The gate opening.", "Kessa, mostly", "", "1 + 1 is fine mid-string"}
	for _, in := range safe {
		if got := sanitizeCell(in); got != in {
			t.Errorf("sanitizeCell(%q) = %q, ordinary prose must be untouched", in, got)
		}
	}
}

// TestWriteCSVRoundTripsAwkwardText proves the encoder handles the things
// Players actually type: quotes, commas, and newlines inside an answer.
func TestWriteCSVRoundTripsAwkwardText(t *testing.T) {
	submitted := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	rows := []ReviewRow{{
		UserID: "u1", PlayerHandle: "trang", PlayerName: "Trang",
		CharacterCardID: "c1", CharacterName: "Trang",
		State: "submitted", SubmittedAt: &submitted, PromptSetVersion: 1,
		Responses: map[string]string{
			"favorite_moments": "The gate, \"finally\" opening.\nAlso: Kessa, who I liked.",
		},
		DoorIntention: "I study the hinges, then lift.",
	}}

	var buf bytes.Buffer
	if err := WriteCSV(&buf, "K75 Run", "K75 Show", rows); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}

	records, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("the export must be parseable CSV: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected a header and one answer row, got %d", len(records))
	}
	got := records[1]
	found := false
	for _, cell := range got {
		if strings.Contains(cell, "finally") && strings.Contains(cell, "Kessa, who I liked") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the answer must survive quoting and newlines intact: %v", got)
	}
}

// TestWriteCSVEmitsUnansweredPlayersExactlyOnce guards S9.5: a Director must
// be able to tell "did not answer" apart from "not in the export".
func TestWriteCSVEmitsUnansweredPlayersExactlyOnce(t *testing.T) {
	rows := []ReviewRow{
		{PlayerHandle: "skipper", CharacterName: "Mara", State: "skipped"},
		{PlayerHandle: "quiet", CharacterName: "Kai", State: "none"},
		{PlayerHandle: "drafting", CharacterName: "Ro", State: "draft"},
	}
	var buf bytes.Buffer
	if err := WriteCSV(&buf, "Run", "Show", rows); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	records, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(records) != 4 {
		t.Fatalf("expected header + 3 rows, got %d", len(records))
	}
	for i, want := range []string{"skipped", "none", "draft"} {
		if records[i+1][5] != want {
			t.Errorf("row %d state = %q, want %q", i, records[i+1][5], want)
		}
	}
}

// TestWriteCSVIsDeterministic: two exports of the same data must be
// byte-identical, so a Director diffing them sees real changes only.
func TestWriteCSVIsDeterministic(t *testing.T) {
	rows := []ReviewRow{{
		PlayerHandle: "trang", CharacterName: "Trang", State: "submitted", PromptSetVersion: 1,
		Responses: map[string]string{
			"next_session":     "More Ra.",
			"favorite_moments": "The gate.",
			"who_surprised":    "Kessa.",
		},
	}}
	var first bytes.Buffer
	if err := WriteCSV(&first, "Run", "Show", rows); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	for i := 0; i < 50; i++ {
		var next bytes.Buffer
		if err := WriteCSV(&next, "Run", "Show", rows); err != nil {
			t.Fatalf("WriteCSV: %v", err)
		}
		if next.String() != first.String() {
			t.Fatalf("export drifted on iteration %d -- responses must be emitted in authored prompt order, not map order", i)
		}
	}
	// And the authored order is the one used.
	records, _ := csv.NewReader(strings.NewReader(first.String())).ReadAll()
	if len(records) != 4 {
		t.Fatalf("expected header + 3 answers, got %d", len(records))
	}
	for i, want := range []string{"favorite_moments", "who_surprised", "next_session"} {
		if records[i+1][9] != want {
			t.Errorf("answer %d = %q, want %q", i, records[i+1][9], want)
		}
	}
}

func TestExportFilenameIsSafe(t *testing.T) {
	at := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	cases := map[string]string{
		"ABC123":            "aftercare-abc123-20260727.csv",
		"a b/c":             "aftercare-abc-20260727.csv",
		`"; rm -rf /`:       "aftercare-rm-rf-20260727.csv",
		"":                  "aftercare-show-20260727.csv",
		"../../etc/passwd":  "aftercare-etcpasswd-20260727.csv",
	}
	for in, want := range cases {
		if got := ExportFilename(in, at); got != want {
			t.Errorf("ExportFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

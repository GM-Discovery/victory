package characters

import "testing"

// TestSaveCharacterJournalValidatesBeforeVenueSessionLookup covers the
// venueID/sessionID parameters added for Kernel 59 (character_journals.venue_id
// and .session_id existed since migration 031 but were never populated by any
// call site until now). These validation errors are returned before any
// database access, so a nil pool is safe here.
func TestSaveCharacterJournalValidatesBeforeVenueSessionLookup(t *testing.T) {
	if _, err := SaveCharacterJournal(nil, nil, "", "body", "private", "venue-1", "session-1"); err == nil || err.Error() != "not_authenticated" {
		t.Fatalf("err = %v, want not_authenticated", err)
	}
	if _, err := SaveCharacterJournal(nil, nil, "user-1", "", "private", "venue-1", "session-1"); err == nil || err.Error() != "journal_body_required" {
		t.Fatalf("err = %v, want journal_body_required", err)
	}
}

package identity

import "testing"

func TestEmailFormatPattern(t *testing.T) {
	valid := []string{"a@b.co", "person@example.com", "first.last+tag@sub.example.org"}
	for _, v := range valid {
		if !emailFormatPattern.MatchString(v) {
			t.Errorf("expected %q to be valid", v)
		}
	}

	invalid := []string{"", "not-an-email", "missing-at.com", "double@@at.com", "no-domain@", "@no-local.com", "spaces in@email.com"}
	for _, v := range invalid {
		if emailFormatPattern.MatchString(v) {
			t.Errorf("expected %q to be invalid", v)
		}
	}
}

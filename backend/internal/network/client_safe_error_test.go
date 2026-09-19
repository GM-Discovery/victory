package network

import (
	"errors"
	"testing"
)

// TestClientSafeError is the Kernel 101 (101-07) closure for the raw
// internal-error leak: ~24 call sites across this package used to send
// err.Error() straight to the client, safe for the many places that
// construct an intentional stable reason code but a real risk for
// anything further downstream that isn't -- concretely reachable via a
// raw, unwrapped database error (see rollaudience.resolveSessionShowID,
// which returns a bare pgx error on any failure other than ErrNoRows).
func TestClientSafeError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil error", nil, ""},
		{"stable reason code passes through unchanged", errors.New("skill_not_found"), "skill_not_found"},
		{"stable reason code with digits", errors.New("mode2_denied"), "mode2_denied"},
		{"raw pgx-shaped error is masked", errors.New(`ERROR: relation "foo" does not exist (SQLSTATE 42P01)`), "internal_error"},
		{"raw network error is masked", errors.New("dial tcp: connect: connection refused"), "internal_error"},
		{"context deadline error is masked", errors.New("context deadline exceeded"), "internal_error"},
		{"uppercase-leading text is masked", errors.New("Something went wrong"), "internal_error"},
		{"empty string is masked", errors.New(""), "internal_error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := clientSafeError(tc.err); got != tc.want {
				t.Fatalf("clientSafeError(%q) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

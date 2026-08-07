package storyboards

// Deterministic serialized slugs for structural objects (Kernel 81A):
// columns and bands (board-scoped), rows (band-scoped). UUIDs remain the
// true stable identity everywhere in this package -- slug exists purely
// as a human-readable, collision-free serialized key for export and any
// future integration, never as a lookup key for authority or mutation.
//
// A slug is assigned exactly once, at creation, and never rewritten by
// any later rename/reorder -- RenameColumn/UpdateBandLabel/RenameRow only
// ever touch Title/Label/Description, never Slug. This is deliberate: a
// stable serialized key that silently changed on every label edit would
// defeat the entire point of having one.
//
// Collision behavior: "Scene" -> "scene", a second "Scene" -> "scene-2",
// a third -> "scene-3". If "scene-2" already exists independently (e.g.
// an object was literally titled "Scene 2"), a fourth "Scene" skips it
// and lands on "scene-3" -- allocateUniqueSlug never collides, it always
// advances to the next open integer.
//
// Race safety: two concurrent creations racing for the same slug cannot
// both win, because the DB carries the real unique constraint (migration
// 094's partial unique indexes) -- this file's retry loop is a liveness
// mechanism (so a race produces a retry, not a user-facing error), not
// the correctness mechanism. The unique index is.

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrSlugAllocationExhausted = errors.New("slug_allocation_exhausted")

// queryRower is satisfied by both *pgxpool.Pool and pgx.Tx, so
// allocateUniqueSlug can run its exists-check either as a standalone call
// (AddColumn/AddBand/AddRow) or inside an already-open transaction
// (CreateBoard's atomic default-column/band/row creation) without two
// copies of the same logic.
type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// maxSlugSuffixScan bounds how many numeric suffixes (scene, scene-2, ...)
// a single allocation attempt will scan before giving up -- generous
// enough that no realistic amount of duplicate labeling ever approaches
// it, while still bounding the loop.
const maxSlugSuffixScan = 10000

// maxSlugRaceRetries bounds how many times allocateUniqueSlug restarts
// its whole scan after losing a race to a concurrent insert/update on the
// exact candidate it just picked. A lost race is rare and self-limiting
// (each retry only has to out-run the specific burst of concurrent
// callers that were already in flight), so this is deliberately much
// smaller than maxSlugSuffixScan.
const maxSlugRaceRetries = 50

var slugNonAlnumRun = regexp.MustCompile(`[^a-z0-9]+`)

// SlugifyLabel converts a human-facing label into the deterministic base
// slug candidate: lowercased, every run of non-alphanumeric characters
// collapsed to a single hyphen, leading/trailing hyphens trimmed. The
// same label always produces the same base candidate (deterministic), and
// slugifying an already-slug-shaped string is a no-op (idempotent) --
// both required by spec. An empty/all-punctuation label falls back to
// "untitled" rather than producing an empty string, matching this
// package's existing "(untitled)" convention for a blank card title.
func SlugifyLabel(label string) string {
	s := strings.ToLower(strings.TrimSpace(label))
	s = slugNonAlnumRun.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "untitled"
	}
	return s
}

// allocateUniqueSlug picks the first available deterministic slug for
// label within one scope (existsSQL must be a `SELECT EXISTS(SELECT 1
// FROM ... WHERE <scope column> = $1 AND slug = $2)` query), then calls
// insert with that candidate. insert performs the actual INSERT/UPDATE
// that consumes the slug and returns the affected row's id.
//
// If insert fails with a Postgres unique-violation (23505 -- the
// candidate was taken by a concurrent caller between the exists-check and
// the write), the whole scan restarts from "base" rather than resuming
// from the lost candidate, since a concurrent burst can take more than
// one candidate between checks.
func allocateUniqueSlug(
	ctx context.Context,
	q queryRower,
	existsSQL string,
	scopeID string,
	label string,
	insert func(ctx context.Context, slug string) (string, error),
) (string, error) {
	base := SlugifyLabel(label)

	for race := 0; race < maxSlugRaceRetries; race++ {
		candidate := base
		for n := 1; n <= maxSlugSuffixScan; n++ {
			if n > 1 {
				candidate = fmt.Sprintf("%s-%d", base, n)
			}

			var exists bool
			if err := q.QueryRow(ctx, existsSQL, scopeID, candidate).Scan(&exists); err != nil {
				return "", err
			}
			if exists {
				continue
			}

			id, err := insert(ctx, candidate)
			if err == nil {
				return id, nil
			}
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "slug") {
				// Lost the race for this exact candidate -- another caller's
				// concurrent write committed first. Restart the scan: other
				// candidates that looked free a moment ago may now also be
				// taken by the same concurrent burst.
				//
				// The constraint-name check matters: storyboard_columns/
				// storyboard_bands/storyboard_rows also carry a pre-existing,
				// unrelated UNIQUE(scope, sort_order) constraint (Kernel 80),
				// whose own insert-time collisions under concurrent callers
				// are a real, separate, out-of-scope race this kernel does
				// not fix (see operator-notes.md). Retrying with a
				// *different slug* would never resolve a sort_order
				// collision -- it would just loop until
				// ErrSlugAllocationExhausted, misreporting an unrelated bug
				// as a slug failure. Only a genuine slug-constraint
				// violation is retried here; anything else propagates as
				// the real error it is.
				break
			}
			return "", err
		}
	}
	return "", ErrSlugAllocationExhausted
}

const columnSlugExistsSQL = `SELECT EXISTS(SELECT 1 FROM storyboard_columns WHERE storyboard_id = $1 AND slug = $2)`
const bandSlugExistsSQL = `SELECT EXISTS(SELECT 1 FROM storyboard_bands WHERE storyboard_id = $1 AND slug = $2)`
const rowSlugExistsSQL = `SELECT EXISTS(SELECT 1 FROM storyboard_rows WHERE band_id = $1 AND slug = $2)`

// referenceFieldSlugExistsSQL (Kernel 82): Reference Panel field slugs are
// board-scoped, same as columns/bands.
const referenceFieldSlugExistsSQL = `SELECT EXISTS(SELECT 1 FROM storyboard_reference_fields WHERE storyboard_id = $1 AND slug = $2)`

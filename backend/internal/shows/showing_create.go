package shows

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateShowingInput is the Kernel 92 "New Showing" form shape: a Director
// picks a parent Show Run, a required Nickname (<=50 chars, not required to
// be globally unique -- the short_code/timestamp pair already provides
// uniqueness where needed per kernel doc §28), and a scheduled date/time.
// There is no separate Title field in the form -- Title is derived from
// Nickname so every other Show-Run-scoped view that reads Title keeps
// working unmodified.
type CreateShowingInput struct {
	ShowRunID        string
	Nickname         string
	ScheduledStartAt time.Time
	ScheduledEndAt   *time.Time
}

const nicknameMaxLength = 50

// CreateShowing wraps CreateShow with the Kernel 92 scheduling-specific
// validation and defaults: nickname is required and capped at 50 characters
// (kernel doc §4), the timestamp is generated/stored canonically rather
// than typed by the Director (kernel doc §5), and the new Showing starts
// life as "scheduled" rather than the pre-Kernel-92 "draft" default --
// scheduling alone never starts anything live (kernel doc §6).
func CreateShowing(ctx context.Context, pool *pgxpool.Pool, actorUserID string, in CreateShowingInput) (Show, error) {
	nickname := strings.TrimSpace(in.Nickname)
	if nickname == "" {
		return Show{}, errors.New("nickname_required")
	}
	if len([]rune(nickname)) > nicknameMaxLength {
		return Show{}, errors.New("nickname_too_long")
	}
	if in.ScheduledStartAt.IsZero() {
		return Show{}, errors.New("scheduled_start_at_required")
	}
	if in.ScheduledEndAt != nil && in.ScheduledEndAt.Before(in.ScheduledStartAt) {
		return Show{}, errors.New("scheduled_end_before_start")
	}

	// Kernel 101 (101-09): a double-click/double-POST here has no
	// idempotency key to dedupe on (HandleShowingsCollection accepts none
	// from the client), and every real submission is genuinely allowed to
	// reuse the same nickname later (nicknames aren't unique -- see this
	// function's own header comment) -- so a naive "same nickname ever"
	// check would wrongly block a legitimate later Showing. Narrowed to
	// the actual double-click shape instead: the same actor, same Show
	// Run, same nickname, created moments ago. Returns the row that
	// already exists rather than erroring, since a double-click's honest
	// intent was "create one Showing," and the caller (a POST response)
	// has no good way to distinguish "your click landed" from "it didn't."
	if existing, found, err := recentDuplicateShowing(ctx, pool, in.ShowRunID, actorUserID, nickname); err != nil {
		return Show{}, err
	} else if found {
		return existing, nil
	}

	slug, err := uniqueShowSlug(ctx, pool, in.ShowRunID, nickname)
	if err != nil {
		return Show{}, err
	}

	status := "scheduled"
	startAt := in.ScheduledStartAt
	return CreateShow(ctx, pool, actorUserID, in.ShowRunID, CreateShowInput{
		Title:            nickname,
		Slug:             slug,
		Nickname:         nickname,
		Status:           &status,
		ScheduledStartAt: &startAt,
		ScheduledEndAt:   in.ScheduledEndAt,
	})
}

// recentDuplicateShowingWindow bounds how long a duplicate-click guard
// looks back -- long enough to cover a genuine double-click or a retried
// POST after a slow/flaky response, short enough that a Director
// deliberately creating a second, identically-named Showing minutes later
// is never silently merged into the first one.
const recentDuplicateShowingWindow = 15 * time.Second

// recentDuplicateShowing is Kernel 101's 101-09 fix: finds a Showing this
// same actor already created for this Show Run, under this exact
// nickname, within the last recentDuplicateShowingWindow -- the narrow
// shape a double-click/double-POST actually produces, not "this nickname
// was ever used before" (which legitimately happens across real,
// deliberate Showings and must never block one).
func recentDuplicateShowing(ctx context.Context, pool *pgxpool.Pool, showRunID, actorUserID, nickname string) (Show, bool, error) {
	row := pool.QueryRow(ctx, `
		SELECT `+showColumns+`
		FROM shows
		WHERE show_run_id = $1
		  AND created_by_user_id = $2
		  AND nickname = $3
		  AND created_at > NOW() - $4::interval
		ORDER BY created_at DESC
		LIMIT 1
	`, showRunID, actorUserID, nickname, recentDuplicateShowingWindow.String())
	s, err := scanShow(row)
	if err != nil {
		// scanShow itself translates pgx.ErrNoRows into "show_not_found"
		// (see its own definition) -- every other caller in this package
		// treats that as a genuine error, but here it's the expected,
		// ordinary case (no recent duplicate exists), not a failure.
		if err.Error() == "show_not_found" {
			return Show{}, false, nil
		}
		return Show{}, false, err
	}
	return s, true, nil
}

// uniqueShowSlug derives a URL-safe slug from the nickname and appends a
// short random suffix (reusing the same alphabet/generation approach as
// short codes) to satisfy shows' UNIQUE(show_run_id, slug) constraint
// without forcing the Director to think about uniqueness (kernel doc §28).
func uniqueShowSlug(ctx context.Context, pool *pgxpool.Pool, showRunID, nickname string) (string, error) {
	base := slugifyNickname(nickname)
	if base == "" {
		base = "showing"
	}
	for attempt := 0; attempt < shortCodeMaxAttempts; attempt++ {
		suffix, err := randomShortCode()
		if err != nil {
			return "", err
		}
		candidate := base + "-" + strings.ToLower(suffix)
		var exists bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM shows WHERE show_run_id = $1 AND slug = $2)
		`, showRunID, candidate).Scan(&exists)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", errors.New("slug_generation_failed")
}

func slugifyNickname(nickname string) string {
	lowered := strings.ToLower(strings.TrimSpace(nickname))
	var b strings.Builder
	lastDash := true
	for _, r := range lowered {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

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

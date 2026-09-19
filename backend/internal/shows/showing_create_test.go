package shows

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCreateShowingRequiresNickname(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "showing_nick_producer")
	_, showRunID, _ := insertShowFixture(t, pool, producer)

	_, err := CreateShowing(context.Background(), pool, producer, CreateShowingInput{
		ShowRunID:        showRunID,
		Nickname:         "   ",
		ScheduledStartAt: time.Now().UTC().Add(24 * time.Hour),
	})
	if err == nil || err.Error() != "nickname_required" {
		t.Fatalf("expected nickname_required, got %v", err)
	}
}

func TestCreateShowingRejectsNicknameOver50Chars(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "showing_len_producer")
	_, showRunID, _ := insertShowFixture(t, pool, producer)

	tooLong := strings.Repeat("a", 51)
	if _, err := CreateShowing(context.Background(), pool, producer, CreateShowingInput{
		ShowRunID:        showRunID,
		Nickname:         tooLong,
		ScheduledStartAt: time.Now().UTC().Add(24 * time.Hour),
	}); err == nil || err.Error() != "nickname_too_long" {
		t.Fatalf("expected nickname_too_long, got %v", err)
	}

	exactly50 := strings.Repeat("b", 50)
	s, err := CreateShowing(context.Background(), pool, producer, CreateShowingInput{
		ShowRunID:        showRunID,
		Nickname:         exactly50,
		ScheduledStartAt: time.Now().UTC().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("expected exactly-50-char nickname to succeed, got %v", err)
	}
	if s.Nickname != exactly50 {
		t.Fatalf("expected nickname to be stored verbatim, got %q", s.Nickname)
	}
}

func TestCreateShowingSetsScheduledStartAndStatus(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "showing_sched_producer")
	_, showRunID, _ := insertShowFixture(t, pool, producer)

	startAt := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	s, err := CreateShowing(context.Background(), pool, producer, CreateShowingInput{
		ShowRunID:        showRunID,
		Nickname:         "Training Arena Night",
		ScheduledStartAt: startAt,
	})
	if err != nil {
		t.Fatalf("CreateShowing: %v", err)
	}
	if s.Status != "scheduled" {
		t.Fatalf("expected status 'scheduled', got %q", s.Status)
	}
	if s.ScheduledStartAt == nil || !s.ScheduledStartAt.Equal(startAt) {
		t.Fatalf("expected ScheduledStartAt %v, got %v", startAt, s.ScheduledStartAt)
	}
}

func TestCreateShowingRequiresManageAuthority(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "showing_auth_producer")
	_, showRunID, _ := insertShowFixture(t, pool, producer)

	outsider := insertShowsTestUser(t, pool, "showing_auth_outsider")
	if _, err := CreateShowing(context.Background(), pool, outsider, CreateShowingInput{
		ShowRunID:        showRunID,
		Nickname:         "Should Not Work",
		ScheduledStartAt: time.Now().UTC().Add(time.Hour),
	}); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized, got %v", err)
	}
}

// TestCreateShowingDoubleClickReturnsSameRow is the Kernel 101 (101-09)
// closure: a double-click/double-POST with no idempotency key must not
// create two Showing rows for what was honestly one submission.
func TestCreateShowingDoubleClickReturnsSameRow(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "showing_dup_producer")
	_, showRunID, _ := insertShowFixture(t, pool, producer)

	startAt := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	first, err := CreateShowing(context.Background(), pool, producer, CreateShowingInput{
		ShowRunID:        showRunID,
		Nickname:         "Opening Night",
		ScheduledStartAt: startAt,
	})
	if err != nil {
		t.Fatalf("first CreateShowing: %v", err)
	}

	second, err := CreateShowing(context.Background(), pool, producer, CreateShowingInput{
		ShowRunID:        showRunID,
		Nickname:         "Opening Night",
		ScheduledStartAt: startAt,
	})
	if err != nil {
		t.Fatalf("second (double-click) CreateShowing: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected the double-click to return the same Showing (id %s), got a new one (id %s)", first.ID, second.ID)
	}

	var count int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM shows WHERE show_run_id = $1 AND nickname = $2
	`, showRunID, "Opening Night").Scan(&count); err != nil {
		t.Fatalf("count shows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 row in the database after the double-click, got %d", count)
	}
}

// TestCreateShowingAllowsLegitimateReuseOfSameNickname proves the guard is
// narrow: nicknames are explicitly not unique (kernel doc §28), so two
// genuinely distinct Showings created moments apart under the same
// nickname -- the normal shape of, say, a recurring weekly "Game Night" --
// must never be silently merged the way an actual double-click is.
func TestCreateShowingAllowsLegitimateReuseOfSameNickname(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "showing_reuse_producer")
	_, showRunID, _ := insertShowFixture(t, pool, producer)

	first, err := CreateShowing(context.Background(), pool, producer, CreateShowingInput{
		ShowRunID:        showRunID,
		Nickname:         "Game Night",
		ScheduledStartAt: time.Now().UTC().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("first CreateShowing: %v", err)
	}

	// Simulate "moments apart, not a double-click" by pushing the first
	// row's created_at outside the dedup window directly, rather than
	// sleeping the test for real.
	if _, err := pool.Exec(context.Background(), `
		UPDATE shows SET created_at = created_at - interval '1 minute' WHERE id = $1
	`, first.ID); err != nil {
		t.Fatalf("backdate first show: %v", err)
	}

	second, err := CreateShowing(context.Background(), pool, producer, CreateShowingInput{
		ShowRunID:        showRunID,
		Nickname:         "Game Night",
		ScheduledStartAt: time.Now().UTC().Add(24 * time.Hour * 7),
	})
	if err != nil {
		t.Fatalf("second CreateShowing: %v", err)
	}
	if second.ID == first.ID {
		t.Fatal("expected a genuinely new Showing, got the same row back -- the dedup window must not block a legitimate later reuse of the same nickname")
	}
}

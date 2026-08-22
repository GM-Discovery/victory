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

package shows

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// createShowingAt creates a Showing and then backdates its scheduled/created
// timestamps directly via SQL -- CreateShowing itself only accepts a real
// (typically future) ScheduledStartAt, so historical-band fixtures have to
// be backdated after the fact.
func createShowingAt(t *testing.T, pool *pgxpool.Pool, producer, showRunID, nickname string, anchor time.Time, status string) string {
	t.Helper()
	s, err := CreateShowing(context.Background(), pool, producer, CreateShowingInput{
		ShowRunID:        showRunID,
		Nickname:         nickname,
		ScheduledStartAt: time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateShowing(%q): %v", nickname, err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE shows SET scheduled_start_at = $1, status = $2, created_at = $1 WHERE id = $3
	`, anchor, status, s.ID); err != nil {
		t.Fatalf("backdate showing %q: %v", nickname, err)
	}
	return s.ID
}

func TestListShowingsBucketsTodayUpcomingHistorical(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "showing_bucket_producer")
	_, showRunID, _ := insertShowFixture(t, pool, producer)

	now := time.Now().UTC()
	todayID := createShowingAt(t, pool, producer, showRunID, "Today Show", now.Add(2*time.Hour), "scheduled")
	upcomingID := createShowingAt(t, pool, producer, showRunID, "Upcoming Show", now.Add(72*time.Hour), "scheduled")
	recentID := createShowingAt(t, pool, producer, showRunID, "Recent Past", now.Add(-3*24*time.Hour), "completed")
	midID := createShowingAt(t, pool, producer, showRunID, "Mid Past", now.Add(-60*24*time.Hour), "completed")
	oldID := createShowingAt(t, pool, producer, showRunID, "Old Past", now.Add(-400*24*time.Hour), "completed")

	result, err := ListShowingsForViewer(context.Background(), pool, producer, now)
	if err != nil {
		t.Fatalf("ListShowingsForViewer: %v", err)
	}

	if !containsShowingID(result.Today, todayID) {
		t.Fatalf("expected Today to contain %q, got %+v", todayID, result.Today)
	}
	if !containsShowingID(result.Upcoming, upcomingID) {
		t.Fatalf("expected Upcoming to contain %q, got %+v", upcomingID, result.Upcoming)
	}
	if !containsShowingID(result.Historical[BandLessThan7Days], recentID) {
		t.Fatalf("expected %s band to contain %q, got %+v", BandLessThan7Days, recentID, result.Historical[BandLessThan7Days])
	}
	if !containsShowingID(result.Historical[Band31To90Days], midID) {
		t.Fatalf("expected %s band to contain %q, got %+v", Band31To90Days, midID, result.Historical[Band31To90Days])
	}
	if !containsShowingID(result.Historical[Band366DaysTo36Mo], oldID) {
		t.Fatalf("expected %s band to contain %q, got %+v", Band366DaysTo36Mo, oldID, result.Historical[Band366DaysTo36Mo])
	}
}

func TestListShowingsAlphabeticalWithinBand(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "showing_alpha_producer")
	_, showRunID, _ := insertShowFixture(t, pool, producer)

	// Offsets are all >48h out so every one of them falls in "Upcoming"
	// regardless of what time of day the test happens to run at (a <24h
	// offset can land in "Today" or "Upcoming" depending on the wall clock).
	now := time.Now().UTC()
	createShowingAt(t, pool, producer, showRunID, "Zeta Night", now.Add(72*time.Hour), "scheduled")
	createShowingAt(t, pool, producer, showRunID, "Alpha Night", now.Add(96*time.Hour), "scheduled")
	createShowingAt(t, pool, producer, showRunID, "Mid Night", now.Add(120*time.Hour), "scheduled")

	result, err := ListShowingsForViewer(context.Background(), pool, producer, now)
	if err != nil {
		t.Fatalf("ListShowingsForViewer: %v", err)
	}
	// The producer's location ('amurray-family') is shared with every other
	// test fixture in this database, so the raw Upcoming bucket also
	// contains leftover Showings from unrelated tests -- filter down to this
	// test's own Show Run before checking relative order. insertShowFixture
	// also creates one base Show via the pre-Kernel-92 CreateShow (no
	// nickname) under the same Show Run -- exclude it too.
	var names []string
	for _, item := range result.Upcoming {
		if item.ShowRunID == showRunID && item.Nickname != "" {
			names = append(names, item.Nickname)
		}
	}
	if len(names) != 3 || names[0] != "Alpha Night" || names[1] != "Mid Night" || names[2] != "Zeta Night" {
		t.Fatalf("expected alphabetical order [Alpha Night, Mid Night, Zeta Night], got %v", names)
	}
}

func TestListShowingsOnlyIncludesManageableShowRuns(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "showing_scope_producer")
	_, showRunID, _ := insertShowFixture(t, pool, producer)
	now := time.Now().UTC()
	visibleID := createShowingAt(t, pool, producer, showRunID, "Visible Showing", now.Add(5*time.Hour), "scheduled")

	outsider := insertShowsTestUser(t, pool, "showing_scope_outsider")
	result, err := ListShowingsForViewer(context.Background(), pool, outsider, now)
	if err != nil {
		t.Fatalf("ListShowingsForViewer: %v", err)
	}
	if containsShowingID(result.Upcoming, visibleID) || containsShowingID(result.Today, visibleID) {
		t.Fatalf("expected outsider not to see a Showing under a Show Run they cannot manage")
	}
}

func TestListShowingsMarksIsLiveFromSession(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "showing_live_producer")
	_, showRunID, showID := insertShowFixture(t, pool, producer)
	now := time.Now().UTC()

	liveID := createShowingAt(t, pool, producer, showRunID, "Live Showing", now.Add(1*time.Hour), "scheduled")
	quietID := createShowingAt(t, pool, producer, showRunID, "Quiet Showing", now.Add(2*time.Hour), "scheduled")
	_ = showID

	var venueID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM venues WHERE slug = 'catharsis'`).Scan(&venueID); err != nil {
		t.Fatalf("load catharsis venue id: %v", err)
	}
	var sessionID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO sessions (venue_id, status, started_at, show_id) VALUES ($1, 'rehearsal', NOW(), $2) RETURNING id::text
	`, venueID, liveID).Scan(&sessionID); err != nil {
		t.Fatalf("insert live session: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, sessionID) })

	result, err := ListShowingsForViewer(context.Background(), pool, producer, now)
	if err != nil {
		t.Fatalf("ListShowingsForViewer: %v", err)
	}
	// A 1-2 hour offset from "now" can land in either Today or Upcoming
	// depending on the wall-clock time the test happens to run at -- check
	// both buckets rather than assuming one.
	combined := append(append([]ShowingListItem{}, result.Today...), result.Upcoming...)
	if !isLiveInList(combined, liveID) {
		t.Fatalf("expected %q to be marked live", liveID)
	}
	if isLiveInList(combined, quietID) {
		t.Fatalf("expected %q to not be marked live", quietID)
	}
}

func containsShowingID(items []ShowingListItem, id string) bool {
	for _, item := range items {
		if item.ShowID == id {
			return true
		}
	}
	return false
}

func isLiveInList(items []ShowingListItem, id string) bool {
	for _, item := range items {
		if item.ShowID == id {
			return item.IsLive
		}
	}
	return false
}

package storyboards

// Kernel 83 backend proof (spec 11): Group Leader/Current Turn state for
// Storyboards' live venue sessions. Reuses ws_dbtest_test.go's dialWS/
// sendJSON/readJSON and http_dbtest_test.go's authenticatedRequest/
// decodeHTTPResponse helpers -- a "session participant" here is a real
// dialed websocket connection with an active watch_board, exactly as
// coordination.go requires (target/actor must be a current board
// watcher), not a stubbed concept.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/network"
	"victory/backend/internal/sessions"
	"victory/backend/internal/venuecoordination"
)

// dialWSFrom is dialWS plus a distinguishing X-Forwarded-For header. The
// shared package-level wsConnectionLimiter (network/ws.go) buckets by
// client IP, and every httptest.Server connection otherwise reports the
// same loopback address -- without this, a single test file opening more
// than a handful of real WS connections in quick succession would trip a
// rate limit meant for abuse detection, not for a test suite simulating
// many distinct real users (each of whom would have their own IP in
// production). Giving each simulated user their own fake forwarded-for
// value keeps every test's connection budget independent, matching what
// the limiter is actually meant to police.
func dialWSFrom(t *testing.T, server *httptest.Server, pool *pgxpool.Pool, userID string) *websocket.Conn {
	t.Helper()
	raw, _, err := sessions.CreateSession(context.Background(), pool, userID, time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	header := http.Header{}
	header.Set("Cookie", sessions.CookieName+"="+raw)
	header.Set("Origin", "http://"+u.Host)
	header.Set("X-Forwarded-For", "10.83.0."+userID[:min(len(userID), 8)])
	u.Scheme = "ws"
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	return conn
}

func testGrantRole(t *testing.T, pool *pgxpool.Pool, owner, boardID, userID, role string) {
	t.Helper()
	ctx := context.Background()
	var handle string
	if err := pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, userID).Scan(&handle); err != nil {
		t.Fatalf("lookup handle for %s: %v", userID, err)
	}
	if _, err := AddGrant(ctx, pool, owner, boardID, handle, role); err != nil {
		t.Fatalf("grant %s to %s: %v", role, userID, err)
	}
}

func drainForType(t *testing.T, conn *websocket.Conn, wantType string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(4 * time.Second))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read waiting for %q: %v", wantType, err)
		}
		var out map[string]any
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out["type"] == wantType {
			return out
		}
	}
	t.Fatalf("timed out waiting for %q", wantType)
	return nil
}

// expectNoMessageOfType asserts wantType does not arrive on conn within a
// short window -- used to prove an unrelated venue session receives no
// event (spec 11.8).
func expectNoMessageOfType(t *testing.T, conn *websocket.Conn, wantType string) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return // timeout (or close) -- nothing of wantType arrived, as expected
		}
		var out map[string]any
		_ = json.Unmarshal(raw, &out)
		if out["type"] == wantType {
			t.Fatalf("unexpected %q delivered to unrelated session", wantType)
		}
	}
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}

func connectAndWatch(t *testing.T, server *httptest.Server, pool *pgxpool.Pool, userID, boardID string) (*websocket.Conn, map[string]any) {
	t.Helper()
	conn := dialWSFrom(t, server, pool, userID)
	sendJSON(t, conn, map[string]any{"type": "watch_board", "board_id": boardID})
	snap := drainForType(t, conn, "snapshot")
	return conn, snap
}

func snapshotCoordination(t *testing.T, snap map[string]any) map[string]any {
	t.Helper()
	data, _ := snap["data"].(map[string]any)
	coord, _ := data["coordination"].(map[string]any)
	if coord == nil {
		t.Fatalf("snapshot has no coordination field: %+v", snap)
	}
	return coord
}

func postCoordination(t *testing.T, pool *pgxpool.Pool, hub *network.Hub, reg *venuecoordination.Registry, boardID, actorID, targetID string, leader bool) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"target_user_id": targetID})
	path := "/api/storyboards/" + boardID + "/coordination/current-turn"
	var handler http.HandlerFunc
	if leader {
		path = "/api/storyboards/" + boardID + "/coordination/group-leader"
		handler = HandleCoordinationGroupLeader(pool, hub, reg)
	} else {
		handler = HandleCoordinationCurrentTurn(pool, hub, reg)
	}
	req := authenticatedRequest(t, pool, http.MethodPost, path, actorID, body)
	req.SetPathValue("board_id", boardID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// ---------- 11.1 Initialization ----------

func TestCoordinationInitLeaderDefaultsToOwnerWhenFirstWatcher(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "kc_owner")
	board := mustCreateBoard(t, pool, owner, "Coord Init Board")

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	conn, snap := connectAndWatch(t, server, pool, owner, board.ID)
	defer conn.Close()

	coord := snapshotCoordination(t, snap)
	if coord["active"] != true {
		t.Fatalf("expected active session, got %+v", coord)
	}
	if coord["group_leader_user_id"] != owner {
		t.Fatalf("expected leader = owner, got %+v", coord)
	}
	if _, ok := coord["current_turn_user_id"]; ok {
		t.Fatalf("expected current_turn_user_id unset, got %+v", coord)
	}
}

func TestCoordinationInitLeaderUnsetWhenOwnerNotFirstWatcher(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "kc_owner2")
	crew := insertTestUser(t, pool, "kc_crew2")
	board := mustCreateBoard(t, pool, owner, "Coord NonOwnerFirst Board")
	testGrantRole(t, pool, owner, board.ID, crew, TierCrew)

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	crewConn, crewSnap := connectAndWatch(t, server, pool, crew, board.ID)
	defer crewConn.Close()
	coord := snapshotCoordination(t, crewSnap)
	if _, ok := coord["group_leader_user_id"]; ok {
		t.Fatalf("expected leader unset when owner absent at session start, got %+v", coord)
	}

	// Owner joining afterward must not retroactively seize leadership
	// (spec 5.2) -- the session already started with crew as first watcher.
	ownerConn, ownerSnap := connectAndWatch(t, server, pool, owner, board.ID)
	defer ownerConn.Close()
	coord2 := snapshotCoordination(t, ownerSnap)
	if _, ok := coord2["group_leader_user_id"]; ok {
		t.Fatalf("owner joining later must not seize leadership, got %+v", coord2)
	}
}

func TestCoordinationInactiveSessionHasNoState(t *testing.T) {
	reg := venuecoordination.NewRegistry()
	if _, active := reg.Get("nonexistent-board"); active {
		t.Fatalf("expected no active session before any watcher connects")
	}
}

// ---------- 11.2 / 11.3 Authority ----------

func TestCoordinationDirectorAssignsLeader(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "kc_owner3")
	director := insertTestUser(t, pool, "kc_director3")
	board := mustCreateBoard(t, pool, owner, "Coord Director Assign Board")
	testGrantRole(t, pool, owner, board.ID, director, TierDirector)

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	ownerConn, _ := connectAndWatch(t, server, pool, owner, board.ID)
	defer ownerConn.Close()
	dirConn, _ := connectAndWatch(t, server, pool, director, board.ID)
	defer dirConn.Close()
	drainForType(t, ownerConn, EventPresenceChanged) // director joining notifies owner

	rec := postCoordination(t, pool, hub, reg, board.ID, director, director, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("director assign leader status = %d body=%s", rec.Code, rec.Body.String())
	}
	out := decodeHTTPResponse(t, rec)
	data, _ := out.Data.(map[string]any)
	coord, _ := data["coordination"].(map[string]any)
	if coord["group_leader_user_id"] != director {
		t.Fatalf("expected leader director, got %+v", coord)
	}
}

func TestCoordinationLeaderPassesLeadershipWithoutBeingDirector(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "kc_owner4")
	crewLeader := insertTestUser(t, pool, "kc_crewleader4")
	castTarget := insertTestUser(t, pool, "kc_casttarget4")
	board := mustCreateBoard(t, pool, owner, "Coord Leader Pass Board")
	testGrantRole(t, pool, owner, board.ID, crewLeader, TierCrew)
	testGrantRole(t, pool, owner, board.ID, castTarget, TierCast)

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	ownerConn, _ := connectAndWatch(t, server, pool, owner, board.ID)
	defer ownerConn.Close()
	crewConn, _ := connectAndWatch(t, server, pool, crewLeader, board.ID)
	defer crewConn.Close()
	castConn, _ := connectAndWatch(t, server, pool, castTarget, board.ID)
	defer castConn.Close()

	// Owner (Director+) makes the Crew user leader.
	rec := postCoordination(t, pool, hub, reg, board.ID, owner, crewLeader, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("owner assign leader status = %d body=%s", rec.Code, rec.Body.String())
	}

	// The new (Crew-tier) leader, despite not being Director+, may pass
	// leadership onward purely because they hold it.
	rec2 := postCoordination(t, pool, hub, reg, board.ID, crewLeader, castTarget, true)
	if rec2.Code != http.StatusOK {
		t.Fatalf("crew leader pass leadership status = %d body=%s", rec2.Code, rec2.Body.String())
	}
	out := decodeHTTPResponse(t, rec2)
	data, _ := out.Data.(map[string]any)
	coord, _ := data["coordination"].(map[string]any)
	if coord["group_leader_user_id"] != castTarget {
		t.Fatalf("expected leader castTarget, got %+v", coord)
	}
}

func TestCoordinationUnauthorizedTiersCannotSeizeLeader(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "kc_owner5")
	crew := insertTestUser(t, pool, "kc_crew5")
	cast := insertTestUser(t, pool, "kc_cast5")
	audience := insertTestUser(t, pool, "kc_audience5")
	board := mustCreateBoard(t, pool, owner, "Coord Seize Board")
	testGrantRole(t, pool, owner, board.ID, crew, TierCrew)
	testGrantRole(t, pool, owner, board.ID, cast, TierCast)
	testGrantRole(t, pool, owner, board.ID, audience, TierAudience)

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	ownerConn, _ := connectAndWatch(t, server, pool, owner, board.ID)
	defer ownerConn.Close()
	crewConn, _ := connectAndWatch(t, server, pool, crew, board.ID)
	defer crewConn.Close()
	castConn, _ := connectAndWatch(t, server, pool, cast, board.ID)
	defer castConn.Close()
	audConn, _ := connectAndWatch(t, server, pool, audience, board.ID)
	defer audConn.Close()

	for _, seizer := range []string{crew, cast, audience} {
		rec := postCoordination(t, pool, hub, reg, board.ID, seizer, seizer, true)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("seizer %s status = %d, want 403, body=%s", seizer, rec.Code, rec.Body.String())
		}
	}
}

func TestCoordinationTargetMustBelongToSession(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "kc_owner6")
	notWatching := insertTestUser(t, pool, "kc_notwatching6")
	board := mustCreateBoard(t, pool, owner, "Coord Target Membership Board")
	testGrantRole(t, pool, owner, board.ID, notWatching, TierCrew)

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	ownerConn, _ := connectAndWatch(t, server, pool, owner, board.ID)
	defer ownerConn.Close()

	rec := postCoordination(t, pool, hub, reg, board.ID, owner, notWatching, true)
	if rec.Code == http.StatusOK {
		t.Fatalf("expected rejection assigning a non-watching user, got 200: %s", rec.Body.String())
	}
	out := decodeHTTPResponse(t, rec)
	data, _ := out.Data.(map[string]any)
	if data["error"] != "coordination_target_not_in_session" {
		t.Fatalf("expected coordination_target_not_in_session, got %+v", out)
	}
}

func TestCoordinationCrossBoardTargetRejected(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "kc_owner7")
	boardA := mustCreateBoard(t, pool, owner, "Coord Cross A")
	boardB := mustCreateBoard(t, pool, owner, "Coord Cross B")
	otherUser := insertTestUser(t, pool, "kc_other7")
	testGrantRole(t, pool, owner, boardB.ID, otherUser, TierCrew)

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	ownerConnA, _ := connectAndWatch(t, server, pool, owner, boardA.ID)
	defer ownerConnA.Close()
	otherConnB, _ := connectAndWatch(t, server, pool, otherUser, boardB.ID)
	defer otherConnB.Close()

	// otherUser is a real, currently-connected watcher, just of a
	// different board -- must still be rejected as a target on boardA.
	rec := postCoordination(t, pool, hub, reg, boardA.ID, owner, otherUser, true)
	out := decodeHTTPResponse(t, rec)
	data, _ := out.Data.(map[string]any)
	if data["error"] != "coordination_target_not_in_session" {
		t.Fatalf("expected cross-board target rejected, got %+v", out)
	}
}

func TestCoordinationCurrentTurnAuthority(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "kc_owner8")
	groupLeader := insertTestUser(t, pool, "kc_leader8")
	turnHolder := insertTestUser(t, pool, "kc_turn8")
	bystanderCrew := insertTestUser(t, pool, "kc_bystander8")
	board := mustCreateBoard(t, pool, owner, "Coord Turn Authority Board")
	testGrantRole(t, pool, owner, board.ID, groupLeader, TierCrew)
	testGrantRole(t, pool, owner, board.ID, turnHolder, TierCast)
	testGrantRole(t, pool, owner, board.ID, bystanderCrew, TierCrew)

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	ownerConn, _ := connectAndWatch(t, server, pool, owner, board.ID)
	defer ownerConn.Close()
	leaderConn, _ := connectAndWatch(t, server, pool, groupLeader, board.ID)
	defer leaderConn.Close()
	turnConn, _ := connectAndWatch(t, server, pool, turnHolder, board.ID)
	defer turnConn.Close()
	bystanderConn, _ := connectAndWatch(t, server, pool, bystanderCrew, board.ID)
	defer bystanderConn.Close()

	// Director+ (owner) gives the turn.
	rec := postCoordination(t, pool, hub, reg, board.ID, owner, turnHolder, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("owner give turn status = %d body=%s", rec.Code, rec.Body.String())
	}

	// Make groupLeader the Group Leader (via owner), then prove Group
	// Leader (Crew-tier, not Director+) may also give the turn.
	if rec := postCoordination(t, pool, hub, reg, board.ID, owner, groupLeader, true); rec.Code != http.StatusOK {
		t.Fatalf("owner assign leader status = %d body=%s", rec.Code, rec.Body.String())
	}
	rec2 := postCoordination(t, pool, hub, reg, board.ID, groupLeader, bystanderCrew, false)
	if rec2.Code != http.StatusOK {
		t.Fatalf("group leader give turn status = %d body=%s", rec2.Code, rec2.Body.String())
	}

	// Current turn holder passes the turn onward themselves.
	rec3 := postCoordination(t, pool, hub, reg, board.ID, bystanderCrew, turnHolder, false)
	if rec3.Code != http.StatusOK {
		t.Fatalf("turn holder pass turn status = %d body=%s", rec3.Code, rec3.Body.String())
	}

	// An unrelated Crew/Cast/Audience user (bystanderCrew no longer holds
	// the turn after the previous step) cannot assign turn arbitrarily.
	rec4 := postCoordination(t, pool, hub, reg, board.ID, bystanderCrew, bystanderCrew, false)
	if rec4.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unrelated crew assigning turn, got %d body=%s", rec4.Code, rec4.Body.String())
	}
}

// ---------- 11.4 No participant order ----------

func TestCoordinationDisconnectDoesNotAutoAdvanceTurn(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "kc_owner9")
	alice := insertTestUser(t, pool, "kc_alice9")
	bob := insertTestUser(t, pool, "kc_bob9")
	board := mustCreateBoard(t, pool, owner, "Coord No Order Board")
	testGrantRole(t, pool, owner, board.ID, alice, TierCrew)
	testGrantRole(t, pool, owner, board.ID, bob, TierCrew)

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	ownerConn, _ := connectAndWatch(t, server, pool, owner, board.ID)
	defer ownerConn.Close()
	aliceConn, _ := connectAndWatch(t, server, pool, alice, board.ID)
	bobConn, _ := connectAndWatch(t, server, pool, bob, board.ID)
	defer bobConn.Close()

	rec := postCoordination(t, pool, hub, reg, board.ID, owner, alice, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("assign turn to alice: %d %s", rec.Code, rec.Body.String())
	}

	aliceConn.Close()
	waitFor(t, 2*time.Second, func() bool {
		s, active := reg.Get(StoryboardVenueSessionID(board.ID))
		return active && s.CurrentTurnUserID == alice
	})

	s, active := reg.Get(StoryboardVenueSessionID(board.ID))
	if !active || s.CurrentTurnUserID != alice {
		t.Fatalf("expected turn to remain alice after disconnect, got %+v active=%v", s, active)
	}
	if s.CurrentTurnUserID == bob {
		t.Fatalf("bob must not be auto-assigned the turn")
	}
}

// ---------- 11.5 Disconnect ----------

func TestCoordinationLeaderDisconnectLeavesAssignmentIntact(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "kc_owner10")
	other := insertTestUser(t, pool, "kc_other10")
	board := mustCreateBoard(t, pool, owner, "Coord Leader Disconnect Board")
	testGrantRole(t, pool, owner, board.ID, other, TierCrew)

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	ownerConn, _ := connectAndWatch(t, server, pool, owner, board.ID)
	otherConn, _ := connectAndWatch(t, server, pool, other, board.ID)
	defer otherConn.Close()

	// Owner is already Group Leader from session start (spec 1.3).
	ownerConn.Close()
	waitFor(t, 2*time.Second, func() bool {
		s, active := reg.Get(StoryboardVenueSessionID(board.ID))
		return active && s.GroupLeaderUserID == owner
	})

	s, active := reg.Get(StoryboardVenueSessionID(board.ID))
	if !active || s.GroupLeaderUserID != owner {
		t.Fatalf("expected leader assignment to survive owner disconnect (other watcher still present), got %+v active=%v", s, active)
	}

	// Reconnect during the same live session -- assignment must still be there.
	reconnConn, reconnSnap := connectAndWatch(t, server, pool, owner, board.ID)
	defer reconnConn.Close()
	coord := snapshotCoordination(t, reconnSnap)
	if coord["group_leader_user_id"] != owner {
		t.Fatalf("expected leader still owner on reconnect, got %+v", coord)
	}
}

func TestCoordinationCurrentTurnHolderDisconnectLeavesAssignmentIntact(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "kc_owner11")
	holder := insertTestUser(t, pool, "kc_holder11")
	board := mustCreateBoard(t, pool, owner, "Coord Turn Disconnect Board")
	testGrantRole(t, pool, owner, board.ID, holder, TierCrew)

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	ownerConn, _ := connectAndWatch(t, server, pool, owner, board.ID)
	defer ownerConn.Close()
	holderConn, _ := connectAndWatch(t, server, pool, holder, board.ID)

	if rec := postCoordination(t, pool, hub, reg, board.ID, owner, holder, false); rec.Code != http.StatusOK {
		t.Fatalf("assign turn: %d %s", rec.Code, rec.Body.String())
	}

	holderConn.Close()
	waitFor(t, 2*time.Second, func() bool {
		s, active := reg.Get(StoryboardVenueSessionID(board.ID))
		return active && s.CurrentTurnUserID == holder
	})
	s, active := reg.Get(StoryboardVenueSessionID(board.ID))
	if !active || s.CurrentTurnUserID != holder {
		t.Fatalf("expected turn assignment to survive disconnect, got %+v active=%v", s, active)
	}
}

// ---------- 11.6 Session end ----------

func TestCoordinationSessionEndClearsStateAndLaterSessionStartsFresh(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "kc_owner12")
	other := insertTestUser(t, pool, "kc_other12")
	board := mustCreateBoard(t, pool, owner, "Coord Session End Board")
	testGrantRole(t, pool, owner, board.ID, other, TierCrew)

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	ownerConn, _ := connectAndWatch(t, server, pool, owner, board.ID)
	otherConn, _ := connectAndWatch(t, server, pool, other, board.ID)

	if rec := postCoordination(t, pool, hub, reg, board.ID, owner, other, false); rec.Code != http.StatusOK {
		t.Fatalf("assign turn: %d %s", rec.Code, rec.Body.String())
	}

	ownerConn.Close()
	otherConn.Close()
	waitFor(t, 2*time.Second, func() bool {
		_, active := reg.Get(StoryboardVenueSessionID(board.ID))
		return !active
	})

	// A new later session (other connects first this time) must start
	// completely fresh -- no restored leader, no restored turn.
	freshConn, freshSnap := connectAndWatch(t, server, pool, other, board.ID)
	defer freshConn.Close()
	coord := snapshotCoordination(t, freshSnap)
	if _, ok := coord["group_leader_user_id"]; ok {
		t.Fatalf("expected fresh session leader unset (other is not owner), got %+v", coord)
	}
	if _, ok := coord["current_turn_user_id"]; ok {
		t.Fatalf("expected fresh session turn unset, got %+v", coord)
	}
}

// ---------- 11.7 Permission independence ----------

func TestCoordinationAssignmentDoesNotTouchGrantsOrPermissions(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "kc_owner13")
	crew := insertTestUser(t, pool, "kc_crew13")
	board := mustCreateBoard(t, pool, owner, "Coord Permission Independence Board")
	testGrantRole(t, pool, owner, board.ID, crew, TierCrew)

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	ownerConn, _ := connectAndWatch(t, server, pool, owner, board.ID)
	defer ownerConn.Close()
	crewConn, _ := connectAndWatch(t, server, pool, crew, board.ID)
	defer crewConn.Close()

	var grantsBefore int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM storyboard_grants WHERE storyboard_id = $1`, board.ID).Scan(&grantsBefore)

	if rec := postCoordination(t, pool, hub, reg, board.ID, owner, crew, true); rec.Code != http.StatusOK {
		t.Fatalf("assign leader: %d %s", rec.Code, rec.Body.String())
	}
	if rec := postCoordination(t, pool, hub, reg, board.ID, crew, crew, false); rec.Code != http.StatusOK {
		t.Fatalf("crew (now leader) self-assign turn: %d %s", rec.Code, rec.Body.String())
	}

	var grantsAfter int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM storyboard_grants WHERE storyboard_id = $1`, board.ID).Scan(&grantsAfter)
	if grantsBefore != grantsAfter {
		t.Fatalf("expected grant row count unchanged, before=%d after=%d", grantsBefore, grantsAfter)
	}

	// Crew tier is unchanged by holding Group Leader/Current Turn --
	// still cannot edit board structure (Director+ only).
	board2, err := LoadBoard(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("reload board: %v", err)
	}
	canStructure, err := CanEditStructure(ctx, pool, crew, board2)
	if err != nil {
		t.Fatalf("CanEditStructure: %v", err)
	}
	if canStructure {
		t.Fatalf("expected Crew-tier Group Leader/Current Turn holder to still lack structural authority")
	}
}

// ---------- 11.8 Live sync ----------

func TestCoordinationChangeBroadcastsLiveAndIsIdempotent(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "kc_owner14")
	other := insertTestUser(t, pool, "kc_other14")
	board := mustCreateBoard(t, pool, owner, "Coord Live Sync Board")
	testGrantRole(t, pool, owner, board.ID, other, TierCrew)

	// An entirely unrelated board/session must not receive this board's events.
	unrelatedOwner := insertTestUser(t, pool, "kc_unrelated14")
	unrelatedBoard := mustCreateBoard(t, pool, unrelatedOwner, "Coord Unrelated Board")

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	ownerConn, _ := connectAndWatch(t, server, pool, owner, board.ID)
	defer ownerConn.Close()
	otherConn, _ := connectAndWatch(t, server, pool, other, board.ID)
	defer otherConn.Close()
	drainForType(t, ownerConn, EventPresenceChanged) // other joining

	unrelatedConn, _ := connectAndWatch(t, server, pool, unrelatedOwner, unrelatedBoard.ID)
	defer unrelatedConn.Close()

	if rec := postCoordination(t, pool, hub, reg, board.ID, owner, other, true); rec.Code != http.StatusOK {
		t.Fatalf("assign leader: %d %s", rec.Code, rec.Body.String())
	}

	// The bystander on the same board receives the change live.
	msg := drainForType(t, otherConn, EventCoordinationChanged)
	data, _ := msg["data"].(map[string]any)
	if data["group_leader_user_id"] != other {
		t.Fatalf("expected broadcast leader = other, got %+v", data)
	}

	// The unrelated board's own watcher never sees it.
	expectNoMessageOfType(t, unrelatedConn, EventCoordinationChanged)

	// Idempotent repeat: same target again succeeds and is harmless.
	rec2 := postCoordination(t, pool, hub, reg, board.ID, owner, other, true)
	if rec2.Code != http.StatusOK {
		t.Fatalf("idempotent repeat assign status = %d body=%s", rec2.Code, rec2.Body.String())
	}
	s, active := reg.Get(StoryboardVenueSessionID(board.ID))
	if !active || s.GroupLeaderUserID != other {
		t.Fatalf("expected leader to remain other after idempotent repeat, got %+v active=%v", s, active)
	}
}

// ---------- Export exclusion (spec 7, 15) ----------

func TestCoordinationStateExcludedFromExport(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "kc_owner15")
	board := mustCreateBoard(t, pool, owner, "Coord Export Exclusion Board")

	hub := network.NewHub()
	reg := venuecoordination.NewRegistry()
	server := httptest.NewServer(ServeStoryboardWS(hub, pool, reg))
	defer server.Close()

	conn, _ := connectAndWatch(t, server, pool, owner, board.ID)
	defer conn.Close()

	// Owner is Group Leader from session start -- if export ever leaked
	// live coordination state, this is exactly the state that would show
	// up in it.
	s, active := reg.Get(StoryboardVenueSessionID(board.ID))
	if !active || s.GroupLeaderUserID != owner {
		t.Fatalf("expected active session with owner as leader, got %+v active=%v", s, active)
	}

	doc, err := BuildBoardExport(ctx, pool, owner, board.ID)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal export: %v", err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatalf("unmarshal export: %v", err)
	}
	for _, forbidden := range []string{"coordination", "presence", "group_leader_user_id", "current_turn_user_id"} {
		if _, ok := asMap[forbidden]; ok {
			t.Fatalf("export document must not contain %q, got %s", forbidden, raw)
		}
	}
}

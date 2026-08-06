package storyboards

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/network"
	"victory/backend/internal/sessions"
)

func authenticatedRequest(t *testing.T, pool *pgxpool.Pool, method, target, userID string, body []byte) *http.Request {
	t.Helper()
	raw, _, err := sessions.CreateSession(context.Background(), pool, userID, time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, target, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	req.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	return req
}

func decodeHTTPResponse(t *testing.T, rec *httptest.ResponseRecorder) response {
	t.Helper()
	var out response
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func TestHandleBoardsRequiresAuthentication(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	req := httptest.NewRequest(http.MethodGet, "/api/storyboards", nil)
	rec := httptest.NewRecorder()
	HandleBoards(pool).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous list status = %d, want 401", rec.Code)
	}
}

func TestHandleBoardsCreateAndList(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "sb_http_owner")

	body, _ := json.Marshal(map[string]any{"title": "HTTP Board"})
	req := authenticatedRequest(t, pool, http.MethodPost, "/api/storyboards", owner, body)
	rec := httptest.NewRecorder()
	HandleBoards(pool).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create status = %d body=%s", rec.Code, rec.Body.String())
	}
	out := decodeHTTPResponse(t, rec)
	if !out.Ok {
		t.Fatalf("expected ok response, got %+v", out)
	}

	req2 := authenticatedRequest(t, pool, http.MethodGet, "/api/storyboards", owner, nil)
	rec2 := httptest.NewRecorder()
	HandleBoards(pool).ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", rec2.Code, rec2.Body.String())
	}
}

// TestRoleForgeryOverHTTPRejected is the end-to-end companion to
// authority_dbtest_test.go's TestServerResolvedTierIgnoresClientClaims: a
// Crew-tier grantee sends a column-create request whose JSON body carries
// extra, unsolicited fields that mimic an elevated-role claim ("role":
// "producer", "granted_role": "producer"). HandleColumns' request struct
// has no such field, so json.Decode silently ignores them -- but this test
// proves the *end-to-end* result is still a 403 driven by the DB grant,
// not merely that the Go struct happens to lack the field.
func TestRoleForgeryOverHTTPRejected(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "sb_http_owner2")
	crew := insertTestUser(t, pool, "sb_http_crew")
	board := mustCreateBoard(t, pool, owner, "Forgery HTTP Board")
	var crewHandle string
	pool.QueryRow(context.Background(), `SELECT handle FROM users WHERE id = $1`, crew).Scan(&crewHandle)
	if _, err := AddGrant(context.Background(), pool, owner, board.ID, crewHandle, "crew"); err != nil {
		t.Fatalf("grant crew: %v", err)
	}

	forgedBody := []byte(`{"title":"Forged Column","role":"producer","granted_role":"producer","is_operator":true}`)
	req := authenticatedRequest(t, pool, http.MethodPost, "/api/storyboards/"+board.ID+"/columns", crew, forgedBody)
	req.SetPathValue("board_id", board.ID)
	rec := httptest.NewRecorder()
	HandleColumns(pool, network.NewHub()).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for crew forging elevated fields, got %d body=%s", rec.Code, rec.Body.String())
	}

	out := decodeHTTPResponse(t, rec)
	if out.Ok {
		t.Fatalf("expected ok=false for rejected forgery attempt")
	}

	cols, err := ListColumns(context.Background(), pool, board.ID)
	if err != nil {
		t.Fatalf("list columns: %v", err)
	}
	for _, c := range cols {
		if c.Title == "Forged Column" {
			t.Fatalf("forged column create must not have succeeded")
		}
	}
}

func TestHandleCardItemVersionConflictReturns409(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "sb_http_owner3")
	board := mustCreateBoard(t, pool, owner, "Conflict Board")
	rows, _ := ListRows(context.Background(), pool, board.ID)
	cols, _ := ListColumns(context.Background(), pool, board.ID)
	card, err := CreateCard(context.Background(), pool, owner, board.ID, rows[0].ID, cols[0].ID, "C", "", "")
	if err != nil {
		t.Fatalf("create card: %v", err)
	}

	staleBody, _ := json.Marshal(map[string]any{"base_version": card.Version + 99, "title": "Stale"})
	req := authenticatedRequest(t, pool, http.MethodPatch, "/api/storyboards/"+board.ID+"/cards/"+card.ID, owner, staleBody)
	req.SetPathValue("board_id", board.ID)
	req.SetPathValue("card_id", card.ID)
	rec := httptest.NewRecorder()
	HandleCardItem(pool, network.NewHub()).ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for stale version, got %d body=%s", rec.Code, rec.Body.String())
	}
}

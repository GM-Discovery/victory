package thirdplace

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"victory/backend/internal/sessions"
)

func newAuthenticatedRequest(t *testing.T, method, target, userID string) *http.Request {
	t.Helper()
	pool := openThirdPlaceTestPool(t)
	raw, _, err := sessions.CreateSession(context.Background(), pool, userID, time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	req := httptest.NewRequest(method, target, nil)
	req.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	return req
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func TestHandleCollectionRequiresAuthentication(t *testing.T) {
	pool := openThirdPlaceTestPool(t)

	req := httptest.NewRequest(http.MethodGet, "/api/third-place/headshots", nil)
	rec := httptest.NewRecorder()
	HandleCollection(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous list status = %d, want 401", rec.Code)
	}
}

func TestHandleMeRequiresAuthentication(t *testing.T) {
	pool := openThirdPlaceTestPool(t)

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		req := httptest.NewRequest(method, "/api/third-place/headshots/me", nil)
		rec := httptest.NewRecorder()
		HandleMe(pool).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s anonymous status = %d, want 401", method, rec.Code)
		}
	}
}

func TestHandleMyHistoryRequiresAuthentication(t *testing.T) {
	pool := openThirdPlaceTestPool(t)

	req := httptest.NewRequest(http.MethodGet, "/api/third-place/headshots/me/history", nil)
	rec := httptest.NewRecorder()
	HandleMyHistory(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous history status = %d, want 401", rec.Code)
	}
}

func TestHandleMeFullLifecycle(t *testing.T) {
	pool := openThirdPlaceTestPool(t)
	userID := insertThirdPlaceTestUser(t, pool, "http_lifecycle")
	setStageNameAndFace(t, pool, userID, "HTTP Lifecycle Stage Name", "", "")

	// GET before leaving: no active headshot.
	getReq := newAuthenticatedRequest(t, http.MethodGet, "/api/third-place/headshots/me", userID)
	getRec := httptest.NewRecorder()
	HandleMe(pool).ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("initial GET status = %d", getRec.Code)
	}
	payload := decodeResponse(t, getRec)
	if payload["data"].(map[string]any)["headshot"] != nil {
		t.Fatalf("expected no active headshot before leaving one")
	}

	// POST leaves a Headshot.
	postReq := newAuthenticatedRequest(t, http.MethodPost, "/api/third-place/headshots/me", userID)
	postRec := httptest.NewRecorder()
	HandleMe(pool).ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("POST status = %d, body=%s", postRec.Code, postRec.Body.String())
	}
	postPayload := decodeResponse(t, postRec)
	postData := postPayload["data"].(map[string]any)
	if created, _ := postData["created"].(bool); !created {
		t.Fatalf("expected created=true on first POST, got %+v", postData)
	}
	headshot := postData["headshot"].(map[string]any)
	if headshot["stage_name"] != "HTTP Lifecycle Stage Name" {
		t.Fatalf("headshot stage_name = %v", headshot["stage_name"])
	}

	// Repeated POST does not create a duplicate.
	postAgainReq := newAuthenticatedRequest(t, http.MethodPost, "/api/third-place/headshots/me", userID)
	postAgainRec := httptest.NewRecorder()
	HandleMe(pool).ServeHTTP(postAgainRec, postAgainReq)
	postAgainPayload := decodeResponse(t, postAgainRec)
	postAgainData := postAgainPayload["data"].(map[string]any)
	if created, _ := postAgainData["created"].(bool); created {
		t.Fatalf("expected created=false on repeated POST")
	}
	if postAgainData["headshot"].(map[string]any)["headshot_id"] != headshot["headshot_id"] {
		t.Fatalf("repeated POST returned a different headshot id")
	}

	// DELETE removes it.
	delReq := newAuthenticatedRequest(t, http.MethodDelete, "/api/third-place/headshots/me", userID)
	delRec := httptest.NewRecorder()
	HandleMe(pool).ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d", delRec.Code)
	}

	// Repeated DELETE is safe.
	delAgainReq := newAuthenticatedRequest(t, http.MethodDelete, "/api/third-place/headshots/me", userID)
	delAgainRec := httptest.NewRecorder()
	HandleMe(pool).ServeHTTP(delAgainRec, delAgainReq)
	if delAgainRec.Code != http.StatusOK {
		t.Fatalf("repeated DELETE status = %d", delAgainRec.Code)
	}

	// History shows one placed+removed record.
	historyReq := newAuthenticatedRequest(t, http.MethodGet, "/api/third-place/headshots/me/history", userID)
	historyRec := httptest.NewRecorder()
	HandleMyHistory(pool).ServeHTTP(historyRec, historyReq)
	historyPayload := decodeResponse(t, historyRec)
	historyRows := historyPayload["data"].(map[string]any)["history"].([]any)
	if len(historyRows) != 1 {
		t.Fatalf("expected 1 history row, got %d", len(historyRows))
	}
	firstRow := historyRows[0].(map[string]any)
	if firstRow["status"] != "removed" {
		t.Fatalf("expected removed status in history, got %v", firstRow["status"])
	}
	for _, forbiddenKey := range []string{"stage_name", "portrait_url", "headline_facts", "email", "handle"} {
		if _, present := firstRow[forbiddenKey]; present {
			t.Fatalf("history row must not contain Face content or account fields, found %q: %+v", forbiddenKey, firstRow)
		}
	}
}

func TestHandleMeIgnoresClientSuppliedUserID(t *testing.T) {
	pool := openThirdPlaceTestPool(t)
	sessionUser := insertThirdPlaceTestUser(t, pool, "session_user")
	otherUser := insertThirdPlaceTestUser(t, pool, "spoof_target")

	// POST with a body naming a different account must never affect that
	// account -- HandleMe never reads a body at all, the mutation always
	// targets the session-derived user (Kernel 65 §9.2).
	req := newAuthenticatedRequest(t, http.MethodPost, "/api/third-place/headshots/me", sessionUser)
	req.Body = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"user_id":"`+otherUser+`"}`)).Body

	rec := httptest.NewRecorder()
	HandleMe(pool).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST status = %d, body=%s", rec.Code, rec.Body.String())
	}

	mine, err := GetMyHeadshot(context.Background(), pool, sessionUser)
	if err != nil {
		t.Fatalf("get session user headshot: %v", err)
	}
	if mine == nil {
		t.Fatalf("expected the session user to have an active headshot")
	}

	other, err := GetMyHeadshot(context.Background(), pool, otherUser)
	if err != nil {
		t.Fatalf("get other user headshot: %v", err)
	}
	if other != nil {
		t.Fatalf("client-supplied user_id in body must not affect that account, but it got a headshot: %+v", other)
	}
}

func TestHandleMeMethodNotAllowed(t *testing.T) {
	pool := openThirdPlaceTestPool(t)
	userID := insertThirdPlaceTestUser(t, pool, "method_check")

	req := newAuthenticatedRequest(t, http.MethodPut, "/api/third-place/headshots/me", userID)
	rec := httptest.NewRecorder()
	HandleMe(pool).ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PUT status = %d, want 405", rec.Code)
	}
}

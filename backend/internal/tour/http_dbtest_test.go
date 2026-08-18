package tour

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"victory/backend/internal/sessions"
)

func TestHandleStateRequiresAuthentication(t *testing.T) {
	pool := openTourTestPool(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/tours/state", HandleState(pool))

	req := httptest.NewRequest(http.MethodGet, "/api/tours/state", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated /api/tours/state, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCompleteIgnoresForeignUserIDInBody(t *testing.T) {
	pool := openTourTestPool(t)
	userID := insertTourTestUser(t, pool, "tour_http_complete")
	otherID := insertTourTestUser(t, pool, "tour_http_other")

	raw, _, err := sessions.CreateSession(context.Background(), pool, userID, time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/tours/{tour_key}/complete", HandleComplete(pool))

	// A forged user_id in the body has nowhere to go -- completeRequest only
	// decodes step_reached, so this documents that the field is structurally
	// ignored rather than merely untested.
	body, _ := json.Marshal(map[string]any{"user_id": otherID, "step_reached": "trailer-pin"})
	req := httptest.NewRequest(http.MethodPost, "/api/tours/"+KeyCampusMandatory+"/complete", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	has, err := HasCompletion(context.Background(), pool, Subject{UserID: userID}, KeyCampusMandatory, "", "")
	if err != nil || !has {
		t.Fatalf("expected the session's own user to have the completion, got has=%v err=%v", has, err)
	}
	hasOther, err := HasCompletion(context.Background(), pool, Subject{UserID: otherID}, KeyCampusMandatory, "", "")
	if err != nil || hasOther {
		t.Fatalf("the forged user_id in the body must not have received a completion, got %v", hasOther)
	}
}

func TestHandleSkipRejectsMandatoryTour(t *testing.T) {
	pool := openTourTestPool(t)
	userID := insertTourTestUser(t, pool, "tour_http_skip")

	raw, _, err := sessions.CreateSession(context.Background(), pool, userID, time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/tours/{tour_key}/skip", HandleSkip(pool))

	req := httptest.NewRequest(http.MethodPost, "/api/tours/"+KeyCampusMandatory+"/skip", nil)
	req.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 skipping a mandatory tour, got %d: %s", rec.Code, rec.Body.String())
	}

	has, err := HasCompletion(context.Background(), pool, Subject{UserID: userID}, KeyCampusMandatory, "", "")
	if err != nil || has {
		t.Fatalf("a rejected skip must not have written a row, got has=%v err=%v", has, err)
	}
}

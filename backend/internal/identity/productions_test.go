package identity

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

func productionsTestSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
}

func insertProductionsTestUser(t *testing.T, handlePrefix string) string {
	t.Helper()
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	handle := handlePrefix + "_" + productionsTestSuffix(t)

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, handle, handlePrefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func productionsTestSessionCookie(t *testing.T, userID string) *http.Cookie {
	t.Helper()
	pool := openDiscordTestPool(t)
	raw, _, err := sessions.CreateSession(context.Background(), pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return &http.Cookie{Name: sessions.CookieName, Value: raw}
}

func TestHandleProductionsCollectionCreateRequiresAuthentication(t *testing.T) {
	pool := openDiscordTestPool(t)

	req := httptest.NewRequest(http.MethodPost, "/api/productions", strings.NewReader(`{"name":"Anon Production"}`))
	rec := httptest.NewRecorder()
	HandleProductionsCollection(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleProductionsCollectionCreateRejectsAudienceOnly(t *testing.T) {
	pool := openDiscordTestPool(t)
	userID := insertProductionsTestUser(t, "prod_audience")

	var locationID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("load location: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'audience', TRUE)
	`, locationID, userID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/productions", strings.NewReader(`{"name":"Audience Production"}`))
	req.AddCookie(productionsTestSessionCookie(t, userID))
	rec := httptest.NewRecorder()
	HandleProductionsCollection(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for audience-only membership, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleProductionsCollectionCreateSucceedsForProducer(t *testing.T) {
	pool := openDiscordTestPool(t)
	userID := insertProductionsTestUser(t, "prod_producer")

	var locationID, locationSlug string
	if err := pool.QueryRow(context.Background(), `SELECT id::text, slug FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID, &locationSlug); err != nil {
		t.Fatalf("load location: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'producer', TRUE)
	`, locationID, userID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}

	name := "Test Create Production " + productionsTestSuffix(t)
	reqBody := `{"name":"` + name + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/productions", strings.NewReader(reqBody))
	req.AddCookie(productionsTestSessionCookie(t, userID))
	rec := httptest.NewRecorder()
	HandleProductionsCollection(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Ok   bool `json:"ok"`
		Data struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			Slug         string `json:"slug"`
			LocationSlug string `json:"location_slug"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Ok || payload.Data.ID == "" {
		t.Fatalf("unexpected response: %s", rec.Body.String())
	}
	if payload.Data.Name != name {
		t.Fatalf("expected name %q, got %q", name, payload.Data.Name)
	}
	if payload.Data.LocationSlug != locationSlug {
		t.Fatalf("expected location_slug %q (server-resolved), got %q", locationSlug, payload.Data.LocationSlug)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, payload.Data.ID)
	})

	// The newly created production must be usable: it appears in the
	// caller's own list (Kernel 68 §5.2 "existing conventions" check --
	// same authority path as HandleListProductions).
	listReq := httptest.NewRequest(http.MethodGet, "/api/productions", nil)
	listReq.AddCookie(productionsTestSessionCookie(t, userID))
	listRec := httptest.NewRecorder()
	HandleProductionsCollection(pool).ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 listing productions, got %d: %s", listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), payload.Data.ID) {
		t.Fatalf("expected newly created production %q in list: %s", payload.Data.ID, listRec.Body.String())
	}
}

func TestHandleProductionsCollectionCreateRejectsBlankName(t *testing.T) {
	pool := openDiscordTestPool(t)
	userID := insertProductionsTestUser(t, "prod_blank")

	var locationID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("load location: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'producer', TRUE)
	`, locationID, userID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/productions", strings.NewReader(`{"name":"   "}`))
	req.AddCookie(productionsTestSessionCookie(t, userID))
	rec := httptest.NewRecorder()
	HandleProductionsCollection(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for blank name, got %d: %s", rec.Code, rec.Body.String())
	}
}

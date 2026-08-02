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

func TestAccountDeletionPrivateOnlyUserSucceeds(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	handle := "del_solo_" + time.Now().UTC().Format("150405.000000")
	userID := insertAccountTestUser(t, pool, handle, "Delete Me Solo")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	locationID := resolveAccountTestLocationID(t, pool, "amurray-family")
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'audience', TRUE)
	`, locationID, userID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}

	var characterID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO character_cards (owner_user_id, location_id, name)
		VALUES ($1, $2, 'Solo Character')
		RETURNING id::text
	`, userID, locationID).Scan(&characterID); err != nil {
		t.Fatalf("insert character card: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO character_journals (character_card_id, author_user_id, body)
		VALUES ($1, $2, 'nobody else should see this')
	`, characterID, userID); err != nil {
		t.Fatalf("insert journal: %v", err)
	}

	raw, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Plan should show no blockers.
	planReq := httptest.NewRequest(http.MethodGet, "/api/account/deletion-plan", nil)
	planReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	planRec := httptest.NewRecorder()
	HandleAccountDeletionPlan(pool).ServeHTTP(planRec, planReq)
	if planRec.Code != http.StatusOK {
		t.Fatalf("plan status = %d, body %s", planRec.Code, planRec.Body.String())
	}
	var planPayload struct {
		Ok   bool         `json:"ok"`
		Data DeletionPlan `json:"data"`
	}
	if err := json.Unmarshal(planRec.Body.Bytes(), &planPayload); err != nil {
		t.Fatalf("decode plan: %v", err)
	}
	if !planPayload.Data.CanDelete {
		t.Fatalf("expected can_delete=true, blockers=%+v", planPayload.Data.Blockers)
	}
	if planPayload.Data.PrivateRecordCounts["characters"] != 1 {
		t.Fatalf("expected 1 character in plan preview, got %+v", planPayload.Data.PrivateRecordCounts)
	}

	// Execute deletion.
	body, _ := json.Marshal(executeDeletionRequest{ConfirmHandle: handle})
	delReq := httptest.NewRequest(http.MethodPost, "/api/account/delete", strings.NewReader(string(body)))
	delReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	delRec := httptest.NewRecorder()
	HandleAccountDelete(pool, t.TempDir()).ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body %s", delRec.Code, delRec.Body.String())
	}

	// Session cookie must be cleared.
	cleared := false
	for _, c := range delRec.Result().Cookies() {
		if c.Name == sessions.CookieName && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("expected session cookie to be cleared")
	}

	// User row, credentials, journal, character must all be gone.
	var stillExists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&stillExists); err != nil {
		t.Fatalf("check user: %v", err)
	}
	if stillExists {
		t.Fatal("user row should be deleted")
	}
	var journalCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM character_journals WHERE character_card_id = $1`, characterID).Scan(&journalCount); err != nil {
		t.Fatalf("check journal: %v", err)
	}
	if journalCount != 0 {
		t.Fatalf("expected private journal to be cascade-deleted, got %d rows", journalCount)
	}
	var sessionCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM auth.sessions WHERE user_id = $1`, userID).Scan(&sessionCount); err != nil {
		t.Fatalf("check sessions: %v", err)
	}
	if sessionCount != 0 {
		t.Fatalf("expected all sessions revoked/removed, got %d", sessionCount)
	}

	// Post-deletion, the cleared cookie must not resolve to anything.
	meReq := httptest.NewRequest(http.MethodGet, "/api/account/me", nil)
	meReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	meRec := httptest.NewRecorder()
	HandleAccountMe(pool).ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected the old session to be dead post-deletion, got %d", meRec.Code)
	}
}

func TestAccountDeletionAnonymizesSharedActionsNotDeletesThem(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	handle := "del_actor_" + time.Now().UTC().Format("150405.000000")
	userID := insertAccountTestUser(t, pool, handle, "Delete Me Actor")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	// A specific, stable, non-catharsis venue: internal/shows has tests
	// sensitive to concurrent session activity on the catharsis venue
	// specifically (go test ./... runs packages in parallel against one
	// shared victory_test database), so avoid it here.
	var venueID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM venues WHERE slug = 'info-booth'`).Scan(&venueID); err != nil {
		t.Fatalf("resolve a venue: %v", err)
	}
	var gameSessionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO sessions (venue_id, status) VALUES ($1, 'rehearsal') RETURNING id::text
	`, venueID).Scan(&gameSessionID); err != nil {
		t.Fatalf("insert game session: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, gameSessionID) })

	var actionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO actions (session_id, moment_id, actor_id, type, target, payload)
		VALUES ($1, 1, $2, 'game/event', '{}'::jsonb, '{}'::jsonb)
		RETURNING id::text
	`, gameSessionID, userID).Scan(&actionID); err != nil {
		t.Fatalf("insert action: %v", err)
	}

	raw, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	body, _ := json.Marshal(executeDeletionRequest{ConfirmHandle: handle})
	delReq := httptest.NewRequest(http.MethodPost, "/api/account/delete", strings.NewReader(string(body)))
	delReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	delRec := httptest.NewRecorder()
	HandleAccountDelete(pool, t.TempDir()).ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body %s", delRec.Code, delRec.Body.String())
	}

	// The action row must survive (shared Show history), reassigned to the
	// tombstone account, not cascade-deleted and not left NULL.
	var actorID string
	if err := pool.QueryRow(ctx, `SELECT actor_id::text FROM actions WHERE id = $1`, actionID).Scan(&actorID); err != nil {
		t.Fatalf("expected action row to survive deletion: %v", err)
	}
	if actorID != tombstoneUserID {
		t.Fatalf("expected actor_id reassigned to tombstone %q, got %q", tombstoneUserID, actorID)
	}

	var tombstoneDisplayName string
	if err := pool.QueryRow(ctx, `SELECT display_name FROM users WHERE id = $1`, tombstoneUserID).Scan(&tombstoneDisplayName); err != nil {
		t.Fatalf("tombstone account must exist: %v", err)
	}
	if tombstoneDisplayName != "Deleted User" {
		t.Fatalf("unexpected tombstone display name %q", tombstoneDisplayName)
	}
}

func TestAccountDeletionBlocksSoleProducerUntilTransfer(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	suffix := time.Now().UTC().Format("150405.000000")
	handle := "del_producer_" + suffix
	userID := insertAccountTestUser(t, pool, handle, "Sole Producer")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	var locationID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO locations (slug, name) VALUES ($1, $2) RETURNING id::text
	`, "del-test-location-"+suffix, "Deletion Test Location").Scan(&locationID); err != nil {
		t.Fatalf("insert location: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, locationID) })

	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'producer', TRUE)
	`, locationID, userID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO productions (location_id, name, slug, created_by_user_id)
		VALUES ($1, 'Deletion Test Production', $2, $3)
	`, locationID, "del-test-production-"+suffix, userID); err != nil {
		t.Fatalf("insert production: %v", err)
	}

	raw, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Blocked: sole producer of a Location with a Production.
	body, _ := json.Marshal(executeDeletionRequest{ConfirmHandle: handle})
	delReq := httptest.NewRequest(http.MethodPost, "/api/account/delete", strings.NewReader(string(body)))
	delReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	delRec := httptest.NewRecorder()
	HandleAccountDelete(pool, t.TempDir()).ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusConflict {
		t.Fatalf("expected 409 deletion_blocked, got %d: %s", delRec.Code, delRec.Body.String())
	}
	if !strings.Contains(delRec.Body.String(), "deletion_blocked") {
		t.Fatalf("expected deletion_blocked error, got %s", delRec.Body.String())
	}

	// Grant another producer at the same Location -- blocker should clear.
	otherProducerID := insertAccountTestUser(t, pool, "del_other_producer_"+suffix, "Other Producer")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, otherProducerID)
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'producer', TRUE)
	`, locationID, otherProducerID); err != nil {
		t.Fatalf("insert second producer membership: %v", err)
	}

	delReq2 := httptest.NewRequest(http.MethodPost, "/api/account/delete", strings.NewReader(string(body)))
	delReq2.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	delRec2 := httptest.NewRecorder()
	HandleAccountDelete(pool, t.TempDir()).ServeHTTP(delRec2, delReq2)
	if delRec2.Code != http.StatusOK {
		t.Fatalf("expected deletion to succeed once a second producer exists, got %d: %s", delRec2.Code, delRec2.Body.String())
	}
}

func TestAccountDeletionBlocksOperatorAccount(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	handle := "del_operator_" + time.Now().UTC().Format("150405.000000")
	userID := insertAccountTestUser(t, pool, handle, "Operator Account")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	t.Setenv("OPERATOR_HANDLE", handle)
	t.Setenv("OPERATOR_USER_ID", "")

	raw, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	body, _ := json.Marshal(executeDeletionRequest{ConfirmHandle: handle})
	delReq := httptest.NewRequest(http.MethodPost, "/api/account/delete", strings.NewReader(string(body)))
	delReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	delRec := httptest.NewRecorder()
	HandleAccountDelete(pool, t.TempDir()).ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusConflict {
		t.Fatalf("expected operator self-deletion to be blocked with 409, got %d: %s", delRec.Code, delRec.Body.String())
	}

	var stillExists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&stillExists); err != nil {
		t.Fatalf("check user: %v", err)
	}
	if !stillExists {
		t.Fatal("operator account must not have been deleted")
	}
}

func TestAccountDeletionRejectsConfirmHandleMismatch(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	handle := "del_mismatch_" + time.Now().UTC().Format("150405.000000")
	userID := insertAccountTestUser(t, pool, handle, "Mismatch Test")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	raw, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	body, _ := json.Marshal(executeDeletionRequest{ConfirmHandle: "not-my-handle"})
	delReq := httptest.NewRequest(http.MethodPost, "/api/account/delete", strings.NewReader(string(body)))
	delReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	delRec := httptest.NewRecorder()
	HandleAccountDelete(pool, t.TempDir()).ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 confirm_handle_mismatch, got %d: %s", delRec.Code, delRec.Body.String())
	}

	var stillExists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&stillExists); err != nil {
		t.Fatalf("check user: %v", err)
	}
	if !stillExists {
		t.Fatal("user must not be deleted on a confirm-handle mismatch")
	}
}

func TestAccountDeletionRequiresAuthentication(t *testing.T) {
	pool := openDiscordTestPool(t)

	body, _ := json.Marshal(executeDeletionRequest{ConfirmHandle: "whoever"})
	delReq := httptest.NewRequest(http.MethodPost, "/api/account/delete", strings.NewReader(string(body)))
	delRec := httptest.NewRecorder()
	HandleAccountDelete(pool, t.TempDir()).ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for an unauthenticated delete request, got %d", delRec.Code)
	}

	planReq := httptest.NewRequest(http.MethodGet, "/api/account/deletion-plan", nil)
	planRec := httptest.NewRecorder()
	HandleAccountDeletionPlan(pool).ServeHTTP(planRec, planReq)
	if planRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for an unauthenticated plan request, got %d", planRec.Code)
	}
}

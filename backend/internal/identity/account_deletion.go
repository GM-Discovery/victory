package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"victory/backend/internal/access"
	"victory/backend/internal/sessions"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// tombstoneUserID is the dedicated, credential-less "Deleted User" account
// created by migration 081. Shared canonical history that would otherwise
// be blocked from deletion by a RESTRICT foreign key (see
// Construction/Domains/Security/kernel-76-data-classification.md §3) is reassigned
// to this account rather than left NULL, so existing rendering code that
// joins to users.display_name needs no changes to show "Deleted User."
//
// Every other foreign key to users (the large majority) is already CASCADE
// or SET NULL, verified correct-as-is by the Kernel 76 audit; this file
// only has to resolve the twelve RESTRICT columns plus the three on assets.
const tombstoneUserID = "00000000-0000-0000-0000-0000000000dd"

// DeletionBlocker explains one reason self-service deletion cannot proceed
// yet, and how to resolve it, per Kernel 77 §1.1 ("must explain the
// blocker... must not silently fail").
type DeletionBlocker struct {
	LocationSlug string `json:"location_slug"`
	LocationName string `json:"location_name"`
	Reason       string `json:"reason"`
	Resolution   string `json:"resolution"`
}

// DeletionPlan is the read-only preview shown before a user commits to
// deletion (Kernel 77 §6.1, §6.2).
type DeletionPlan struct {
	Handle              string            `json:"handle"`
	IsOperator          bool              `json:"is_operator"`
	HasPassword         bool              `json:"has_password"`
	Blockers            []DeletionBlocker `json:"blockers"`
	CanDelete           bool              `json:"can_delete"`
	PrivateRecordCounts map[string]int    `json:"private_record_counts"`
	RequiresPassword    bool              `json:"requires_current_password"`
	RequiresRecentLogin bool              `json:"requires_recent_login"`
}

// DeletionReceipt is the minimal, non-identifying audit record kept after a
// successful deletion (Kernel 77 §6.7). It deliberately does not carry the
// original user id, email, handle, or Discord id.
type DeletionReceipt struct {
	ID                    string    `json:"id"`
	CreatedAt             time.Time `json:"created_at"`
	DeletedPrivateCount   int       `json:"deleted_private_count"`
	AnonymizedSharedCount int       `json:"anonymized_shared_count"`
	Status                string    `json:"status"`
}

// recentLoginWindow bounds how old the current session may be for it to
// count as "recent authentication" under Kernel 77 §6.1 step 4 for accounts
// with no password credential (the ordinary case: password signup is closed,
// so most accounts are Discord-only, see account_email.go's identical
// limitation for a narrower endpoint). A session younger than this proves
// the user completed a real login (password or Discord OAuth) recently,
// without building a separate step-up-auth flow: the client simply needs to
// sign out and back in if its session is older than this window.
const recentLoginWindow = 15 * time.Minute

func currentSessionCreatedAt(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (time.Time, error) {
	raw, err := sessions.ReadSessionCookie(r)
	if err != nil {
		return time.Time{}, err
	}
	tokenHash := sessions.HashToken(raw)
	var createdAt time.Time
	err = pool.QueryRow(ctx, `
		SELECT created_at FROM auth.sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, tokenHash).Scan(&createdAt)
	return createdAt, err
}

// BuildDeletionPlan computes what would happen if userID deleted their
// account right now, without changing anything.
func BuildDeletionPlan(ctx context.Context, pool *pgxpool.Pool, userID string) (*DeletionPlan, error) {
	plan := &DeletionPlan{
		Blockers:            []DeletionBlocker{},
		PrivateRecordCounts: map[string]int{},
	}

	if err := pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, userID).Scan(&plan.Handle); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errAccountNotFound
		}
		return nil, err
	}

	isOperator, err := access.IsOperatorUser(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	plan.IsOperator = isOperator
	if isOperator {
		plan.Blockers = append(plan.Blockers, DeletionBlocker{
			Reason:     "this account is the Victory operator account",
			Resolution: "operator accounts cannot self-delete; reassign OPERATOR_HANDLE to a different account first (operator shell action, not self-service)",
		})
	}

	var hasPassword bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM auth.password_credentials WHERE user_id = $1)`, userID).Scan(&hasPassword); err != nil {
		return nil, err
	}
	plan.HasPassword = hasPassword
	plan.RequiresPassword = hasPassword
	plan.RequiresRecentLogin = !hasPassword

	// Sole-producer blocker: authority to manage a Location's Productions
	// and Show Runs in this schema comes from an active `producer` role
	// membership at that Location (not per-Production ownership -- see
	// account-deletion-dependency-map.md). If this user is the only active
	// producer at a Location that has any Production, deleting them would
	// leave it with no one able to administer it.
	rows, err := pool.Query(ctx, `
		SELECT l.id::text, l.slug, l.name
		FROM location_memberships lm
		JOIN locations l ON l.id = lm.location_id
		WHERE lm.user_id = $1 AND lm.role = 'producer' AND lm.active = TRUE
	`, userID)
	if err != nil {
		return nil, err
	}
	type loc struct{ id, slug, name string }
	var producerLocations []loc
	for rows.Next() {
		var l loc
		if err := rows.Scan(&l.id, &l.slug, &l.name); err != nil {
			rows.Close()
			return nil, err
		}
		producerLocations = append(producerLocations, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, l := range producerLocations {
		var otherProducerExists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM location_memberships
				WHERE location_id = $1 AND role = 'producer' AND active = TRUE AND user_id != $2
			)
		`, l.id, userID).Scan(&otherProducerExists); err != nil {
			return nil, err
		}
		if otherProducerExists {
			continue
		}

		var productionCount int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM productions WHERE location_id = $1`, l.id).Scan(&productionCount); err != nil {
			return nil, err
		}
		if productionCount == 0 {
			continue
		}

		plan.Blockers = append(plan.Blockers, DeletionBlocker{
			LocationSlug: l.slug,
			LocationName: l.name,
			Reason:       "you are the only Producer at this Location, which has active Productions",
			Resolution:   "grant Producer access to another member at this Location (Account Settings > Members, or ask them to accept an invite), then retry deletion",
		})
	}

	// Kernel 80: owned-Storyboard blocker. storyboards.owner_user_id is
	// ON DELETE RESTRICT (migration 090) precisely so this blocker, not a
	// cascade, is what clears ownership -- "no ownerless boards can
	// remain" (spec 12.2). Every owned board blocks, archived included:
	// the spec's own text says "owned active Storyboards appear in the
	// deletion plan," but archived boards still hold that same RESTRICT
	// FK, and Kernel 80 has no ownership-transfer or archived-board
	// reassignment mechanism -- treating only "active" as blocking would
	// leave a real path to an unresolvable RESTRICT violation at execute
	// time. Simpler and safer to block on all owned boards uniformly
	// (confirmed decision, not a spec gap): the owner must archive or
	// delete every owned board, or transfer ownership if a future kernel
	// adds that, before deletion can proceed. Raw SQL, not the
	// storyboards package's Go API -- identity cannot import storyboards
	// (network already imports identity, and storyboards imports network
	// for its live WS events, so identity -> storyboards would close an
	// import cycle).
	boardRows, err := pool.Query(ctx, `SELECT id::text, title, archived_at FROM storyboards WHERE owner_user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	type ownedBoard struct {
		id, title  string
		archivedAt *time.Time
	}
	var ownedBoards []ownedBoard
	for boardRows.Next() {
		var b ownedBoard
		if err := boardRows.Scan(&b.id, &b.title, &b.archivedAt); err != nil {
			boardRows.Close()
			return nil, err
		}
		ownedBoards = append(ownedBoards, b)
	}
	boardRows.Close()
	if err := boardRows.Err(); err != nil {
		return nil, err
	}
	for _, b := range ownedBoards {
		status := "active"
		if b.archivedAt != nil {
			status = "archived"
		}
		plan.Blockers = append(plan.Blockers, DeletionBlocker{
			Reason:     "you own a " + status + " Storyboard: " + b.title,
			Resolution: "delete this Storyboard (Storyboards > board settings > Delete), or transfer ownership if a future feature adds that, then retry deletion",
		})
	}

	// Representative private-record counts for the plan preview -- not
	// exhaustive across every private table, enough to show the user what
	// "private data" concretely means for their account.
	countQueries := map[string]string{
		"characters":        `SELECT COUNT(*) FROM character_cards WHERE owner_user_id = $1`,
		"journal_entries":   `SELECT COUNT(*) FROM character_journals cj JOIN character_cards cc ON cc.id = cj.character_card_id WHERE cc.owner_user_id = $1`,
		"relationships":     `SELECT COUNT(*) FROM player_relationships WHERE observer_user_id = $1 OR subject_user_id = $1`,
		"messages_received": `SELECT COUNT(*) FROM messages WHERE to_user_id = $1`,
		"uploads":           `SELECT COUNT(*) FROM assets WHERE (owner_user_id = $1 OR uploader_user_id = $1) AND is_deleted = FALSE`,
		"ewritings":         `SELECT COUNT(*) FROM ewrite_publications WHERE created_by = $1`,
		"storyboards_owned": `SELECT COUNT(*) FROM storyboards WHERE owner_user_id = $1`,
	}
	for label, q := range countQueries {
		var n int
		if err := pool.QueryRow(ctx, q, userID).Scan(&n); err != nil {
			return nil, err
		}
		plan.PrivateRecordCounts[label] = n
	}

	plan.CanDelete = len(plan.Blockers) == 0
	return plan, nil
}

type deletionPlanRequest struct{}

func HandleAccountDeletionPlan(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		plan, err := BuildDeletionPlan(ctx, pool, userID)
		if err != nil {
			if errors.Is(err, errAccountNotFound) {
				writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "failed_to_build_plan"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": plan})
	}
}

type executeDeletionRequest struct {
	ConfirmHandle   string `json:"confirm_handle"`
	CurrentPassword string `json:"current_password"`
}

// HandleAccountDelete executes a self-service account deletion. It always
// re-resolves the requester from the session (never a client-supplied user
// id) and always re-checks the plan at execution time, per Kernel 77 §6.2 --
// a blocker that appeared after the plan was fetched (e.g. someone else
// just lost producer access) must still be caught.
func HandleAccountDelete(pool *pgxpool.Pool, storageRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		var input executeDeletionRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}

		plan, err := BuildDeletionPlan(ctx, pool, userID)
		if err != nil {
			if errors.Is(err, errAccountNotFound) {
				writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "failed_to_build_plan"})
			return
		}
		if !plan.CanDelete {
			writeJSON(w, http.StatusConflict, map[string]any{"ok": false, "error": "deletion_blocked", "data": plan})
			return
		}

		// Deliberate final confirmation phrase (§6.1 step 5): the exact
		// handle, case-sensitive.
		if strings.TrimSpace(input.ConfirmHandle) != plan.Handle {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "confirm_handle_mismatch"})
			return
		}

		// Recent authentication or equivalent (§6.1 step 4).
		if plan.HasPassword {
			var storedHash string
			if err := pool.QueryRow(ctx, `SELECT password_hash FROM auth.password_credentials WHERE user_id = $1`, userID).Scan(&storedHash); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "reauth_check_failed"})
				return
			}
			ok, err := VerifyPassword(input.CurrentPassword, storedHash)
			if err != nil || !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "invalid_credentials"})
				return
			}
		} else {
			createdAt, err := currentSessionCreatedAt(ctx, pool, r)
			if err != nil || time.Since(createdAt) > recentLoginWindow {
				writeJSON(w, http.StatusUnauthorized, map[string]any{
					"ok": false, "error": "recent_login_required",
					"detail": "sign out and back in with Discord, then retry within 15 minutes",
				})
				return
			}
		}

		receipt, filesToDelete, err := executeDeletion(ctx, pool, userID, storageRoot)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "deletion_failed"})
			return
		}

		// Best-effort filesystem cleanup after the transaction has
		// committed. A failure here does not undo the deletion -- the
		// database is the source of truth, and an orphaned file with no
		// database row is inert and can be swept later.
		for _, path := range filesToDelete {
			_ = os.RemoveAll(path)
		}

		sessions.ClearSessionCookie(w, r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https")
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": receipt})
	}
}

// executeDeletion performs the actual deletion inside one transaction:
// reassign RESTRICT-FK history to the tombstone account, resolve assets
// (anonymize if in active shared use, otherwise mark for hard delete),
// delete the user row (cascading everything else per the existing,
// Kernel-76-verified CASCADE/SET NULL design), and record a receipt.
func executeDeletion(ctx context.Context, pool *pgxpool.Pool, userID, storageRoot string) (*DeletionReceipt, []string, error) {
	// Representative private-record count, taken before deletion, for the
	// receipt -- see BuildDeletionPlan's identical set.
	deletedPrivate := 0
	for _, q := range []string{
		`SELECT COUNT(*) FROM character_cards WHERE owner_user_id = $1`,
		`SELECT COUNT(*) FROM character_journals cj JOIN character_cards cc ON cc.id = cj.character_card_id WHERE cc.owner_user_id = $1`,
		`SELECT COUNT(*) FROM player_relationships WHERE observer_user_id = $1 OR subject_user_id = $1`,
		`SELECT COUNT(*) FROM messages WHERE to_user_id = $1`,
	} {
		var n int
		if err := pool.QueryRow(ctx, q, userID).Scan(&n); err != nil {
			return nil, nil, err
		}
		deletedPrivate += n
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	anonymized := 0

	// eWrite (Kernel 78), ordered BEFORE the generic reassign slice
	// because the draft rule needs the pre-reassignment created_by:
	//   - sole-owned drafts (no other named editor) are private unfinished
	//     work -> hard-deleted; revisions/sections/aliases/links CASCADE.
	//   - everything else (published/archived works, drafts with other
	//     named editors, collections) is shared production material ->
	//     reassigned to the tombstone; the work survives, authorship is
	//     anonymized (kernel 78 spec 12.4: "shared published rules are not
	//     destroyed casually", "no ownerless publication remains").
	if _, err := tx.Exec(ctx, `
		DELETE FROM ewrite_publications p
		WHERE p.created_by = $1
		  AND p.status = 'draft'
		  AND NOT EXISTS (
			SELECT 1 FROM ewrite_editors e
			WHERE e.publication_id = p.id AND e.user_id <> $1
		  )
	`, userID); err != nil {
		return nil, nil, err
	}

	reassign := []string{
		`UPDATE ewrite_publications SET created_by = $2 WHERE created_by = $1`,
		`UPDATE ewrite_publications SET updated_by = $2 WHERE updated_by = $1`,
		`UPDATE ewrite_collections SET created_by = $2 WHERE created_by = $1`,
		`UPDATE ewrite_revisions SET created_by = $2 WHERE created_by = $1`,
		`UPDATE ewrite_editors SET granted_by = $2 WHERE granted_by = $1`,
		`UPDATE ewrite_object_links SET created_by = $2 WHERE created_by = $1`,
		`UPDATE actions SET actor_id = $2 WHERE actor_id = $1`,
		`UPDATE cue_executions SET triggered_by_user_id = $2 WHERE triggered_by_user_id = $1`,
		`UPDATE character_inventory_items SET acquired_by_user_id = $2 WHERE acquired_by_user_id = $1`,
		`UPDATE permission_grants SET granted_by_user_id = $2 WHERE granted_by_user_id = $1`,
		`UPDATE show_run_audience_blocks SET blocked_by_user_id = $2 WHERE blocked_by_user_id = $1`,
		`UPDATE show_run_roster_members SET added_by_user_id = $2 WHERE added_by_user_id = $1`,
		`UPDATE show_runs SET created_by_user_id = $2 WHERE created_by_user_id = $1`,
		`UPDATE showings SET created_by = $2 WHERE created_by = $1`,
		`UPDATE shows SET created_by_user_id = $2 WHERE created_by_user_id = $1`,
		// Kernel 80: authored cards remain with anonymized authorship
		// (spec 12.2's "authored cards remain... where continuity
		// requires it"), and grantor attribution is reassigned the same
		// way ewrite_editors.granted_by already is above. storyboard_
		// grants rows where THIS user is the grantee (recipient, not
		// grantor) need no code here at all: storyboard_grants.user_id is
		// ON DELETE CASCADE (migration 090), so the `DELETE FROM users`
		// below already removes them -- "non-owner deletion removes
		// their grants without creating ghost grants" is satisfied by
		// that FK alone.
		`UPDATE storyboard_cards SET author_user_id = $2 WHERE author_user_id = $1`,
		`UPDATE storyboard_grants SET granted_by = $2 WHERE granted_by = $1`,
	}
	for _, q := range reassign {
		tag, err := tx.Exec(ctx, q, userID, tombstoneUserID)
		if err != nil {
			return nil, nil, err
		}
		anonymized += int(tag.RowsAffected())
	}

	// Assets: anonymize (keep file + row) if still in active shared use as
	// a venue's current map (assets.owner/producer/uploader are RESTRICT,
	// and venue_active_maps.asset_id is itself RESTRICT, so an in-use map
	// image cannot be hard-deleted without breaking a live venue anyway).
	// Everything else the user exclusively owned/uploaded/produced is
	// hard-deleted: the row (cascading asset_derivatives) and the files on
	// disk, collected here and removed after commit.
	rows, err := tx.Query(ctx, `
		SELECT id::text, producer_user_id::text
		FROM assets
		WHERE (owner_user_id = $1 OR producer_user_id = $1 OR uploader_user_id = $1)
	`, userID)
	if err != nil {
		return nil, nil, err
	}
	type assetRow struct{ id, producerUserID string }
	var candidateAssets []assetRow
	for rows.Next() {
		var a assetRow
		if err := rows.Scan(&a.id, &a.producerUserID); err != nil {
			rows.Close()
			return nil, nil, err
		}
		candidateAssets = append(candidateAssets, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var filesToDelete []string
	for _, a := range candidateAssets {
		var inActiveUse bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM venue_active_maps WHERE asset_id = $1)`, a.id).Scan(&inActiveUse); err != nil {
			return nil, nil, err
		}
		if inActiveUse {
			tag, err := tx.Exec(ctx, `
				UPDATE assets SET owner_user_id = $2, producer_user_id = $2, uploader_user_id = $2
				WHERE id = $1
			`, a.id, tombstoneUserID)
			if err != nil {
				return nil, nil, err
			}
			anonymized += int(tag.RowsAffected())
			continue
		}

		if _, err := tx.Exec(ctx, `DELETE FROM assets WHERE id = $1`, a.id); err != nil {
			return nil, nil, err
		}
		deletedPrivate++
		// Matches internal/assets/upload.go's assetDir layout exactly:
		// {STORAGE_ROOT}/producers/{producer_user_id}/assets/{asset_id}/,
		// containing both original/ and derived/. Removing the whole
		// directory covers both without needing per-derivative rows.
		filesToDelete = append(filesToDelete, filepath.Join(storageRoot, "producers", a.producerUserID, "assets", a.id))
	}

	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		return nil, nil, err
	}

	receipt := &DeletionReceipt{
		Status:                "completed",
		DeletedPrivateCount:   deletedPrivate,
		AnonymizedSharedCount: anonymized,
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO account_deletion_receipts (deleted_private_count, anonymized_shared_count, status)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, receipt.DeletedPrivateCount, receipt.AnonymizedSharedCount, receipt.Status).Scan(&receipt.ID, &receipt.CreatedAt); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	return receipt, filesToDelete, nil
}

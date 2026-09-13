// ---
// Value Function (Conceptual)
//
// V(person) ≈ (A × R × C) / P
//
// A = Agency (ability to act and be accountable)
// R = Relational continuity (history + context across time)
// C = Capacity for correction (ability to responsibly override)
// P = Replaceability (→ 0 for true identity)
//
// As P → 0, value → ∞.
//
// A person is not a user account.
// ---
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/assets"
	"victory/backend/internal/audienceadmission"
	"victory/backend/internal/audiencenotes"
	"victory/backend/internal/audienceprojection"
	"victory/backend/internal/characters"
	"victory/backend/internal/cohorts"
	"victory/backend/internal/cues"
	"victory/backend/internal/db"
	"victory/backend/internal/directorprep"
	"victory/backend/internal/drawing"
	"victory/backend/internal/ewrite"
	"victory/backend/internal/firstrun"
	"victory/backend/internal/identity"
	"victory/backend/internal/mailer"
	"victory/backend/internal/merchant"
	"victory/backend/internal/messages"
	"victory/backend/internal/migrate"
	"victory/backend/internal/network"
	"victory/backend/internal/participation"
	"victory/backend/internal/playerprofile"
	"victory/backend/internal/playerrelationships"
	"victory/backend/internal/profiles"
	"victory/backend/internal/ratelimit"
	"victory/backend/internal/scenes"
	"victory/backend/internal/showings"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
	"victory/backend/internal/showtime"
	"victory/backend/internal/socio"
	"victory/backend/internal/stageobjects"
	"victory/backend/internal/storyboards"
	"victory/backend/internal/thirdplace"
	"victory/backend/internal/tickets"
	"victory/backend/internal/tour"
	"victory/backend/internal/venuecoordination"
	"victory/backend/internal/venues"
	"victory/backend/internal/world"
)

func main() {
	port := getenv("PORT", "8081")
	// Empty by default (binds every interface, ":8081") -- exactly
	// today's behavior, required for the container deployments (Grant's
	// production server, packaging/podman) where the backend has to be
	// reachable from other containers on the network, not just itself.
	// The Windows native launcher sets this to "127.0.0.1" specifically:
	// binding all interfaces there is what triggers a Windows Firewall
	// "allow this app" prompt (Windows can't tell the difference between
	// "accepts connections from the internet" and "accepts connections
	// from Caddy on the same machine" just from the bind address) --
	// confirmed on real hardware. A loopback-only bind can't be reached
	// from another machine at all, so Windows has nothing to prompt about.
	bindHost := getenv("BIND_HOST", "")
	databaseURL := getenv("DATABASE_URL", "postgres://victory:REDACTED@victory-postgres:5432/victory?sslmode=disable")
	secureCookie := cookieSecureFromEnv()
	storageRoot := getenv("STORAGE_ROOT", "/opt/victory/storage")
	exportsRoot := getenv("EXPORTS_ROOT", "/opt/victory/exports")
	discordOAuthConfig := discordOAuthConfigFromEnv()
	discordServerLinkConfig := discordServerLinkConfigFromEnv()
	discordGatewayConfig := discordGatewayConfigFromEnv()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	// Kernel 72: embedded migrations are the single schema truth. This runs
	// before every Ensure* seed bootstrap and gets its own generous timeout —
	// a first-time adoption pass re-applies the full history.
	//
	// Kernel 96: split into two phases so access.EnsureDefaultLocation runs
	// between them. Migration 001 only creates schema (no location rows);
	// 002 onward attaches canonical content to whichever location is
	// is_default. That location doesn't exist until EnsureDefaultLocation
	// creates it under this install's own DEFAULT_LOCATION_SLUG -- running
	// it here, mid-migration, is what lets content-seeding migrations stay
	// slug-agnostic instead of hardcoding one install's lot name.
	migrateOpts := migrate.OptionsFromEnv(databaseURL)
	migrateCtx, migrateCancel := context.WithTimeout(context.Background(), 10*time.Minute)
	phaseOneOpts := migrateOpts
	phaseOneOpts.StopAfter = "001_init.sql"
	if err := migrate.Run(migrateCtx, pool, phaseOneOpts); err != nil {
		log.Fatalf("migrations (schema phase): %v", err)
	}

	if err := access.EnsureDefaultLocation(ctx, pool); err != nil {
		log.Fatalf("default location bootstrap failed: %v", err)
	}

	if err := migrate.Run(migrateCtx, pool, migrateOpts); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	migrateCancel()
	if err := profiles.EnsureKernel9ProfileSurface(ctx, pool); err != nil {
		log.Fatalf("kernel 9 profile bootstrap failed: %v", err)
	}
	if err := access.EnsureKernel16VenueSurface(ctx, pool); err != nil {
		log.Fatalf("kernel 16 venue bootstrap failed: %v", err)
	}
	// Kernel 78: every Victory install ships with the Socio v1.1 core
	// rulebook already published in the Library -- create-if-absent only,
	// so a later hand edit (e.g. the known 4d12/5d12 fix) is never reverted.
	if err := ewrite.EnsureCanonicalSocioManuscript(ctx, pool); err != nil {
		log.Fatalf("kernel 78 socio manuscript seed failed: %v", err)
	}
	// Kernel 79: organizes the ruleset into Core Rulebook / Quickstart /
	// Niava Series (reparenting the pre-existing Core Rulebook publication
	// non-destructively) and seeds the two new manuscripts. Must run after
	// EnsureCanonicalSocioManuscript above, which creates the ruleset and
	// (on a fresh install) the Core Rulebook publication this reparents.
	if err := ewrite.EnsureSocioSeriesHierarchy(ctx, pool); err != nil {
		log.Fatalf("kernel 79 socio series hierarchy failed: %v", err)
	}
	if err := ewrite.EnsureQuickstartManuscript(ctx, pool); err != nil {
		log.Fatalf("kernel 79 quickstart manuscript seed failed: %v", err)
	}
	if err := ewrite.EnsureNiavaManuscript(ctx, pool); err != nil {
		log.Fatalf("kernel 79 niava manuscript seed failed: %v", err)
	}
	// Kernel 79: backfills ewrite_publication_assets for any publication
	// that predates that table (e.g. the live Core Rulebook, seeded under
	// Kernel 78) -- belt-and-suspenders alongside the direct reconcile
	// calls in the seed functions above and in every ordinary save.
	if err := ewrite.BackfillPublicationAssetRefs(ctx, pool); err != nil {
		log.Fatalf("kernel 79 publication asset backfill failed: %v", err)
	}
	// Kernel 79A: seeds the Skill Directory (100 catalogue skills, each
	// auto-matched to its exact Core Rulebook section by title). Must run
	// after EnsureSocioSeriesHierarchy/EnsureCanonicalSocioManuscript above,
	// which create the ruleset and its sections this reads. The catalogue is
	// bridged here rather than imported by ewrite directly -- see
	// ewrite.SkillCatalogueEntry's doc comment for the cycle it avoids.
	skillCatalogue := make([]ewrite.SkillCatalogueEntry, len(characters.Chapter4Skills))
	for i, s := range characters.Chapter4Skills {
		skillCatalogue[i] = ewrite.SkillCatalogueEntry{
			ID: s.ID, Name: s.Name, AttributeName: s.AttributeName, CardDescription: s.CardDescription,
		}
	}
	if err := ewrite.EnsureSkillDirectory(ctx, pool, skillCatalogue); err != nil {
		log.Fatalf("kernel 79a skill directory seed failed: %v", err)
	}
	// Kernel 73: must run after the venue bootstrap above -- migration 057
	// seeds the same Courtyard Scene for existing databases, but on a fresh
	// install migrations run before catharsis exists as a venue row.
	if err := merchant.EnsureCourtyardScene(ctx, pool); err != nil {
		log.Fatalf("kernel 73 courtyard scene bootstrap failed: %v", err)
	}
	// Kernel 73A: composition is no longer projected into a session via a
	// separate action/hook -- world.LoadVenueSnapshot (backend/internal/
	// world/snapshot.go) now resolves the current placement's Base+Show
	// composition fresh on every snapshot read, and both
	// handleShowCurrentSceneWithLiveBridge and cues.HandleCueGo already broadcast
	// network.BroadcastShowStageInvalidation on a successful current-Scene
	// change, which every connected client already refetches on. No
	// wiring needed here beyond those two existing broadcast call sites.
	characters.SetMaxCharacterCardsPerAccount(parseIntEnv("CHARACTER_ACCOUNT_LIMIT", 50))
	if err := identity.EnsureKernel39DiscordGatewaySurface(ctx, pool); err != nil {
		log.Fatalf("kernel 39 discord gateway bootstrap failed: %v", err)
	}
	if err := assets.EnsureKernel49WarehouseStorageSurface(ctx, pool); err != nil {
		log.Fatalf("kernel 49 warehouse bootstrap failed: %v", err)
	}
	if err := playerprofile.EnsureKernel61PlayerWorkbookSurface(ctx, pool); err != nil {
		log.Fatalf("kernel 61 player workbook bootstrap failed: %v", err)
	}
	if err := storyboards.EnsureKernel81AStoryboardSlugsSurface(ctx, pool); err != nil {
		log.Fatalf("kernel 81a storyboard slug backfill failed: %v", err)
	}
	if _, err := playerrelationships.LoadCatalogue(); err != nil {
		log.Fatalf("kernel 62 relationship catalogue invalid: %v", err)
	}

	hub := network.NewHub()
	discordAudioPresenceStore := identity.NewDiscordAudioPresenceStore()
	venueCoordination := venuecoordination.NewRegistry()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		if err := identity.ReconcileDiscordBootstrap(ctx, pool, discordServerLinkConfig); err != nil {
			log.Printf("discord bootstrap reconcile failed: %v", err)
		}
	}()

	go network.RunDiscordGatewayWorker(context.Background(), pool, hub, discordAudioPresenceStore, discordServerLinkConfig, discordGatewayConfig)

	// Kernel 77 K77-05: revoking a session must eventually disconnect any
	// WebSocket already open under it, not just block new connections.
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			hub.RevalidateSessions(ctx, pool)
			cancel()
		}
	}()

	// Kernel 77 §7.5: expired export archives must be cleaned up even if no
	// one ever checks their status again after the link goes stale.
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			n, err := identity.SweepExpiredExports(ctx, pool)
			cancel()
			if err != nil {
				log.Printf("export sweep failed: %v", err)
			} else if n > 0 {
				log.Printf("export sweep removed %d expired archive(s)", n)
			}
		}
	}()

	access.SetThirdPlaceReadinessChecker(func(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
		result, err := playerprofile.TrailerFaceReady(ctx, pool, userID)
		if err != nil {
			return false, err
		}
		return result.Ready, nil
	})

	identity.OnBootstrapAccount = firstrun.BootstrapFirstOperator

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"service": "victory-backend",
			"time":    time.Now().UTC(),
		})
	})

	// Unauthenticated like /health, and just as safe to expose: a bare
	// count carries no session/venue/user identity. Kernel 100's Windows
	// launcher calls this before applying a backend-changing update, to
	// avoid restarting mid-Show -- a scheduled 2:30am-local apply window
	// still checks this first, since a Show can genuinely be live then.
	//
	// the-cave is excluded: migration 004_seed_session.sql unconditionally
	// seeds it a permanent 'live' session on every fresh database, but
	// the-cave itself is "a deliberate, permanently hidden DOM-only test
	// harness" (see the-cave's own comment above, Kernel 70A) -- not a
	// real Show anyone is ever actually in. Left in, this count was never
	// once zero on any fresh install, so no backend-changing update could
	// ever pass this check -- confirmed on real hardware, 2026-09-09:
	// every manual "Check for Updates" click reported "a Show looks like
	// it's in progress" indefinitely on an install that had never run one.
	mux.HandleFunc("GET /api/system/live-sessions", func(w http.ResponseWriter, r *http.Request) {
		var liveCount int
		err := pool.QueryRow(r.Context(), `
			SELECT COUNT(*)
			FROM sessions s
			JOIN venues v ON v.id = s.venue_id
			WHERE s.status IN ('rehearsal', 'live')
			  AND v.slug != 'the-cave'
		`).Scan(&liveCount)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "could not check live sessions"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         true,
			"live_count": liveCount,
		})
	})

	// Unauthenticated like /health: a public base URL carries no identity
	// of its own. Confirmed on real hardware as a real bug: an invite link
	// built client-side from window.location.origin is always "localhost"
	// for an Operator browsing their own machine -- useless to anyone
	// else, since localhost on the recipient's machine means their
	// machine, not the Operator's. The backend has no way to know its own
	// reachable address on its own, so the Windows launcher tells it where
	// to look (PUBLIC_URL_FILE) -- read fresh on every request rather than
	// cached, since a Quick Tunnel's address changes on every restart.
	// Empty/unset PUBLIC_URL_FILE (every non-Windows deployment, or
	// TUNNEL_MODE=off) reports public_url: null; callers fall back to
	// their own request origin in that case.
	mux.HandleFunc("GET /api/system/public-url", func(w http.ResponseWriter, r *http.Request) {
		publicURL := ""
		if path := strings.TrimSpace(os.Getenv("PUBLIC_URL_FILE")); path != "" {
			if data, err := os.ReadFile(path); err == nil {
				publicURL = strings.TrimSpace(string(data))
			}
		}
		var publicURLValue any
		if publicURL != "" {
			publicURLValue = publicURL
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         true,
			"public_url": publicURLValue,
		})
	})

	// Kernel 72: every endpoint that accepts a credential guess shares one
	// per-IP token bucket (burst 10, refill 10/min). Ordinary API routes are
	// deliberately unthrottled.
	credentialLimiter := ratelimit.New(10, 10)
	// Kernel 77 K77-06: ordinary API routes were deliberately unthrottled
	// beyond the Kernel-76 body-size cap. This is a more generous bucket
	// than credentialLimiter for legitimate higher-frequency in-session
	// actions (chat, note cards, uploads, command execution) -- bounded
	// against flooding, not tuned to make live play feel rate-limited.
	actionLimiter := ratelimit.New(30, 60)
	mux.HandleFunc("/api/auth/signup", ratelimit.Middleware(credentialLimiter, identity.HandleSignup(pool, secureCookie, discordOAuthConfig)))
	mux.HandleFunc("/api/auth/login", ratelimit.Middleware(credentialLimiter, identity.HandleLogin(pool, secureCookie)))
	mux.HandleFunc("/api/auth/logout", identity.HandleLogout(pool, secureCookie))
	mux.HandleFunc("/api/auth/providers", identity.HandleDiscordOAuthProviders(discordOAuthConfig))
	mux.HandleFunc("GET /auth/discord/start", identity.HandleDiscordOAuthStart(pool, discordOAuthConfig))
	mux.HandleFunc("GET /auth/discord/callback", identity.HandleDiscordOAuthCallback(pool, discordOAuthConfig, secureCookie))
	mux.HandleFunc("GET /api/auth/discord/start", identity.HandleDiscordOAuthStart(pool, discordOAuthConfig))
	mux.HandleFunc("GET /api/auth/discord/callback", identity.HandleDiscordOAuthCallback(pool, discordOAuthConfig, secureCookie))
	mux.HandleFunc("/api/auth/password-reset/request", ratelimit.Middleware(credentialLimiter, identity.HandleForgotPassword(pool, forgotPasswordConfigFromEnv())))
	mux.HandleFunc("/api/auth/password-reset/confirm", ratelimit.Middleware(credentialLimiter, identity.HandleResetPassword(pool, secureCookie)))
	mux.HandleFunc("GET /api/invites/authority", identity.HandleInviteAuthority(pool))
	// Kernel 96: neither had any rate limit at all. Invite-accept
	// specifically creates a new account -- the exact "signup/invite
	// acceptance" category spec §26 calls out as abuse-prone -- and
	// invite creation is a sensitive, low-frequency action for any
	// legitimate Operator, so the same credential-guessing bucket fits
	// both.
	mux.HandleFunc("/api/invites", ratelimit.Middleware(credentialLimiter, identity.HandleCreateInvite(pool)))
	mux.HandleFunc("GET /api/invites/preview", identity.HandlePreviewInvite(pool))
	mux.HandleFunc("/api/invites/accept", ratelimit.Middleware(credentialLimiter, identity.HandleAcceptInvite(pool, secureCookie)))
	mux.HandleFunc("GET /api/discord/server-link/status", identity.HandleDiscordServerLinkStatus(pool, discordServerLinkConfig))
	mux.HandleFunc("/api/discord/server/bootstrap", identity.HandleDiscordServerBootstrap(pool, discordServerLinkConfig))
	mux.HandleFunc("GET /api/discord/gateway/status", identity.HandleDiscordGatewayStatus(pool, discordGatewayConfig))
	mux.HandleFunc("GET /api/discord/audio/status", identity.HandleDiscordAudioStatus(pool, discordServerLinkConfig, discordAudioPresenceStore))
	mux.HandleFunc("/api/discord/gateway/debug", identity.HandleDiscordGatewayDebug(pool))
	mux.HandleFunc("GET /auth/discord/server/install", identity.HandleDiscordServerInstall(pool, discordServerLinkConfig))
	mux.HandleFunc("GET /auth/discord/server/callback", identity.HandleDiscordServerCallback(pool, discordServerLinkConfig))
	mux.HandleFunc("POST /api/discord/server/unlink", identity.HandleDiscordServerUnlink(pool, discordServerLinkConfig))
	mux.HandleFunc("GET /api/discord/channel-mapping/status", identity.HandleDiscordChannelMappingStatus(pool, discordServerLinkConfig))
	mux.HandleFunc("POST /api/discord/channel-mapping/repair", identity.HandleDiscordChannelMappingRepair(pool, discordServerLinkConfig))
	mux.HandleFunc("POST /api/discord/interactions", identity.HandleDiscordInteractions(pool, discordServerLinkConfig))
	mux.HandleFunc("GET /api/discord/mic/status", identity.HandleDiscordMicStatus(pool, discordServerLinkConfig))
	mux.HandleFunc("POST /api/discord/mic/register", identity.HandleDiscordMicRegister(pool, discordServerLinkConfig))
	mux.HandleFunc("POST /api/discord/mic/control", identity.HandleDiscordMicControl(pool, discordServerLinkConfig))
	mux.HandleFunc("POST /api/session/control", network.HandleSessionControl(hub, pool))
	mux.HandleFunc("/api/requests/create", identity.HandleCreatePermissionRequest(pool))
	mux.HandleFunc("/api/requests/mine", identity.HandleListMyPermissionRequests(pool))
	mux.HandleFunc("/api/requests/incoming", identity.HandleListIncomingPermissionRequests(pool))
	mux.HandleFunc("/api/requests/respond", identity.HandleRespondPermissionRequest(pool))
	mux.HandleFunc("/api/productions", identity.HandleProductionsCollection(pool))
	mux.HandleFunc("/api/account/me", identity.HandleAccountMe(pool))
	mux.HandleFunc("/api/account/email", ratelimit.Middleware(credentialLimiter, identity.HandleUpdateAccountEmail(pool)))
	mux.HandleFunc("/api/account/email/verify", ratelimit.Middleware(credentialLimiter, identity.HandleRequestEmailVerification(pool, forgotPasswordConfigFromEnv())))
	mux.HandleFunc("/api/account/email/verify/confirm", ratelimit.Middleware(credentialLimiter, identity.HandleConfirmEmailVerification(pool)))
	mux.HandleFunc("/api/account/deletion-plan", ratelimit.Middleware(credentialLimiter, identity.HandleAccountDeletionPlan(pool)))
	mux.HandleFunc("/api/account/delete", ratelimit.Middleware(credentialLimiter, identity.HandleAccountDelete(pool, storageRoot)))
	mux.HandleFunc("/api/account/export", ratelimit.Middleware(credentialLimiter, identity.HandleExportCollection(pool, storageRoot, exportsRoot)))
	mux.HandleFunc("/api/account/export/status", identity.HandleExportStatus(pool))
	mux.HandleFunc("/api/account/export/download", identity.HandleExportDownload(pool))
	mux.HandleFunc("/api/operator/backup-status", identity.HandleBackupStatus(pool))
	mux.HandleFunc("/api/session/me", identity.HandleMe(pool))
	mux.HandleFunc("/api/profiles/me", profiles.HandleGetMyProfile(pool))
	mux.HandleFunc("/api/profiles/public", profiles.HandleGetPublicProfile(pool))
	mux.HandleFunc("/api/profiles/me/save", profiles.HandleSaveMyProfile(pool))
	mux.HandleFunc("/api/profiles/me/publish", profiles.HandlePublishMyProfile(pool))
	mux.HandleFunc("/api/profiles/admin/save", profiles.HandleAdminSaveProfile(pool))
	mux.HandleFunc("/api/profiles/admin/publish", profiles.HandleAdminPublishProfile(pool))
	playerProfileNotify := func(ctx context.Context, userID string, changed []string) {
		network.BroadcastPlayerProfileProjectionInvalidation(ctx, hub, pool, userID, changed)
	}
	mux.HandleFunc("/api/player-profile/catalogue", playerprofile.HandleCatalogue(pool))
	mux.HandleFunc("/api/player-profile/me", playerprofile.HandleOwnerWorkbook(pool))
	mux.HandleFunc("/api/player-profile/me/face-readiness", playerprofile.HandleFaceReadiness(pool))
	mux.HandleFunc("/api/player-profile/face-visibility", playerprofile.HandleFaceVisibility(pool, playerProfileNotify))
	mux.HandleFunc("/api/player-profile/face-priority", playerprofile.HandleFacePriority(pool, playerProfileNotify))
	mux.HandleFunc("/api/player-profile/stage-name", playerprofile.HandleStageName(pool, playerProfileNotify))
	mux.HandleFunc("/api/player-profile/pages/", playerprofile.HandlePageCommit(pool, playerProfileNotify))
	mux.HandleFunc("/api/player-profile/events/", playerprofile.HandleDeleteEvent(pool, playerProfileNotify))
	mux.HandleFunc("/api/player-profile/", playerprofile.HandleSocialFace(pool))
	mux.HandleFunc("/api/player-relationships/catalogue", playerrelationships.HandleCatalogue(pool))
	// Kernel 95 §13: split so the POST (create -- one new row per distinct
	// pair, but nothing stopped one account from creating many rows fast)
	// picks up the same actionLimiter already applied to other
	// creates-a-thing endpoints (messages, note-cards, audience notes);
	// GET (listing your own relationships) stays unthrottled.
	mux.HandleFunc("GET /api/player-relationships", playerrelationships.HandleCollection(pool))
	mux.HandleFunc("POST /api/player-relationships", ratelimit.Middleware(actionLimiter, playerrelationships.HandleCollection(pool)))
	mux.HandleFunc("/api/player-relationships/", playerrelationships.HandleByID(pool))
	mux.HandleFunc("/api/third-place/headshots", thirdplace.HandleCollection(pool))
	mux.HandleFunc("/api/third-place/headshots/me", thirdplace.HandleMe(pool))
	mux.HandleFunc("/api/third-place/headshots/me/history", thirdplace.HandleMyHistory(pool))
	mux.HandleFunc("GET /api/show-runs", showruns.HandleCollection(pool))
	mux.HandleFunc("POST /api/show-runs", showruns.HandleCollection(pool))
	mux.HandleFunc("GET /api/show-runs/{id}", showruns.HandleByID(pool))
	mux.HandleFunc("PATCH /api/show-runs/{id}", showruns.HandleByID(pool))
	mux.HandleFunc("POST /api/show-runs/{id}/archive", showruns.HandleArchive(pool))
	mux.HandleFunc("POST /api/show-runs/{id}/unarchive", showruns.HandleUnarchive(pool))
	mux.HandleFunc("GET /api/show-runs/{id}/roster", showruns.HandleRosterCollection(pool))
	mux.HandleFunc("POST /api/show-runs/{id}/roster", showruns.HandleRosterCollection(pool))
	mux.HandleFunc("PATCH /api/show-runs/{id}/roster/{member_id}", showruns.HandleRosterUpdate(pool))
	mux.HandleFunc("DELETE /api/show-runs/{id}/roster/{member_id}", showruns.HandleRosterRemove(pool))
	mux.HandleFunc("POST /api/show-runs/{id}/roster/self-join", showruns.HandleSelfJoin(pool))
	mux.HandleFunc("POST /api/show-runs/{id}/roster/self-join-as-player", showruns.HandleSelfJoinAsPlayer(pool))
	mux.HandleFunc("GET /api/show-runs/{id}/roster/me", showruns.HandleMyRosterMember(pool))
	mux.HandleFunc("POST /api/show-runs/{id}/roster/me/character", showruns.HandleSelectCharacter(pool))
	mux.HandleFunc("GET /api/show-runs/{id}/audience-program", showruns.HandleAudienceProgram(pool))
	mux.HandleFunc("POST /api/show-runs/{id}/blocks", showruns.HandleBlock(pool))
	mux.HandleFunc("DELETE /api/show-runs/{id}/blocks/{block_id}", showruns.HandleUnblock(pool))
	mux.HandleFunc("POST /api/show-runs/{id}/tickets/request", tickets.HandleRequest(pool))
	mux.HandleFunc("POST /api/show-runs/{id}/tickets/invite", tickets.HandleInvite(pool))
	mux.HandleFunc("GET /api/show-runs/{id}/tickets/incoming", tickets.HandleListIncoming(pool))
	mux.HandleFunc("POST /api/tickets/{ticket_id}/punch", tickets.HandlePunch(pool))
	mux.HandleFunc("POST /api/tickets/{ticket_id}/decline", tickets.HandleDecline(pool))
	mux.HandleFunc("POST /api/tickets/{ticket_id}/withdraw", tickets.HandleWithdraw(pool))
	mux.HandleFunc("GET /api/tickets/mine", tickets.HandleListMine(pool))
	mux.HandleFunc("GET /api/show-runs/{show_run_id}/shows", shows.HandleShowRunShows(pool))
	mux.HandleFunc("POST /api/show-runs/{show_run_id}/shows", shows.HandleShowRunShows(pool))
	mux.HandleFunc("GET /api/shows/{show_id}", shows.HandleShowByID(pool))
	mux.HandleFunc("PATCH /api/shows/{show_id}", shows.HandleShowByID(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/archive", shows.HandleShowArchive(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/program", shows.HandleShowProgram(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/sessions/{session_id}/link", shows.HandleShowSessionLink(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/sessions/{session_id}/unlink", shows.HandleShowSessionUnlink(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/sessions/start", shows.HandleShowSessionStart(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/current-scene", handleShowCurrentSceneWithLiveBridge(pool, hub))
	mux.HandleFunc("PATCH /api/shows/{show_id}/short-code", shows.HandleUpdateShortCode(pool))
	mux.HandleFunc("GET /api/shows/by-code", shows.HandleResolveByShortCode(pool))
	// Kernel 92's "Showing" (scheduled performance instance) is a distinct
	// concept from Kernel 22's `showings` table/HTTP namespace (the live
	// audience-visibility wrapper reviewed at GET /api/showings below) --
	// mounted under /api/showtime/ to avoid colliding with that existing
	// route, not because it isn't also a `shows` domain operation.
	mux.HandleFunc("GET /api/showtime/showings", shows.HandleShowingsCollection(pool))
	mux.HandleFunc("POST /api/showtime/showings", shows.HandleShowingsCollection(pool))
	mux.HandleFunc("POST /api/showtime/control", showtime.HandleShowtimeControl(pool, discordServerLinkConfig))
	mux.HandleFunc("POST /api/audience-admissions", audienceadmission.HandleIssue(pool))
	mux.HandleFunc("GET /api/audience-admissions", audienceadmission.HandleList(pool))
	mux.HandleFunc("GET /api/audience-config", audienceprojection.HandleGet(pool))
	mux.HandleFunc("PUT /api/audience-config", audienceprojection.HandleUpdate(pool))
	mux.HandleFunc("GET /api/scenes", scenes.HandleScenesCollection(pool))
	mux.HandleFunc("POST /api/scenes", scenes.HandleScenesCollection(pool))
	mux.HandleFunc("GET /api/scenes/{scene_id}", scenes.HandleSceneByID(pool))
	mux.HandleFunc("PATCH /api/scenes/{scene_id}", scenes.HandleSceneByID(pool))
	mux.HandleFunc("POST /api/scenes/{scene_id}/archive", scenes.HandleSceneArchive(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/scenes", scenes.HandleShowScenesCollection(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/scenes", scenes.HandleShowScenesCollection(pool))
	mux.HandleFunc("PATCH /api/shows/{show_id}/scenes/{placement_id}", scenes.HandleShowScenePlacementByID(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/scenes/{placement_id}/archive", scenes.HandleShowScenePlacementArchive(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/scenes/program", scenes.HandleShowSceneProgram(pool))
	mux.HandleFunc("GET /api/scenes/{scene_id}/stage-elements", scenes.HandleSceneStageElementsCollection(pool))
	mux.HandleFunc("POST /api/scenes/{scene_id}/stage-elements", scenes.HandleSceneStageElementsCollection(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/scenes/{placement_id}/stage-elements", scenes.HandlePlacementStageElementsCollection(pool))
	mux.HandleFunc("PATCH /api/stage-elements/{element_id}", scenes.HandleStageElementByID(pool))
	mux.HandleFunc("DELETE /api/stage-elements/{element_id}", scenes.HandleStageElementByID(pool))
	mux.HandleFunc("POST /api/stage-elements/{element_id}/binding", scenes.HandleStageElementBinding(pool))
	mux.HandleFunc("DELETE /api/stage-elements/{element_id}/binding", scenes.HandleStageElementBinding(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/scenes/{placement_id}/stage-composition", scenes.HandlePlacementStageComposition(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/scenes/{placement_id}/update-current-scene", scenes.HandleUpdateCurrentScene(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/scenes/{placement_id}/save-as-new-scene", scenes.HandleSaveAsNewScene(pool))
	// Kernel 85: Show Cohorts -- Director-run sustained-play grouping.
	mux.HandleFunc("GET /api/shows/{show_id}/cohorts", cohorts.HandleCohortsCollection(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/cohorts", cohorts.HandleCohortsCollection(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/cohorts/{cohort_id}/archive", cohorts.HandleCohortArchive(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/cohorts/{cohort_id}/assignments", cohorts.HandleCohortAssignment(pool))
	mux.HandleFunc("DELETE /api/shows/{show_id}/cohorts/assignments/{user_id}", cohorts.HandleCohortUnassign(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/cohorts/{cohort_id}/current-scene", cohorts.HandleCohortCurrentScene(pool, hub))
	// Kernel 87: Cartograph shared drawing objects + measured tabletop.
	mux.HandleFunc("GET /api/sessions/{session_id}/drawing-objects", drawing.HandleObjectsCollection(pool, hub, venueCoordination))
	mux.HandleFunc("POST /api/sessions/{session_id}/drawing-objects", drawing.HandleObjectsCollection(pool, hub, venueCoordination))
	mux.HandleFunc("PATCH /api/sessions/{session_id}/drawing-objects/{object_id}", drawing.HandleObjectItem(pool, hub))
	mux.HandleFunc("DELETE /api/sessions/{session_id}/drawing-objects/{object_id}", drawing.HandleObjectItem(pool, hub))
	mux.HandleFunc("POST /api/sessions/{session_id}/drawing-objects/{object_id}/lock", drawing.HandleObjectLock(pool, hub))
	mux.HandleFunc("POST /api/sessions/{session_id}/drawing-objects/{object_id}/z-order", drawing.HandleObjectZOrder(pool, hub))
	mux.HandleFunc("GET /api/shows/{show_id}/drawing-settings", drawing.HandleSettings(pool, hub))
	mux.HandleFunc("PUT /api/shows/{show_id}/drawing-settings", drawing.HandleSettings(pool, hub))
	mux.HandleFunc("GET /api/drawing/stamps", drawing.HandleStampPalette())
	mux.HandleFunc("GET /api/sessions/{session_id}/drawing-coordination/{role}", drawing.HandleCoordination(pool, venueCoordination, hub))
	mux.HandleFunc("POST /api/sessions/{session_id}/drawing-coordination/{role}", drawing.HandleCoordination(pool, venueCoordination, hub))
	// Kernel 85: Socio Game Status -- HP pools + status effects.
	mux.HandleFunc("GET /api/socio/statuses", socio.HandleStatusRegistry(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/cohorts/{cohort_id}/game-status", socio.HandleGameStatus(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/characters/{character_card_id}/socio/pools/{pool_key}", socio.HandleCharacterPool(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/characters/{character_card_id}/socio/statuses", socio.HandleCharacterStatus(pool))
	mux.HandleFunc("DELETE /api/shows/{show_id}/characters/{character_card_id}/socio/statuses/{status_key}", socio.HandleCharacterStatus(pool))
	// Kernel 88: Socio Guided Play Surface -- Fate, Stance, blank state
	// flags, tiered Player/Director projection, Current Turn, and the
	// Interrupt/Help pending-action stack. Reuses the same venueCoordination
	// registry Storyboards/drawing already share (Kernel 83) -- it's
	// venue-agnostic and Socio's session-ID scheme ("showID:cohortID")
	// can't collide with those venues' own IDs.
	mux.HandleFunc("GET /api/shows/{show_id}/characters/{character_card_id}/socio/view", socio.HandleSocioProjection(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/characters/{character_card_id}/socio/mechanics", socio.HandleCharacterMechanics(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/characters/{character_card_id}/socio/fate/spend", socio.HandleFateSpend(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/characters/{character_card_id}/socio/fate/award", socio.HandleFateAward(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/characters/{character_card_id}/socio/fate/creation-mode", socio.HandleFateCreationMode(pool))
	mux.HandleFunc("GET /api/socio/stances", socio.HandleStanceRegistry(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/characters/{character_card_id}/socio/stance", socio.HandleCharacterStance(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/characters/{character_card_id}/socio/flags", socio.HandleCharacterFlags(pool))
	mux.HandleFunc("DELETE /api/shows/{show_id}/characters/{character_card_id}/socio/flags/{flag_id}", socio.HandleCharacterFlags(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/cohorts/{cohort_id}/socio/current-turn", socio.HandleCurrentTurn(pool, venueCoordination))
	mux.HandleFunc("POST /api/shows/{show_id}/cohorts/{cohort_id}/socio/current-turn", socio.HandleCurrentTurn(pool, venueCoordination))
	mux.HandleFunc("GET /api/shows/{show_id}/socio/pending-actions", socio.HandlePendingActions(pool, venueCoordination))
	mux.HandleFunc("POST /api/shows/{show_id}/socio/pending-actions", socio.HandlePendingActions(pool, venueCoordination))
	mux.HandleFunc("POST /api/shows/{show_id}/socio/pending-actions/{action_id}/interrupt", socio.HandleInterrupt(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/socio/pending-actions/{action_id}/resolve", socio.HandleInterruptResolve(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/socio/pending-actions/{action_id}/cancel", socio.HandlePendingActionCancel(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/scenes/{placement_id}/cues", cues.HandlePlacementCuesCollection(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/scenes/{placement_id}/cues", cues.HandlePlacementCuesCollection(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/scenes/{placement_id}/player-cues", cues.HandlePlacementPlayerCues(pool))
	mux.HandleFunc("GET /api/cues/{cue_id}", cues.HandleCueByID(pool))
	mux.HandleFunc("PATCH /api/cues/{cue_id}", cues.HandleCueByID(pool))
	mux.HandleFunc("POST /api/cues/{cue_id}/go", cues.HandleCueGo(pool, hub))
	mux.HandleFunc("GET /api/shows/{show_id}/scenes/{placement_id}/participant-interactions", merchant.HandlePlacementInteractionsCollection(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/scenes/{placement_id}/participant-interactions", merchant.HandlePlacementInteractionsCollection(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/scenes/{placement_id}/player-interactions", merchant.HandlePlacementPlayerInteractions(pool))
	mux.HandleFunc("PATCH /api/participant-interactions/{interaction_id}", merchant.HandleInteractionByID(pool))
	mux.HandleFunc("POST /api/participant-interactions/{interaction_id}/open", merchant.HandleInteractionOpen(pool))
	mux.HandleFunc("GET /api/participant-interactions/{interaction_id}/preview", merchant.HandleInteractionPreview(pool))
	mux.HandleFunc("POST /api/participant-interactions/{interaction_id}/stance", merchant.HandleInteractionStance(pool))
	mux.HandleFunc("GET /api/participant-interactions/{interaction_id}/haggle", merchant.HandleInteractionHaggle(pool))
	mux.HandleFunc("POST /api/participant-interactions/{interaction_id}/haggle", merchant.HandleInteractionHaggle(pool))
	mux.HandleFunc("POST /api/participant-interactions/{interaction_id}/purchase", merchant.HandleInteractionPurchase(pool, hub))
	mux.HandleFunc("GET /api/characters/{character_card_id}/inventory", merchant.HandleCharacterInventory(pool))
	// Kernel 74: the Player-controlled tutorial tail. /complete is "Leave
	// Kessa's Stall", /submit is the freeform door intention, and
	// /dialogue/{action} is Ra's guided conversation. None of them has a
	// Director GO counterpart -- that absence is the point.
	mux.HandleFunc("POST /api/participant-interactions/{interaction_id}/complete", merchant.HandleInteractionComplete(pool, hub))
	mux.HandleFunc("POST /api/participant-interactions/{interaction_id}/submit", merchant.HandleInteractionSubmit(pool, hub))
	mux.HandleFunc("POST /api/participant-interactions/{interaction_id}/dialogue/{action}", merchant.HandleInteractionDialogue(pool, hub))
	// Kernel 75: the tutorial's ending. /tutorial/continue is the Player's
	// Continue press after watching Ra open the gate; /tutorial/completion
	// is the reopen path and records nothing. Registered under the existing
	// participant-interactions prefix so they inherit the same
	// ResolveEligibleContext gate as every other participant action.
	mux.HandleFunc("POST /api/participant-interactions/{interaction_id}/tutorial/continue", merchant.HandleTutorialContinue(pool, hub))
	mux.HandleFunc("GET /api/participant-interactions/{interaction_id}/tutorial/completion", merchant.HandleTutorialCompletion(pool))
	// Story So Far. Deliberately under /api/characters/ rather than
	// /api/character-cards/, which is a PREFIX handler below and would
	// swallow a sibling route.
	mux.HandleFunc("GET /api/characters/{character_card_id}/story-so-far", merchant.HandleCharacterStorySoFar(pool))
	mux.HandleFunc("PATCH /api/story-events/{event_id}", merchant.HandleStoryEventVisibility(pool))
	// Director-authored Character moments (S1.9/S6.4). Reuses the existing
	// character_journals store; a separate route so the Player's own journal
	// handler keeps its stricter "your active Character only" rule.
	mux.HandleFunc("POST /api/characters/{character_card_id}/director-journal", merchant.HandleDirectorCharacterJournal(pool))
	mux.HandleFunc("GET /api/player-recognition/me", merchant.HandleMyRecognition(pool))
	// Aftercare (Kernel 75 S8). Show-keyed rather than interaction-keyed so
	// a Player can write it later from the Greenroom, when no Program is on
	// screen and possibly no Session is running.
	mux.HandleFunc("GET /api/shows/{show_id}/aftercare", merchant.HandleShowAftercare(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/aftercare", merchant.HandleShowAftercare(pool))
	mux.HandleFunc("PUT /api/shows/{show_id}/aftercare/draft", merchant.HandleShowAftercareDraft(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/aftercare/skip", merchant.HandleShowAftercareSkip(pool))
	// Directors+ review (S9). Read-only: no POST/PATCH/DELETE exists for
	// these paths. The two gates differ on purpose -- reading the table is
	// backstage visibility (CanViewBackstage), while exporting takes
	// Player-written reflection off the platform (CanManageShowRun).
	// Kernel 89 §17: Send Aftercare, the Director's manual delivery of the
	// Kernel 75 form above. Deliberately NOT reached by /showtime end --
	// see backend/internal/merchant/kernel89_aftercare_send.go.
	mux.HandleFunc("POST /api/shows/{show_id}/aftercare/send", merchant.HandleShowAftercareSend(pool, hub))
	mux.HandleFunc("GET /api/shows/{show_id}/aftercare-review", merchant.HandleShowAftercareReview(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/aftercare-review.csv", merchant.HandleShowAftercareReviewCSV(pool))
	// Dialogue authoring (Kernel 75). Reads take CanViewBackstage; writes
	// take CanManageShowRun -- stricter than interaction editing, because an
	// NPC's prose is canon every Player reads verbatim.
	mux.HandleFunc("GET /api/locations/{location_id}/dialogue-packets", merchant.HandleLocationDialoguePackets(pool))
	mux.HandleFunc("GET /api/dialogue-packets/{packet_id}", merchant.HandleDialoguePacketByID(pool))
	mux.HandleFunc("PATCH /api/dialogue-packets/{packet_id}", merchant.HandleDialoguePacketByID(pool))
	mux.HandleFunc("GET /api/dialogue-packets/{packet_id}/revisions", merchant.HandleDialoguePacketRevisions(pool))
	mux.HandleFunc("PATCH /api/dialogue-topics/{topic_id}", merchant.HandleDialogueTopicByID(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/tutorial-progress", merchant.HandleShowTutorialProgress(pool))
	mux.HandleFunc("DELETE /api/shows/{show_id}/local-projections/{user_id}", merchant.HandleClearLocalProjection(pool, hub))
	mux.HandleFunc("POST /api/shows/{show_id}/prepare-locked-courtyard-opening", merchant.HandlePrepareLockedCourtyardOpening(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/kessa-reachability", merchant.HandleKessaReachabilityDiagnostics(pool))
	mux.HandleFunc("GET /api/venues/{venue_slug}/equipment", merchant.HandleVenueEquipmentCollection(pool))
	mux.HandleFunc("POST /api/venues/{venue_slug}/equipment", merchant.HandleVenueEquipmentCollection(pool))
	mux.HandleFunc("PATCH /api/equipment/{equipment_item_id}", merchant.HandleEquipmentItemByID(pool))
	// Kernel 89 §9: merchant authoring over Kernel 73's canonical
	// merchant_packets model, with stock chosen from the one existing
	// equipment_items corpus (returned alongside as "catalog").
	mux.HandleFunc("GET /api/shows/{show_id}/merchant-packets", merchant.HandleShowMerchantPackets(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/merchant-packets", merchant.HandleShowMerchantPackets(pool))
	mux.HandleFunc("PATCH /api/merchant-packets/{packet_id}", merchant.HandleMerchantPacketByID(pool))
	// Kernel 89 §7/§10/§26: Director preparations. Director+ only end to
	// end -- there is no Player-facing read path in this family at all.
	mux.HandleFunc("GET /api/announcement-styles", directorprep.HandleAnnouncementStyles(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/director-preparations", directorprep.HandleShowPreparations(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/director-preparations", directorprep.HandleShowPreparations(pool))
	mux.HandleFunc("PATCH /api/director-preparations/{preparation_id}", directorprep.HandlePreparationByID(pool))
	mux.HandleFunc("DELETE /api/director-preparations/{preparation_id}", directorprep.HandlePreparationByID(pool))

	// Kernel 90: canonical stage-object visibility/interaction state. Both
	// routes are Director+ only, the READ included -- the state set describes
	// what is hidden and from whom, so exposing it to a Player would hand
	// them exactly the metadata §35/§36 require be withheld.
	//
	// The Director gate and the projection-invalidation notifier are injected
	// rather than imported by stageobjects, and the reason is structural, not
	// stylistic: world/snapshot.go imports stageobjects for projection, while
	// shows -> network -> world means importing either showruns or network
	// inside stageobjects would close a real import cycle. Wiring them here
	// keeps stageobjects a leaf package and still leaves exactly one opinion
	// in the product about what Director+ means (directorprep.RequireDirector,
	// which itself defers to showruns.CanManageShowRun).
	stageObjectDirectorGate := func(ctx context.Context, actorUserID, showID string) error {
		return directorprep.RequireDirector(ctx, pool, actorUserID, showID)
	}
	stageObjectNotifier := func(ctx context.Context, showID string) {
		network.BroadcastShowStageInvalidation(ctx, hub, pool, showID, "stage_object_state_changed")
	}
	mux.HandleFunc("GET /api/shows/{show_id}/stage-object-states", stageobjects.HandleShowObjectStates(pool, stageObjectDirectorGate, stageObjectNotifier))
	mux.HandleFunc("POST /api/shows/{show_id}/stage-object-states", stageobjects.HandleShowObjectStates(pool, stageObjectDirectorGate, stageObjectNotifier))
	mux.HandleFunc("GET /api/shows/{show_id}/stage-object-scope-targets", stageobjects.HandleSupportedScopeTargets(pool, stageObjectDirectorGate))

	mux.HandleFunc("GET /api/characters/parentage-chart", characters.HandleParentageChart())
	mux.HandleFunc("GET /api/characters/chapter2-rules", characters.HandleChapter2Rules())
	mux.HandleFunc("GET /api/character-cards/me", characters.HandleMyCharacterCards(pool))
	mux.HandleFunc("POST /api/character-cards/parentage-roll", characters.HandleRequestParentageRoll(pool))
	mux.HandleFunc("POST /api/character-cards/coin-flip", characters.HandleCatharsisCoinFlip(pool))
	mux.HandleFunc("POST /api/character-cards/chapter2-roll", characters.HandleChapter2Roll(pool))
	mux.HandleFunc("GET /api/character-cards/chapter2-roll", characters.HandleChapter2RollStatus(pool))
	mux.HandleFunc("POST /api/character-cards/chapter2-stage", characters.HandleChapter2Stage(pool))
	mux.HandleFunc("GET /api/characters/chapter3-archetypes", characters.HandleChapter3Archetypes())
	mux.HandleFunc("POST /api/character-cards/chapter3-confirm", characters.HandleChapter3Confirm(pool))
	mux.HandleFunc("GET /api/character-cards/chapter4-group", characters.HandleChapter4Group(pool))
	mux.HandleFunc("POST /api/character-cards/chapter4-confirm", characters.HandleChapter4Confirm(pool))
	mux.HandleFunc("GET /api/characters/chapter4-courtyard", characters.HandleChapter4Courtyard())
	mux.HandleFunc("/api/character-cards", characters.HandleCreateCharacterCard(pool))
	mux.HandleFunc("/api/character-cards/", characters.HandleCharacterCardByID(pool, func(ctx context.Context, cardID string, changed []string, sourceEventID string) {
		network.BroadcastCharacterProjectionInvalidation(ctx, hub, pool, cardID, changed, sourceEventID)
	}))
	mux.HandleFunc("/api/character-workbooks/", characters.HandleCharacterWorkbookByID(pool, func(ctx context.Context, cardID string, changed []string, sourceEventID string) {
		network.BroadcastCharacterProjectionInvalidation(ctx, hub, pool, cardID, changed, sourceEventID)
	}))
	mux.HandleFunc("/api/character-journals", characters.HandleCharacterJournals(pool))
	mux.HandleFunc("GET /api/commands/available", network.HandleCommandsAvailable(pool))
	mux.HandleFunc("POST /api/commands/preview", network.HandleCommandsPreview(pool))
	mux.HandleFunc("POST /api/commands/execute", ratelimit.Middleware(actionLimiter, network.HandleCommandsExecute(hub, pool)))
	mux.HandleFunc("GET /api/characters/venue-sheet", network.HandleVenueCharacterSheet(pool))
	mux.HandleFunc("POST /api/character-card-permissions", characters.HandleGrantCharacterPermission(pool))
	mux.HandleFunc("POST /api/character-card-permissions/revoke", characters.HandleRevokeCharacterPermission(pool))
	mux.HandleFunc("GET /api/showings", showings.HandleReviewShowings(pool))
	mux.HandleFunc("GET /api/showings/{id}/review", showings.HandleReviewShowingByID(pool))
	mux.HandleFunc("GET /api/director-console/current", network.HandleDirectorConsoleCurrent(hub, pool))
	mux.HandleFunc("POST /api/showings/{id}/audience-view", network.HandleDirectorConsoleAudienceView(hub, pool))
	mux.HandleFunc("POST /api/showings/{id}/close", network.HandleDirectorConsoleCloseShowing(hub, pool))
	mux.HandleFunc("POST /api/showings/start", network.HandleDirectorConsoleStartShowing(hub, pool))
	mux.HandleFunc("POST /api/venues/{slug}/chat-policy", network.HandleDirectorConsoleChatPolicy(hub, pool))
	mux.HandleFunc("GET /api/messages", messages.HandleMessages(pool))
	mux.HandleFunc("POST /api/messages", ratelimit.Middleware(actionLimiter, messages.HandleMessages(pool)))
	mux.HandleFunc("GET /api/messages/{id}", messages.HandleMessageByID(pool))
	mux.HandleFunc("PATCH /api/messages/{id}", messages.HandleMessageByID(pool))
	mux.HandleFunc("DELETE /api/messages/{id}", messages.HandleMessageByID(pool))
	mux.HandleFunc("POST /api/note-cards", ratelimit.Middleware(actionLimiter, messages.HandleNoteCards(pool)))
	// Kernel 93 A19: Audience -> Director notes, Catharsis-scoped (see
	// audiencenotes' doc comment for why this isn't just a wider note-cards
	// route). Local mailbox delivery always works; Discord export is a
	// separate, best-effort batch action a Director triggers by hand.
	mux.HandleFunc("POST /api/session/catharsis/notes", ratelimit.Middleware(actionLimiter, audiencenotes.HandleSubmit(pool)))
	mux.HandleFunc("POST /api/session/catharsis/notes/export", ratelimit.Middleware(actionLimiter, network.HandleExportAudienceNotesToDiscord(pool, discordServerLinkConfig)))
	// Kernel 74 S8.1: Directors+ read the door-intention note inside
	// Catharsis, without opening Stage Management or the mailbox.
	mux.HandleFunc("GET /api/backstage-notes", messages.HandleBackstageNotes(pool))
	mux.HandleFunc("/api/index-cards", network.HandleIndexCardSave(hub, pool))

	// Kernel 78: eWrite -- Writer's Room authoring surface. All writes
	// behind the shared action limiter; every handler re-derives authority
	// server-side (Crew+ in scope, named grants), never from the client.
	mux.HandleFunc("GET /api/ewrite/tree", ewrite.HandleTree(pool))
	mux.HandleFunc("POST /api/ewrite/collections", ratelimit.Middleware(actionLimiter, ewrite.HandleCreateCollection(pool)))
	mux.HandleFunc("PATCH /api/ewrite/collections/{collection_id}", ratelimit.Middleware(actionLimiter, ewrite.HandleCollectionItem(pool)))
	mux.HandleFunc("DELETE /api/ewrite/collections/{collection_id}", ratelimit.Middleware(actionLimiter, ewrite.HandleCollectionItem(pool)))
	mux.HandleFunc("POST /api/ewrite/publications", ratelimit.Middleware(actionLimiter, ewrite.HandleCreatePublication(pool)))
	mux.HandleFunc("GET /api/ewrite/publications/{publication_id}", ewrite.HandlePublicationItem(pool))
	mux.HandleFunc("PATCH /api/ewrite/publications/{publication_id}", ratelimit.Middleware(actionLimiter, ewrite.HandlePublicationItem(pool)))
	mux.HandleFunc("DELETE /api/ewrite/publications/{publication_id}", ratelimit.Middleware(actionLimiter, ewrite.HandlePublicationItem(pool)))
	mux.HandleFunc("PUT /api/ewrite/publications/{publication_id}/source", ratelimit.Middleware(actionLimiter, ewrite.HandleSaveSource(pool)))
	mux.HandleFunc("POST /api/ewrite/publications/{publication_id}/publish", ratelimit.Middleware(actionLimiter, ewrite.HandlePublish(pool)))
	mux.HandleFunc("POST /api/ewrite/publications/{publication_id}/unpublish", ratelimit.Middleware(actionLimiter, ewrite.HandleUnpublish(pool)))
	mux.HandleFunc("POST /api/ewrite/publications/{publication_id}/import", ratelimit.Middleware(actionLimiter, ewrite.HandleImport(pool)))
	mux.HandleFunc("GET /api/ewrite/publications/{publication_id}/revisions", ewrite.HandleRevisionList(pool))
	mux.HandleFunc("GET /api/ewrite/publications/{publication_id}/editors", ewrite.HandleEditors(pool))
	mux.HandleFunc("POST /api/ewrite/publications/{publication_id}/editors", ratelimit.Middleware(actionLimiter, ewrite.HandleEditors(pool)))
	mux.HandleFunc("GET /api/ewrite/revisions/{revision_id}", ewrite.HandleRevisionItem(pool))
	mux.HandleFunc("POST /api/ewrite/preview", ratelimit.Middleware(actionLimiter, ewrite.HandlePreview(pool)))
	mux.HandleFunc("DELETE /api/ewrite/editors/{grant_id}", ratelimit.Middleware(actionLimiter, ewrite.HandleEditorItem(pool)))
	mux.HandleFunc("GET /api/ewrite/object-links", ewrite.HandleObjectLinks(pool))
	mux.HandleFunc("POST /api/ewrite/object-links", ratelimit.Middleware(actionLimiter, ewrite.HandleObjectLinks(pool)))
	mux.HandleFunc("DELETE /api/ewrite/object-links/{link_id}", ratelimit.Middleware(actionLimiter, ewrite.HandleObjectLinkItem(pool)))
	mux.HandleFunc("GET /api/ewrite/directories", ewrite.HandleDirectories(pool))
	mux.HandleFunc("GET /api/ewrite/directories/{directory_id}/entries", ewrite.HandleDirectoryEntries(pool))
	mux.HandleFunc("PUT /api/ewrite/directory-entries/{entry_id}/link", ratelimit.Middleware(actionLimiter, ewrite.HandleDirectoryEntryLink(pool)))

	// Kernel 80: Storyboards Core.
	mux.HandleFunc("GET /api/storyboards", storyboards.HandleBoards(pool))
	mux.HandleFunc("POST /api/storyboards", ratelimit.Middleware(actionLimiter, storyboards.HandleBoards(pool)))
	mux.HandleFunc("GET /api/storyboards/{board_id}", storyboards.HandleBoardItem(pool, hub, venueCoordination))
	mux.HandleFunc("PATCH /api/storyboards/{board_id}", ratelimit.Middleware(actionLimiter, storyboards.HandleBoardItem(pool, hub, venueCoordination)))
	mux.HandleFunc("DELETE /api/storyboards/{board_id}", ratelimit.Middleware(actionLimiter, storyboards.HandleBoardItem(pool, hub, venueCoordination)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/archive", ratelimit.Middleware(actionLimiter, storyboards.HandleBoardArchive(pool, hub)))
	mux.HandleFunc("GET /api/storyboards/{board_id}/export", storyboards.HandleBoardExport(pool))
	mux.HandleFunc("GET /api/storyboards/{board_id}/grants", storyboards.HandleGrants(pool, hub))
	mux.HandleFunc("POST /api/storyboards/{board_id}/grants", ratelimit.Middleware(actionLimiter, storyboards.HandleGrants(pool, hub)))
	mux.HandleFunc("DELETE /api/storyboards/{board_id}/grants/{grant_id}", ratelimit.Middleware(actionLimiter, storyboards.HandleGrantItem(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/columns", ratelimit.Middleware(actionLimiter, storyboards.HandleColumns(pool, hub)))
	mux.HandleFunc("PATCH /api/storyboards/{board_id}/columns/{column_id}", ratelimit.Middleware(actionLimiter, storyboards.HandleColumnItem(pool, hub)))
	mux.HandleFunc("DELETE /api/storyboards/{board_id}/columns/{column_id}", ratelimit.Middleware(actionLimiter, storyboards.HandleColumnItem(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/columns/reorder", ratelimit.Middleware(actionLimiter, storyboards.HandleColumnsReorder(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/bands", ratelimit.Middleware(actionLimiter, storyboards.HandleBands(pool, hub)))
	mux.HandleFunc("PATCH /api/storyboards/{board_id}/bands/{band_id}", ratelimit.Middleware(actionLimiter, storyboards.HandleBandItem(pool, hub)))
	mux.HandleFunc("DELETE /api/storyboards/{board_id}/bands/{band_id}", ratelimit.Middleware(actionLimiter, storyboards.HandleBandItem(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/bands/{band_id}/lock", ratelimit.Middleware(actionLimiter, storyboards.HandleBandLock(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/bands/{band_id}/collapse", ratelimit.Middleware(actionLimiter, storyboards.HandleBandCollapse(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/bands/reorder", ratelimit.Middleware(actionLimiter, storyboards.HandleBandsReorder(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/bands/{band_id}/rows/reorder", ratelimit.Middleware(actionLimiter, storyboards.HandleBandRowsReorder(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/rows", ratelimit.Middleware(actionLimiter, storyboards.HandleRows(pool, hub)))
	mux.HandleFunc("PATCH /api/storyboards/{board_id}/rows/{row_id}", ratelimit.Middleware(actionLimiter, storyboards.HandleRowItem(pool, hub)))
	mux.HandleFunc("DELETE /api/storyboards/{board_id}/rows/{row_id}", ratelimit.Middleware(actionLimiter, storyboards.HandleRowItem(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/rows/{row_id}/move", ratelimit.Middleware(actionLimiter, storyboards.HandleRowMove(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/cards", ratelimit.Middleware(actionLimiter, storyboards.HandleCards(pool, hub)))
	mux.HandleFunc("PATCH /api/storyboards/{board_id}/cards/{card_id}", ratelimit.Middleware(actionLimiter, storyboards.HandleCardItem(pool, hub)))
	mux.HandleFunc("DELETE /api/storyboards/{board_id}/cards/{card_id}", ratelimit.Middleware(actionLimiter, storyboards.HandleCardItem(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/cards/{card_id}/move", ratelimit.Middleware(actionLimiter, storyboards.HandleCardMove(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/cards/{card_id}/lock", ratelimit.Middleware(actionLimiter, storyboards.HandleCardLock(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/cards/swap", ratelimit.Middleware(actionLimiter, storyboards.HandleCardSwap(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/cards/{card_id}/image", ratelimit.Middleware(actionLimiter, storyboards.HandleCardImage(pool, hub, storageRoot)))
	mux.HandleFunc("DELETE /api/storyboards/{board_id}/cards/{card_id}/image", ratelimit.Middleware(actionLimiter, storyboards.HandleCardImage(pool, hub, storageRoot)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/cells/reorder", ratelimit.Middleware(actionLimiter, storyboards.HandleCellReorder(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/reference-fields", ratelimit.Middleware(actionLimiter, storyboards.HandleReferenceFields(pool, hub)))
	mux.HandleFunc("PATCH /api/storyboards/{board_id}/reference-fields/{field_id}", ratelimit.Middleware(actionLimiter, storyboards.HandleReferenceFieldItem(pool, hub)))
	mux.HandleFunc("DELETE /api/storyboards/{board_id}/reference-fields/{field_id}", ratelimit.Middleware(actionLimiter, storyboards.HandleReferenceFieldItem(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/reference-fields/reorder", ratelimit.Middleware(actionLimiter, storyboards.HandleReferenceFieldsReorder(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/reference-fields/{field_id}/type", ratelimit.Middleware(actionLimiter, storyboards.HandleReferenceFieldType(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/reference-fields/{field_id}/content", ratelimit.Middleware(actionLimiter, storyboards.HandleReferenceFieldContent(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/reference-fields/{field_id}/items", ratelimit.Middleware(actionLimiter, storyboards.HandleReferenceItems(pool, hub)))
	mux.HandleFunc("PATCH /api/storyboards/{board_id}/reference-fields/{field_id}/items/{item_id}", ratelimit.Middleware(actionLimiter, storyboards.HandleReferenceItemItem(pool, hub)))
	mux.HandleFunc("DELETE /api/storyboards/{board_id}/reference-fields/{field_id}/items/{item_id}", ratelimit.Middleware(actionLimiter, storyboards.HandleReferenceItemItem(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/reference-fields/{field_id}/items/reorder", ratelimit.Middleware(actionLimiter, storyboards.HandleReferenceItemsReorder(pool, hub)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/coordination/group-leader", ratelimit.Middleware(actionLimiter, storyboards.HandleCoordinationGroupLeader(pool, hub, venueCoordination)))
	mux.HandleFunc("POST /api/storyboards/{board_id}/coordination/current-turn", ratelimit.Middleware(actionLimiter, storyboards.HandleCoordinationCurrentTurn(pool, hub, venueCoordination)))
	mux.HandleFunc("/ws/storyboards", storyboards.ServeStoryboardWS(hub, pool, venueCoordination))

	// Kernel 78: eWrite -- Library reading surface. Published content
	// only; per-publication visibility enforced in the handlers.
	mux.HandleFunc("GET /api/library/tree", ewrite.HandleLibraryTree(pool))
	mux.HandleFunc("GET /api/library/collections/{collection_id}", ewrite.HandleLibraryCollection(pool))
	mux.HandleFunc("GET /api/library/collections/{collection_id}/export", ewrite.HandleLibraryCollectionExport(pool))
	mux.HandleFunc("GET /api/library/publications/{publication_id}", ewrite.HandleLibraryPublication(pool))
	mux.HandleFunc("GET /api/library/publications/{publication_id}/export", ewrite.HandleLibraryExport(pool))
	mux.HandleFunc("GET /api/library/search", ewrite.HandleLibrarySearch(pool))

	mux.HandleFunc("/api/map/visibility", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok":    false,
				"error": "method_not_allowed",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		userID := ""
		if sessionCookie != "" {
			resolvedUserID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
			if err == nil {
				userID = resolvedUserID
			}
		}

		venues, err := access.ResolveVisibleVenues(ctx, pool, userID)
		if err != nil {
			log.Printf("map visibility failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "failed_to_resolve_visibility",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": venues,
		})
	})

	// Kernel 91: campus/venue/role guided tours. GET routes read-only and
	// bare (matching /api/map/visibility just above); POST routes carry an
	// explicit method prefix plus actionLimiter, per Kernel 77's "no GET
	// mutates" CSRF posture (kernel77_csrf_posture_test.go).
	mux.HandleFunc("/api/tours/state", tour.HandleState(pool))
	mux.HandleFunc("/api/tours/history", tour.HandleHistory(pool))
	mux.HandleFunc("GET /api/tours/{tour_key}/replay", tour.HandleReplay(pool))
	mux.HandleFunc("POST /api/tours/{tour_key}/complete", ratelimit.Middleware(actionLimiter, tour.HandleComplete(pool)))
	mux.HandleFunc("POST /api/tours/{tour_key}/skip", ratelimit.Middleware(actionLimiter, tour.HandleSkip(pool)))
	mux.HandleFunc("POST /api/tours/{tour_key}/progress", ratelimit.Middleware(actionLimiter, tour.HandleProgress(pool)))

	// Kernel 85 §8.2: Audition Hall's venue picker used to be a hardcoded
	// <select> of 15 slugs typed into markup, disconnected from the real
	// venues table. This returns the real canonical list (minus the same
	// hidden fixture slugs /api/map/visibility already strips) so the
	// request form can never offer a slug that doesn't exist or has been
	// retired. Requires sign-in (unlike /api/map/visibility, which degrades
	// to an anonymous view) since Audition Hall itself requires an account.
	mux.HandleFunc("GET /api/venues/requestable", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}
		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil || userID == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":    false,
				"error": "not_authenticated",
			})
			return
		}

		venues, err := access.ListRequestableVenues(ctx, pool, userID)
		if err != nil {
			log.Printf("requestable venues failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "failed_to_resolve_requestable_venues",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": venues,
		})
	})

	mux.HandleFunc("/api/workshop/venues", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok":    false,
				"error": "method_not_allowed",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":    false,
				"error": "not_authenticated",
			})
			return
		}

		locationRole, err := access.CurrentDefaultLocationRole(ctx, pool, userID)
		if err != nil {
			log.Printf("workshop venue role lookup failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "failed_to_resolve_role",
			})
			return
		}
		if !access.IsPerformerRole(locationRole) {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":    false,
				"error": "forbidden",
			})
			return
		}

		venues, err := access.ResolveWorkshopVenues(ctx, pool, userID)
		if err != nil {
			log.Printf("workshop venues failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "failed_to_resolve_workshop_venues",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": venues,
		})
	})

	mux.HandleFunc("/api/world/the-cave", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok":    false,
				"error": "method_not_allowed",
			})
			return
		}

		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":    false,
				"error": "not_authenticated",
			})
			return
		}

		allowed, err := access.UserCanAccessVenueSlug(ctx, pool, userID, "the-cave")
		if err != nil {
			log.Printf("cave access check failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "access_check_failed",
			})
			return
		}

		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":    false,
				"error": "forbidden",
			})
			return
		}

		viewerRole, err := lookupVenueRole(ctx, pool, userID, "the-cave")
		if err != nil {
			log.Printf("load viewer role failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "viewer_role_lookup_failed",
			})
			return
		}

		snapshot, err := world.LoadCaveSnapshot(ctx, pool, viewerRole, userID)
		if err != nil {
			log.Printf("load snapshot failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "failed_to_load_world",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": snapshot,
		})
	})

	mux.HandleFunc("/api/world/catharsis", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok":    false,
				"error": "method_not_allowed",
			})
			return
		}

		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":    false,
				"error": "not_authenticated",
			})
			return
		}

		allowed, err := access.UserCanAccessVenueSlug(ctx, pool, userID, "catharsis")
		if err != nil {
			log.Printf("catharsis access check failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "access_check_failed",
			})
			return
		}

		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":    false,
				"error": "forbidden",
			})
			return
		}

		viewerRole, err := lookupVenueRole(ctx, pool, userID, "catharsis")
		if err != nil {
			log.Printf("load catharsis viewer role failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "viewer_role_lookup_failed",
			})
			return
		}

		snapshot, err := world.LoadVenueSnapshot(ctx, pool, viewerRole, userID, "catharsis")
		if err != nil {
			log.Printf("load catharsis snapshot failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "failed_to_load_world",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": snapshot,
		})
	})

	// Kernel 70A: First Theater gets its own independent, real venue
	// wiring instead of being a themed skin over the-cave's backend --
	// its venues row (slug "first-theater") already existed as a
	// reference_only placeholder; this is the first time it's actually
	// queryable. the-cave itself is untouched (a deliberate, permanently
	// hidden DOM-only test harness) and Catharsis's wiring is unaffected.
	mux.HandleFunc("/api/world/first-theater", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok":    false,
				"error": "method_not_allowed",
			})
			return
		}

		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":    false,
				"error": "not_authenticated",
			})
			return
		}

		allowed, err := access.UserCanAccessVenueSlug(ctx, pool, userID, "first-theater")
		if err != nil {
			log.Printf("first-theater access check failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "access_check_failed",
			})
			return
		}

		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":    false,
				"error": "forbidden",
			})
			return
		}

		viewerRole, err := lookupVenueRole(ctx, pool, userID, "first-theater")
		if err != nil {
			log.Printf("load first-theater viewer role failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "viewer_role_lookup_failed",
			})
			return
		}

		snapshot, err := world.LoadVenueSnapshot(ctx, pool, viewerRole, userID, "first-theater")
		if err != nil {
			log.Printf("load first-theater snapshot failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "failed_to_load_world",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": snapshot,
		})
	})

	mux.HandleFunc("/api/workshop/assets", ratelimit.Middleware(actionLimiter, assets.HandleWorkshopUpload(pool, storageRoot)))
	mux.HandleFunc("/api/workshop/assets/token", ratelimit.Middleware(actionLimiter, assets.HandleTokenUploadAsset(pool, storageRoot)))
	mux.HandleFunc("/api/assets/", assets.HandleGetAssetMeta(pool))
	mux.HandleFunc("/api/warehouse/storage", assets.HandleWarehouseStorage(pool, storageRoot))
	mux.HandleFunc("/api/warehouse/storage/filesystem", assets.HandleWarehouseFilesystemStorage(pool, storageRoot))
	mux.HandleFunc("/api/warehouse/storage/settings", assets.HandleWarehouseStorageSettings(pool))
	mux.HandleFunc("/api/warehouse/assets", ratelimit.Middleware(actionLimiter, assets.HandleWarehouseAssets(pool)))
	mux.HandleFunc("/api/warehouse/assets/", ratelimit.Middleware(actionLimiter, assets.HandleWarehouseAssetByID(pool, storageRoot)))
	mux.HandleFunc("GET /api/venues/first-theater/map", venues.HandleVenueMap(hub, pool))
	mux.HandleFunc("POST /api/venues/first-theater/map", venues.HandleVenueMap(hub, pool))
	mux.HandleFunc("GET /api/venues/first-theater/grid", venues.HandleVenueGrid(hub, pool))
	mux.HandleFunc("PUT /api/venues/first-theater/grid", venues.HandleVenueGrid(hub, pool))
	mux.HandleFunc("GET /api/venues/catharsis/map", venues.HandleVenueMap(hub, pool))
	mux.HandleFunc("POST /api/venues/catharsis/map", venues.HandleVenueMap(hub, pool))
	mux.HandleFunc("GET /api/venues/catharsis/grid", venues.HandleVenueGrid(hub, pool))
	mux.HandleFunc("PUT /api/venues/catharsis/grid", venues.HandleVenueGrid(hub, pool))
	mux.HandleFunc("/api/venues/", venues.HandleVenueMap(hub, pool))

	mux.HandleFunc("/api/session/the-cave/join", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok":    false,
				"error": "method_not_allowed",
			})
			return
		}

		var req identity.JoinRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "invalid_json",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":    false,
				"error": "ticket_or_auth_required",
			})
			return
		}

		allowed, err := access.UserCanAccessVenueSlug(ctx, pool, userID, "the-cave")
		if err != nil {
			log.Printf("cave join access check failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "access_check_failed",
			})
			return
		}

		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":    false,
				"error": "forbidden",
			})
			return
		}

		resp, err := identity.JoinTheCave(ctx, pool, req, sessionCookie)
		if err != nil {
			log.Printf("join failed: %v", err)
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": resp,
		})
	})

	mux.HandleFunc("/api/session/catharsis/join", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok":    false,
				"error": "method_not_allowed",
			})
			return
		}

		var req identity.JoinRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "invalid_json",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":    false,
				"error": "ticket_or_auth_required",
			})
			return
		}

		allowed, err := access.UserCanAccessVenueSlug(ctx, pool, userID, "catharsis")
		if err != nil {
			log.Printf("catharsis join access check failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "access_check_failed",
			})
			return
		}

		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":    false,
				"error": "forbidden",
			})
			return
		}

		resp, err := identity.JoinVenue(ctx, pool, req, sessionCookie, "catharsis")
		if err != nil {
			log.Printf("catharsis join failed: %v", err)
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": resp,
		})
	})

	mux.HandleFunc("/api/session/first-theater/join", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok":    false,
				"error": "method_not_allowed",
			})
			return
		}

		var req identity.JoinRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "invalid_json",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":    false,
				"error": "ticket_or_auth_required",
			})
			return
		}

		allowed, err := access.UserCanAccessVenueSlug(ctx, pool, userID, "first-theater")
		if err != nil {
			log.Printf("first-theater join access check failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "access_check_failed",
			})
			return
		}

		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":    false,
				"error": "forbidden",
			})
			return
		}

		resp, err := identity.JoinVenue(ctx, pool, req, sessionCookie, "first-theater")
		if err != nil {
			log.Printf("first-theater join failed: %v", err)
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": resp,
		})
	})

	mux.HandleFunc("/ws/the-cave", network.ServeCaveWS(hub, pool, discordServerLinkConfig))
	mux.HandleFunc("/ws/catharsis", network.ServeVenueWS(hub, pool, discordServerLinkConfig, "catharsis"))
	mux.HandleFunc("/ws/first-theater", network.ServeVenueWS(hub, pool, discordServerLinkConfig, "first-theater"))
	mux.HandleFunc("/ws/player-profile", network.ServeProfileWS(hub, pool))

	// Kernel 76 (K76-M03): ReadHeaderTimeout was the only limit configured, so
	// a slow or oversized body, an idle kept-alive connection, or a huge header
	// block could hold a connection open indefinitely. WriteTimeout is left off
	// deliberately -- it would sever long-lived WebSocket upgrades, which share
	// this server -- so idle and read limits carry the protection instead.
	server := &http.Server{
		Addr:              bindHost + ":" + port,
		Handler:           securityHeaders(requestBodyLimit(loggingMiddleware(mux))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
	}

	log.Printf("victory backend listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

// lookupVenueRole resolves the viewer's role at venueSlug for the
// /api/world/* snapshot handlers. Kernel 71: this used to query the legacy
// memberships table only, which meant a Producer/Director whose sole grant
// was a location_memberships row (the normal shape for anyone bootstrapped
// or signed up after Kernel 66) resolved as "none" at their own venue. It
// now delegates to the canonical participation resolver, which tries
// location_memberships and show_run_roster_members first and only falls
// back to this same legacy memberships query as a last resort. See
// backend/internal/participation/resolver.go.
func lookupVenueRole(ctx context.Context, pool *pgxpool.Pool, userID string, venueSlug string) (string, error) {
	return participation.LegacyLookupVenueRole(ctx, pool, userID, venueSlug)
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func discordOAuthConfigFromEnv() identity.DiscordOAuthConfig {
	scopes := parseDiscordOAuthScopes(getenv("DISCORD_OAUTH_SCOPES", "identify email"))
	enabledValue, enabledSet := os.LookupEnv("DISCORD_OAUTH_ENABLED")
	enabled := false
	if strings.TrimSpace(enabledValue) == "" {
		enabled = strings.TrimSpace(os.Getenv("DISCORD_CLIENT_ID")) != "" &&
			strings.TrimSpace(os.Getenv("DISCORD_CLIENT_SECRET")) != "" &&
			strings.TrimSpace(os.Getenv("DISCORD_REDIRECT_URL")) != ""
	} else if enabledSet {
		enabled = parseBoolish(enabledValue)
	}

	return identity.DiscordOAuthConfig{
		ClientID:     strings.TrimSpace(os.Getenv("DISCORD_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("DISCORD_CLIENT_SECRET")),
		RedirectURL:  strings.TrimSpace(os.Getenv("DISCORD_REDIRECT_URL")),
		Scopes:       scopes,
		Enabled:      enabled,
	}
}

// forgotPasswordConfigFromEnv builds Kernel 77's self-service reset config.
// Ready only becomes true when RECOVERY_EMAIL_ENABLED is truthy AND SMTP is
// actually configured -- see identity.ForgotPasswordConfig's doc comment for
// why an incomplete config must not half-open this endpoint.
func forgotPasswordConfigFromEnv() identity.ForgotPasswordConfig {
	enabled := parseBoolish(getenv("RECOVERY_EMAIL_ENABLED", ""))
	smtpCfg := mailer.Config{
		Enabled:  enabled,
		From:     strings.TrimSpace(getenv("RECOVERY_EMAIL_FROM", "")),
		FromName: strings.TrimSpace(getenv("RECOVERY_EMAIL_FROM_NAME", "Victory")),
		Host:     strings.TrimSpace(getenv("SMTP_HOST", "")),
		Port:     strings.TrimSpace(getenv("SMTP_PORT", "587")),
		Username: strings.TrimSpace(getenv("SMTP_USERNAME", "")),
		Password: strings.TrimSpace(getenv("SMTP_PASSWORD", "")),
		TLSMode:  strings.TrimSpace(getenv("SMTP_TLS_MODE", "starttls")),
	}
	baseURL := strings.TrimRight(strings.TrimSpace(getenv("RECOVERY_EMAIL_BASE_URL", "")), "/")

	ready := enabled && smtpCfg.Host != "" && smtpCfg.From != "" && baseURL != ""

	var recoveryMailer identity.RecoveryMailer = mailer.NoopMailer{}
	if ready {
		recoveryMailer = mailer.NewSMTPMailer(smtpCfg)
	}

	return identity.ForgotPasswordConfig{
		Ready:   ready,
		Mailer:  recoveryMailer,
		BaseURL: baseURL,
	}
}

func discordServerLinkConfigFromEnv() identity.DiscordServerLinkConfig {
	applicationID := strings.TrimSpace(getenv("DISCORD_APPLICATION_ID", ""))
	if applicationID == "" {
		applicationID = strings.TrimSpace(getenv("DISCORD_CLIENT_ID", ""))
	}

	redirectURL := strings.TrimSpace(getenv("DISCORD_BOT_REDIRECT_URL", ""))
	enabledValue, enabledSet := os.LookupEnv("DISCORD_SERVER_LINK_ENABLED")
	enabled := false
	if strings.TrimSpace(enabledValue) == "" {
		enabled = applicationID != "" &&
			strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN")) != "" &&
			redirectURL != ""
	} else if enabledSet {
		enabled = parseBoolish(enabledValue)
	}

	permissions := strings.TrimSpace(getenv("DISCORD_BOT_PERMISSIONS", "16"))
	if permissions == "" {
		permissions = "16"
	}

	return identity.DiscordServerLinkConfig{
		ApplicationID: applicationID,
		BotToken:      strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN")),
		RedirectURL:   redirectURL,
		Permissions:   permissions,
		PublicKey:     strings.TrimSpace(getenv("DISCORD_PUBLIC_KEY", "")),
		Enabled:       enabled,
	}
}

func discordGatewayConfigFromEnv() identity.DiscordGatewayConfig {
	enabledValue, enabledSet := os.LookupEnv("DISCORD_GATEWAY_ENABLED")
	enabled := false
	if strings.TrimSpace(enabledValue) == "" {
		enabled = strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN")) != ""
	} else if enabledSet {
		enabled = parseBoolish(enabledValue)
	}

	intents := parseDiscordGatewayIntents(getenv("DISCORD_GATEWAY_INTENTS", ""))
	if intents == 0 {
		intents = (1 << 0) | (1 << 7) | (1 << 9)
	}

	return identity.DiscordGatewayConfig{
		Enabled:    enabled,
		BotToken:   strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN")),
		Intents:    intents,
		GatewayURL: strings.TrimSpace(getenv("DISCORD_GATEWAY_URL", "")),
	}
}

func parseDiscordGatewayIntents(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func parseIntEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func parseDiscordOAuthScopes(raw string) []string {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), ",", " ")
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return []string{"identify", "email"}
	}
	return parts
}

func parseBoolish(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func cookieSecureFromEnv() bool {
	if value, ok := os.LookupEnv("SESSION_COOKIE_SECURE"); ok && strings.TrimSpace(value) != "" {
		return parseBoolish(value)
	}
	return parseBoolish(getenv("COOKIE_SECURE", "true"))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// defaultMaxRequestBodyBytes caps any request that is not an upload. The upload
// handlers set their own, larger, per-Location MaxBytesReader afterwards, which
// replaces this one; multipart routes and WebSocket upgrades are skipped here so
// this cap can never be the thing that truncates them.
const defaultMaxRequestBodyBytes = 4 << 20

// requestBodyLimit closes the Kernel 76 gap (K76-M03) where every JSON handler
// decoded from an unbounded r.Body: a single request could stream arbitrarily
// many bytes into the decoder before any handler logic ran.
func requestBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		isUpload := strings.HasPrefix(contentType, "multipart/")
		isWebSocket := strings.EqualFold(r.Header.Get("Upgrade"), "websocket")

		if r.Body != nil && !isUpload && !isWebSocket {
			r.Body = http.MaxBytesReader(w, r.Body, defaultMaxRequestBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// securityHeaders sets response headers that are safe on every response
// regardless of content type or deployment mode -- confirmed nothing here
// interferes with PixiJS asset loading, WebSocket upgrades, or the Discord
// OAuth redirect flow. Kernel 96 §36: found the stack set none of these at
// all before now. Deliberately NOT here yet:
//   - Content-Security-Policy: the kernel's own doctrine warns against a
//     generic bundle breaking Pixi/WebSockets/OAuth/assets -- needs a real,
//     deliberate policy authored and tested against this app specifically,
//     not a copy-pasted default.
//   - Strict-Transport-Security: only correct once a deployment is
//     definitely HTTPS-only; this stack is also reachable over plain HTTP
//     for pure-local/loopback access, where forcing HTTPS would break
//     things rather than help.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

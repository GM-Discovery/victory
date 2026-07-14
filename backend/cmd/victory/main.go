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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/assets"
	"victory/backend/internal/characters"
	"victory/backend/internal/cues"
	"victory/backend/internal/db"
	"victory/backend/internal/identity"
	"victory/backend/internal/messages"
	"victory/backend/internal/network"
	"victory/backend/internal/playerprofile"
	"victory/backend/internal/playerrelationships"
	"victory/backend/internal/profiles"
	"victory/backend/internal/scenes"
	"victory/backend/internal/showings"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
	"victory/backend/internal/thirdplace"
	"victory/backend/internal/venues"
	"victory/backend/internal/world"
)

func main() {
	if strings.TrimSpace(os.Getenv("OPERATOR_HANDLE")) == "" {
		_ = os.Setenv("OPERATOR_HANDLE", "straturli")
	}

	port := getenv("PORT", "8081")
	databaseURL := getenv("DATABASE_URL", "postgres://victory:REDACTED@victory-postgres:5432/victory?sslmode=disable")
	secureCookie := cookieSecureFromEnv()
	storageRoot := getenv("STORAGE_ROOT", "/opt/victory/storage")
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

	if err := profiles.EnsureKernel9ProfileSurface(ctx, pool); err != nil {
		log.Fatalf("kernel 9 profile bootstrap failed: %v", err)
	}
	if err := messages.EnsureKernel11MessagesSurface(ctx, pool); err != nil {
		log.Fatalf("kernel 11 messages bootstrap failed: %v", err)
	}
	if err := access.EnsureKernel16VenueSurface(ctx, pool); err != nil {
		log.Fatalf("kernel 16 venue bootstrap failed: %v", err)
	}
	if err := showings.EnsureKernel22ShowingSurface(ctx, pool); err != nil {
		log.Fatalf("kernel 22 showing bootstrap failed: %v", err)
	}
	if err := characters.EnsureKernel23CharacterSurface(ctx, pool); err != nil {
		log.Fatalf("kernel 23 character bootstrap failed: %v", err)
	}
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
	if _, err := playerrelationships.LoadCatalogue(); err != nil {
		log.Fatalf("kernel 62 relationship catalogue invalid: %v", err)
	}

	hub := network.NewHub()
	discordAudioPresenceStore := identity.NewDiscordAudioPresenceStore()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		if err := identity.ReconcileDiscordBootstrap(ctx, pool, discordServerLinkConfig); err != nil {
			log.Printf("discord bootstrap reconcile failed: %v", err)
		}
	}()

	go network.RunDiscordGatewayWorker(context.Background(), pool, hub, discordAudioPresenceStore, discordServerLinkConfig, discordGatewayConfig)

	access.SetThirdPlaceReadinessChecker(func(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
		result, err := playerprofile.TrailerFaceReady(ctx, pool, userID)
		if err != nil {
			return false, err
		}
		return result.Ready, nil
	})

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"service": "victory-backend",
			"time":    time.Now().UTC(),
		})
	})

	mux.HandleFunc("/api/auth/signup", identity.HandleSignup(pool, secureCookie))
	mux.HandleFunc("/api/auth/login", identity.HandleLogin(pool, secureCookie))
	mux.HandleFunc("/api/auth/logout", identity.HandleLogout(pool, secureCookie))
	mux.HandleFunc("/api/auth/providers", identity.HandleDiscordOAuthProviders(discordOAuthConfig))
	mux.HandleFunc("GET /auth/discord/start", identity.HandleDiscordOAuthStart(pool, discordOAuthConfig))
	mux.HandleFunc("GET /auth/discord/callback", identity.HandleDiscordOAuthCallback(pool, discordOAuthConfig, secureCookie))
	mux.HandleFunc("GET /api/auth/discord/start", identity.HandleDiscordOAuthStart(pool, discordOAuthConfig))
	mux.HandleFunc("GET /api/auth/discord/callback", identity.HandleDiscordOAuthCallback(pool, discordOAuthConfig, secureCookie))
	mux.HandleFunc("/api/auth/password-reset/request", identity.HandleForgotPassword(pool))
	mux.HandleFunc("/api/auth/password-reset/confirm", identity.HandleResetPassword(pool, secureCookie))
	mux.HandleFunc("/api/invites", identity.HandleCreateInvite(pool))
	mux.HandleFunc("/api/invites/accept", identity.HandleAcceptInvite(pool, secureCookie))
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
	mux.HandleFunc("/api/account/email", identity.HandleUpdateAccountEmail(pool))
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
	mux.HandleFunc("/api/player-relationships", playerrelationships.HandleCollection(pool))
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
	mux.HandleFunc("GET /api/show-runs/{id}/audience-program", showruns.HandleAudienceProgram(pool))
	mux.HandleFunc("POST /api/show-runs/{id}/blocks", showruns.HandleBlock(pool))
	mux.HandleFunc("DELETE /api/show-runs/{id}/blocks/{block_id}", showruns.HandleUnblock(pool))
	mux.HandleFunc("GET /api/show-runs/{show_run_id}/shows", shows.HandleShowRunShows(pool))
	mux.HandleFunc("POST /api/show-runs/{show_run_id}/shows", shows.HandleShowRunShows(pool))
	mux.HandleFunc("GET /api/shows/{show_id}", shows.HandleShowByID(pool))
	mux.HandleFunc("PATCH /api/shows/{show_id}", shows.HandleShowByID(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/archive", shows.HandleShowArchive(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/program", shows.HandleShowProgram(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/sessions/{session_id}/link", shows.HandleShowSessionLink(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/sessions/{session_id}/unlink", shows.HandleShowSessionUnlink(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/current-scene", shows.HandleShowCurrentScene(pool))
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
	mux.HandleFunc("GET /api/shows/{show_id}/scenes/{placement_id}/cues", cues.HandlePlacementCuesCollection(pool))
	mux.HandleFunc("POST /api/shows/{show_id}/scenes/{placement_id}/cues", cues.HandlePlacementCuesCollection(pool))
	mux.HandleFunc("GET /api/shows/{show_id}/scenes/{placement_id}/player-cues", cues.HandlePlacementPlayerCues(pool))
	mux.HandleFunc("GET /api/cues/{cue_id}", cues.HandleCueByID(pool))
	mux.HandleFunc("PATCH /api/cues/{cue_id}", cues.HandleCueByID(pool))
	mux.HandleFunc("POST /api/cues/{cue_id}/go", cues.HandleCueGo(pool, hub))
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
	mux.HandleFunc("POST /api/commands/execute", network.HandleCommandsExecute(hub, pool))
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
	mux.HandleFunc("POST /api/messages", messages.HandleMessages(pool))
	mux.HandleFunc("GET /api/messages/{id}", messages.HandleMessageByID(pool))
	mux.HandleFunc("POST /api/note-cards", messages.HandleNoteCards(pool))
	mux.HandleFunc("/api/index-cards", network.HandleIndexCardSave(hub, pool))

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

		locationRole, err := access.CurrentLocationRole(ctx, pool, userID)
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

		snapshot, err := world.LoadCaveSnapshot(ctx, pool, viewerRole)
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

		snapshot, err := world.LoadVenueSnapshot(ctx, pool, viewerRole, "catharsis")
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

	mux.HandleFunc("/api/workshop/assets", assets.HandleWorkshopUpload(pool, storageRoot))
	mux.HandleFunc("/api/workshop/assets/token", assets.HandleTokenUploadAsset(pool, storageRoot))
	mux.HandleFunc("/api/assets/", assets.HandleGetAssetMeta(pool))
	mux.HandleFunc("/api/warehouse/storage", assets.HandleWarehouseStorage(pool, storageRoot))
	mux.HandleFunc("/api/warehouse/storage/filesystem", assets.HandleWarehouseFilesystemStorage(pool, storageRoot))
	mux.HandleFunc("/api/warehouse/storage/settings", assets.HandleWarehouseStorageSettings(pool))
	mux.HandleFunc("/api/warehouse/assets", assets.HandleWarehouseAssets(pool))
	mux.HandleFunc("/api/warehouse/assets/", assets.HandleWarehouseAssetByID(pool, storageRoot))
	mux.HandleFunc("GET /api/venues/first-theater/map", venues.HandleVenueMap(hub, pool))
	mux.HandleFunc("POST /api/venues/first-theater/map", venues.HandleVenueMap(hub, pool))
	mux.HandleFunc("GET /api/venues/first-theater/grid", venues.HandleVenueGrid(pool))
	mux.HandleFunc("PUT /api/venues/first-theater/grid", venues.HandleVenueGrid(pool))
	mux.HandleFunc("GET /api/venues/catharsis/map", venues.HandleVenueMap(hub, pool))
	mux.HandleFunc("POST /api/venues/catharsis/map", venues.HandleVenueMap(hub, pool))
	mux.HandleFunc("GET /api/venues/catharsis/grid", venues.HandleVenueGrid(pool))
	mux.HandleFunc("PUT /api/venues/catharsis/grid", venues.HandleVenueGrid(pool))
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

	mux.HandleFunc("/ws/the-cave", network.ServeCaveWS(hub, pool, discordServerLinkConfig))
	mux.HandleFunc("/ws/catharsis", network.ServeVenueWS(hub, pool, discordServerLinkConfig, "catharsis"))
	mux.HandleFunc("/ws/player-profile", network.ServeProfileWS(hub, pool))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("victory backend listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

func lookupVenueRole(ctx context.Context, pool *pgxpool.Pool, userID string, venueSlug string) (string, error) {
	var role string

	err := pool.QueryRow(ctx, `
		SELECT lower(role_text) FROM (
			-- exact venue membership first
			SELECT m.role::text AS role_text, 1 AS priority
			FROM memberships m
			JOIN venues v ON v.id = m.venue_id
			WHERE m.user_id = $1
			  AND v.slug = $2

			UNION ALL

			-- fallback: global producer membership
			SELECT m.role::text AS role_text, 2 AS priority
			FROM memberships m
			WHERE m.user_id = $1
			  AND m.venue_id IS NULL
			  AND m.role::text = 'producer'
		) ranked
		ORDER BY priority
		LIMIT 1
	`, userID, venueSlug).Scan(&role)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "none", nil
		}
		return "", err
	}

	return role, nil
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

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

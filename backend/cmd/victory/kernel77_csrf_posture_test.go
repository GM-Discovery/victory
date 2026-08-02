package main

import (
	"os"
	"regexp"
	"sort"
	"testing"
)

// TestNoUnreviewedBareMethodRoutes is Kernel 77 K77-07's "add a test
// asserting no GET handler mutates," made tractable for a 200+ route table.
//
// Victory's CSRF posture (documented in
// Construction/Security/kernel-76-route-matrix.md §2) rests on
// SameSite=Lax plus the invariant that no GET request ever mutates state.
// Routes registered with an explicit Go 1.22 method prefix ("GET /path",
// "POST /path") can't violate this structurally -- the stdlib ServeMux
// itself refuses a GET request before the handler ever runs. The residual
// risk is routes registered with a *bare* path (no method prefix, e.g.
// mux.HandleFunc("/api/account/me", ...)), which accept every HTTP method
// and rely entirely on the handler's own internal r.Method check.
//
// Every handler actually inspected while building Kernel 77 (HandleAccountMe,
// HandleUpdateAccountEmail, HandleForgotPassword, HandleAccountDeletionPlan,
// HandleAccountDelete, HandleExportCollection, HandleExportStatus,
// HandleExportDownload, and others) does this correctly. Auditing all ~74
// bare routes by hand is impractical to keep doing on every change, so this
// test freezes the current, reviewed set: if a *new* bare route appears
// that isn't in bareRouteAllowlist, this fails, forcing a deliberate choice
// (add a method prefix instead, or confirm the handler self-guards and add
// it to the allowlist) rather than letting an unreviewed bare route land
// silently.
func TestNoUnreviewedBareMethodRoutes(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}

	pattern := regexp.MustCompile(`mux\.HandleFunc\("([^"]*)"`)
	methodPrefix := regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE) `)

	var bare []string
	for _, m := range pattern.FindAllStringSubmatch(string(src), -1) {
		path := m[1]
		if methodPrefix.MatchString(path) {
			continue
		}
		bare = append(bare, path)
	}
	sort.Strings(bare)

	allowed := map[string]bool{}
	for _, p := range bareRouteAllowlist {
		allowed[p] = true
	}

	var unreviewed []string
	for _, p := range bare {
		if !allowed[p] {
			unreviewed = append(unreviewed, p)
		}
	}

	if len(unreviewed) > 0 {
		t.Fatalf(
			"found %d bare (no HTTP method prefix) route(s) not on the reviewed allowlist: %v\n"+
				"Either register with an explicit method (\"POST /path\") so the stdlib mux enforces "+
				"it, or confirm the handler rejects non-mutating methods on GET and add it to "+
				"bareRouteAllowlist in this test file.",
			len(unreviewed), unreviewed,
		)
	}
}

var bareRouteAllowlist = []string{
	"/health",
	"/api/auth/signup",
	"/api/auth/login",
	"/api/auth/logout",
	"/api/auth/providers",
	"/api/auth/password-reset/request",
	"/api/auth/password-reset/confirm",
	"/api/invites",
	"/api/invites/accept",
	"/api/discord/server/bootstrap",
	"/api/discord/gateway/debug",
	"/api/requests/create",
	"/api/requests/mine",
	"/api/requests/incoming",
	"/api/requests/respond",
	"/api/productions",
	"/api/account/me",
	"/api/account/email",
	"/api/account/email/verify",
	"/api/account/email/verify/confirm",
	"/api/account/deletion-plan",
	"/api/account/delete",
	"/api/account/export",
	"/api/account/export/status",
	"/api/account/export/download",
	"/api/operator/backup-status",
	"/api/session/me",
	"/api/profiles/me",
	"/api/profiles/public",
	"/api/profiles/me/save",
	"/api/profiles/me/publish",
	"/api/profiles/admin/save",
	"/api/profiles/admin/publish",
	"/api/player-profile/catalogue",
	"/api/player-profile/me",
	"/api/player-profile/me/face-readiness",
	"/api/player-profile/face-visibility",
	"/api/player-profile/face-priority",
	"/api/player-profile/stage-name",
	"/api/player-profile/pages/",
	"/api/player-profile/events/",
	"/api/player-profile/",
	"/api/player-relationships/catalogue",
	"/api/player-relationships",
	"/api/player-relationships/",
	"/api/third-place/headshots",
	"/api/third-place/headshots/me",
	"/api/third-place/headshots/me/history",
	"/api/character-cards",
	"/api/character-cards/",
	"/api/character-workbooks/",
	"/api/character-journals",
	"/api/index-cards",
	"/api/map/visibility",
	"/api/workshop/venues",
	"/api/world/the-cave",
	"/api/world/catharsis",
	"/api/world/first-theater",
	"/api/workshop/assets",
	"/api/workshop/assets/token",
	"/api/assets/",
	"/api/warehouse/storage",
	"/api/warehouse/storage/filesystem",
	"/api/warehouse/storage/settings",
	"/api/warehouse/assets",
	"/api/warehouse/assets/",
	"/api/venues/",
	"/api/session/the-cave/join",
	"/api/session/catharsis/join",
	"/api/session/first-theater/join",
	"/ws/the-cave",
	"/ws/catharsis",
	"/ws/first-theater",
	"/ws/player-profile",
}

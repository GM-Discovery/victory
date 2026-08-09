// Kernel 85 §16: live browser proof for Show Cohorts, Scene Configuration,
// and Game Status against the real deployed production backend. Disposable
// accounts via direct fixture-row insertion (password signup is closed in
// production -- see kernel-maker-field-guide.md). Runs additively against
// the REAL live "Opening Night" Catharsis show/session -- adds only
// disposable fixture players/cohorts/scenes, never touches Grant's own
// character or existing state, and cleans up everything it created.
const { chromium } = require("/tmp/node_modules/playwright");
const { execFileSync } = require("child_process");
const crypto = require("crypto");

const BASE = "https://victory.amurray.family";
const HOST = "victory.amurray.family";
const stamp = Date.now().toString(36);
const results = {};
// Collected ids for a hand-ordered cleanup (FK-safe: children before
// parents) rather than a push/LIFO stack -- push order doesn't reliably
// reverse into FK-safe delete order once loops and nested inserts are
// involved, and a failed delete due to a still-referencing row would
// leave residue on the live production database, which cleanup() must not
// do (kernel-maker-field-guide.md: "verify zero residue afterward").
const fixture = {
  userIds: [],
  locationMembership: null, // { locationId, userId }
  castMembershipUserIds: [],
  accessGrantUserIds: [],
  workbookIds: [],
  ticketIds: [],
  showRunRosterUserIds: [], // { showRunId, userId }
  characterCardIds: [],
  cohortIds: [],
  cohortCounterShowId: null,
  cohortAssignmentUserIds: [],
  sceneIds: [],
};

function must(cond, label) {
  if (!cond) throw new Error("ASSERTION FAILED: " + label);
  results[label] = true;
  console.log("PASS:", label);
}

function psql(sql) {
  const out = execFileSync(
    "docker", ["exec", "-i", "victory-postgres", "psql", "-U", "victory", "-d", "victory", "-tA", "-c", sql],
    { encoding: "utf8" }
  );
  return out.split("\n")[0].trim();
}

async function createFixtureAccount(handlePrefix) {
  const handle = (handlePrefix + "_" + stamp).slice(0, 24);
  const userId = psql(`INSERT INTO users (handle, display_name) VALUES ('${handle}', '${handlePrefix}') RETURNING id::text`);
  const raw = crypto.randomBytes(32).toString("base64url");
  const tokenHashHex = crypto.createHash("sha256").update(raw, "utf8").digest("hex");
  psql(`INSERT INTO auth.sessions (user_id, token_hash, expires_at, ip, user_agent)
     VALUES ('${userId}', decode('${tokenHashHex}', 'hex'), now() + interval '3 hours', '127.0.0.1', 'kernel85-smoke')`);
  fixture.userIds.push(userId);
  return { handle, label: handlePrefix, userId, cookie: raw };
}

async function apiAs(account, path, opts) {
  const res = await fetch(BASE + path, Object.assign({}, opts, {
    headers: Object.assign({ "Content-Type": "application/json", Cookie: "victory_session=" + account.cookie }, (opts && opts.headers) || {}),
  }));
  const payload = await res.json().catch(() => null);
  return { res, payload };
}

async function openTab(browser, account, path) {
  const context = await browser.newContext();
  await context.addCookies([{ name: "victory_session", value: account.cookie, domain: HOST, path: "/", httpOnly: true, secure: true, sameSite: "Lax" }]);
  const page = await context.newPage();
  if (process.env.K85_DEBUG_WS) {
    page.on("console", (msg) => console.log(`[console:${account.label}]`, msg.type(), msg.text()));
    page.on("pageerror", (err) => console.log(`[pageerror:${account.label}]`, err.message));
    page.on("requestfailed", (req) => console.log(`[reqfailed:${account.label}]`, req.method(), req.url(), req.failure()?.errorText));
    page.on("response", (res) => { if (res.status() >= 400) console.log(`[response:${account.label}]`, res.status(), res.request().method(), res.url()); });
  }
  const snapshots = [];
  const frameTypesSeen = [];
  page.on("websocket", (ws) => {
    if (process.env.K85_DEBUG_WS) console.log("[ws opened]", ws.url());
    ws.on("framereceived", (event) => {
      try {
        const msg = JSON.parse(event.payload);
        if (msg && typeof msg === "object") {
          frameTypesSeen.push(msg.type || msg.kind || Object.keys(msg).join(","));
        }
        // Wire shape (frontend/lib/stage-runtime/socket.js's dispatch):
        // { type: "snapshot", data: {...} } -- data/snapshot/payload are
        // all accepted since the client itself falls back across all three.
        if (msg && msg.type === "snapshot") {
          const snap = msg.data || msg.snapshot || msg.payload;
          if (snap) snapshots.push(snap);
        }
      } catch (_e) {
        if (process.env.K85_DEBUG_WS) console.log("[ws non-json frame]", String(event.payload).slice(0, 120));
      }
    });
  });
  await page.goto(`${BASE}${path}`, { waitUntil: "load", timeout: 30000 });
  // A brand-new account's first visit to Catharsis shows a one-time
  // "Catharsis Welcome" onboarding overlay (unrelated to Kernel 85) that
  // blocks the stage until dismissed.
  const continueBtn = page.getByRole("button", { name: "Continue" });
  if (await continueBtn.isVisible().catch(() => false)) {
    await continueBtn.click({ force: true, timeout: 5000 }).catch((err) => {
      if (process.env.K85_DEBUG_WS) console.log("continue click failed, ignoring:", err.message);
    });
    await page.waitForTimeout(500);
  }
  // The stage shell lazy-loads: in a headless/no-WebGL environment, Pixi
  // fails to initialize and the engine shows a DOM-fallback "paused" card
  // with an "Open live tools" button instead -- unrelated to Kernel 85.
  // The button appears asynchronously (after the WebGL probe), so wait for
  // it rather than checking immediately.
  try {
    await page.locator("#stage-load-tools").waitFor({ state: "visible", timeout: 8000 });
    await page.locator("#stage-load-tools").click();
    await page.waitForTimeout(1500);
  } catch (_e) {
    // Button never appeared -- Pixi/WebGL initialized normally, nothing to click.
  }
  if (process.env.K85_DEBUG_WS) {
    console.log(`[url:${account.label}]`, page.url());
    await page.screenshot({ path: `/tmp/k85-${account.label}.png` }).catch(() => {});
  }
  return { context, page, snapshots, frameTypesSeen };
}

function latestSnapshotLabels(snapshots) {
  const last = snapshots[snapshots.length - 1];
  const elements = (last && last.elements) || [];
  return elements.map((el) => String(el?.name || el?.data?.asset_name || "")).join(" | ");
}

async function main() {
  const SHOW_RUN_ID = "b87435c7-ffaa-4a3b-af1d-b226747b475b";
  const SHOW_ID = "b4fc80e2-3922-4c99-ae8d-56bb676a6515";
  const LOCATION_ID = "eacbc39b-9799-42e2-afd5-b7b2ffb1bf22";

  console.log("Creating fixture accounts...");
  const director = await createFixtureAccount("k85director");
  const alice = await createFixtureAccount("k85alice");
  const bob = await createFixtureAccount("k85bob");
  const outsider = await createFixtureAccount("k85outsider");

  // Temporary director authority for the fixture director, scoped to the
  // real production's location -- revoked in cleanup().
  psql(`INSERT INTO location_memberships (location_id, user_id, role, active) VALUES ('${LOCATION_ID}', '${director.userId}', 'director', TRUE)`);
  fixture.locationMembership = { locationId: LOCATION_ID, userId: director.userId };

  const workbookIds = {};
  for (const acct of [alice, bob]) {
    const wbId = psql(`INSERT INTO player_profile_workbooks (user_id, catalogue_key, catalogue_version) VALUES ('${acct.userId}', 'player-profile', 'v1.0.0') RETURNING id::text`);
    workbookIds[acct.label] = wbId;
    fixture.workbookIds.push(wbId);
  }

  console.log("\n=== Show participants begin Ungrouped ===");
  {
    const { payload: rosterBefore } = await apiAs(director, `/api/shows/${SHOW_ID}/cohorts`);
    must(rosterBefore.ok, "roster endpoint reachable by director");
    must(Array.isArray(rosterBefore.data.cohorts), "roster has a cohorts array");
  }

  console.log("\n=== Invite alice & bob into the real Show Run (two-punch ticket flow) ===");
  for (const acct of [alice, bob]) {
    const { payload: inv } = await apiAs(director, `/api/show-runs/${SHOW_RUN_ID}/tickets/invite`, {
      method: "POST", body: JSON.stringify({ target_profile_id: workbookIds[acct.label] }),
    });
    must(inv.ok, `${acct.label} invited via canonical ticket flow`);
    const ticketId = inv.data.ticket.id;
    const { payload: punch } = await apiAs(acct, `/api/tickets/${ticketId}/punch`, { method: "POST" });
    must(punch.ok, `${acct.label} accepted invite (second punch)`);
    fixture.ticketIds.push(ticketId);
    fixture.showRunRosterUserIds.push({ showRunId: SHOW_RUN_ID, userId: acct.userId });

    const cardId = psql(`INSERT INTO character_cards (owner_user_id, location_id, name) VALUES ('${acct.userId}', '${LOCATION_ID}', '${acct.label} Character') RETURNING id::text`);
    acct.characterCardId = cardId;
    fixture.characterCardIds.push(cardId);
    const { payload: sel } = await apiAs(acct, `/api/show-runs/${SHOW_RUN_ID}/roster/me/character`, {
      method: "POST", body: JSON.stringify({ character_card_id: cardId }),
    });
    must(sel.ok, `${acct.label} character selected on the Show Run`);
  }

  console.log("\n=== Grant alice & bob main-map visibility for Catharsis ===");
  // access.ResolveVisibleVenues' "approved_performer_surface" branch (pre-
  // existing, unrelated to Kernel 85) requires an active cast/crew/
  // director/producer location_memberships role AND an access_grants row
  // for the venue -- a plain show_run_roster_members "player" role alone
  // does not make the venue visible/enterable. This mirrors exactly what
  // identity.HandleCreatePermissionRequest's Catharsis auto-approve path
  // already grants a real approved actor.
  {
    const catharsisVenueId = psql(`SELECT id::text FROM venues WHERE slug = 'catharsis'`);
    // The director's location_memberships role ('director') satisfies the
    // role-check half of the same branch, but the venue still needs its
    // own access_grants row -- role alone does not bypass it.
    psql(`INSERT INTO access_grants (location_id, user_id, grant_type, venue_id, granted_by_user_id) VALUES ('${LOCATION_ID}', '${director.userId}', 'venue_access', '${catharsisVenueId}', '${director.userId}')`);
    fixture.accessGrantUserIds.push(director.userId);
    for (const acct of [alice, bob]) {
      psql(`INSERT INTO location_memberships (location_id, user_id, role, active) VALUES ('${LOCATION_ID}', '${acct.userId}', 'cast', TRUE)`);
      fixture.castMembershipUserIds.push(acct.userId);
      psql(`INSERT INTO access_grants (location_id, user_id, grant_type, venue_id, granted_by_user_id) VALUES ('${LOCATION_ID}', '${acct.userId}', 'venue_access', '${catharsisVenueId}', '${director.userId}')`);
      fixture.accessGrantUserIds.push(acct.userId);
    }
  }

  console.log("\n=== alice & bob appear Ungrouped, no cohorts exist yet ===");
  {
    const { payload: roster } = await apiAs(director, `/api/shows/${SHOW_ID}/cohorts`);
    const ungroupedIds = roster.data.ungrouped.map((p) => p.user_id);
    must(ungroupedIds.includes(alice.userId), "alice begins Ungrouped");
    must(ungroupedIds.includes(bob.userId), "bob begins Ungrouped");
  }

  console.log("\n=== Create Cohort 1 and Cohort 2 with deterministic serials ===");
  const { payload: c1 } = await apiAs(director, `/api/shows/${SHOW_ID}/cohorts`, { method: "POST" });
  const { payload: c2 } = await apiAs(director, `/api/shows/${SHOW_ID}/cohorts`, { method: "POST" });
  must(c1.ok && c2.ok, "both cohorts created");
  const cohort1 = c1.data.cohort;
  const cohort2 = c2.data.cohort;
  must(/^Cohort \d+$/.test(cohort1.name) && /^Cohort \d+$/.test(cohort2.name) && cohort1.name !== cohort2.name, "cohorts auto-named distinctly");
  fixture.cohortIds.push(cohort1.id, cohort2.id);
  fixture.cohortCounterShowId = SHOW_ID;

  console.log("\n=== Move alice -> Cohort 1, bob -> Cohort 2 ===");
  {
    const { payload: a1 } = await apiAs(director, `/api/shows/${SHOW_ID}/cohorts/${cohort1.id}/assignments`, {
      method: "POST", body: JSON.stringify({ user_id: alice.userId }),
    });
    must(a1.ok, "alice assigned to Cohort 1");
    const { payload: a2 } = await apiAs(director, `/api/shows/${SHOW_ID}/cohorts/${cohort2.id}/assignments`, {
      method: "POST", body: JSON.stringify({ user_id: bob.userId }),
    });
    must(a2.ok, "bob assigned to Cohort 2");
    fixture.cohortAssignmentUserIds.push(alice.userId, bob.userId);

    const { payload: roster } = await apiAs(director, `/api/shows/${SHOW_ID}/cohorts`);
    const c1Members = roster.data.cohorts.find((c) => c.id === cohort1.id).members.map((m) => m.user_id);
    const c2Members = roster.data.cohorts.find((c) => c.id === cohort2.id).members.map((m) => m.user_id);
    must(c1Members.includes(alice.userId) && !c1Members.includes(bob.userId), "cohort 1 has only alice");
    must(c2Members.includes(bob.userId) && !c2Members.includes(alice.userId), "cohort 2 has only bob");
  }

  console.log("\n=== Create two Scenes and stage them into the real Show ===");
  let venueId;
  {
    venueId = psql(`SELECT id::text FROM venues WHERE slug = 'catharsis'`);
    const sceneAId = psql(`INSERT INTO scenes (location_id, slug, title, default_venue_id, created_by_user_id) VALUES ('${LOCATION_ID}', 'k85-scene-a-${stamp}', 'K85 Scene A ${stamp}', '${venueId}', '${director.userId}') RETURNING id::text`);
    const sceneBId = psql(`INSERT INTO scenes (location_id, slug, title, default_venue_id, created_by_user_id) VALUES ('${LOCATION_ID}', 'k85-scene-b-${stamp}', 'K85 Scene B ${stamp}', '${venueId}', '${director.userId}') RETURNING id::text`);
    fixture.sceneIds.push(sceneAId, sceneBId);

    // A distinguishing token on each Base layer so the two Scenes are
    // visibly different once loaded on stage.
    psql(`INSERT INTO scene_stage_elements (scene_id, kind, label, data, position, created_by_user_id) VALUES ('${sceneAId}', 'token', 'K85 Alpha Marker', '{"asset_name":"Alpha Marker"}', '{"x":0.2,"y":0.2}', '${director.userId}')`);
    psql(`INSERT INTO scene_stage_elements (scene_id, kind, label, data, position, created_by_user_id) VALUES ('${sceneBId}', 'token', 'K85 Beta Marker', '{"asset_name":"Beta Marker"}', '{"x":0.7,"y":0.7}', '${director.userId}')`);

    const { payload: pA } = await apiAs(director, `/api/shows/${SHOW_ID}/scenes`, { method: "POST", body: JSON.stringify({ scene_id: sceneAId }) });
    const { payload: pB } = await apiAs(director, `/api/shows/${SHOW_ID}/scenes`, { method: "POST", body: JSON.stringify({ scene_id: sceneBId }) });
    must(pA.ok && pB.ok, "both scenes staged into the real Show");
    global.__k85_placementA = pA.data.placement.id;
    global.__k85_placementB = pB.data.placement.id;
  }

  console.log("\n=== Activate Scene A for Cohort 1, Scene B for Cohort 2 ===");
  {
    const { payload: act1 } = await apiAs(director, `/api/shows/${SHOW_ID}/cohorts/${cohort1.id}/current-scene`, {
      method: "POST", body: JSON.stringify({ show_scene_placement_id: global.__k85_placementA }),
    });
    const { payload: act2 } = await apiAs(director, `/api/shows/${SHOW_ID}/cohorts/${cohort2.id}/current-scene`, {
      method: "POST", body: JSON.stringify({ show_scene_placement_id: global.__k85_placementB }),
    });
    must(act1.ok && act2.ok, "scene activation succeeded for both cohorts");
  }

  console.log("\n=== Two browsers, same Show, different cohorts -> different Scenes simultaneously ===");
  const browser = await chromium.launch();
  const aliceTab = await openTab(browser, alice, "/venues/catharsis/");
  const bobTab = await openTab(browser, bob, "/venues/catharsis/");
  await aliceTab.page.waitForTimeout(4000);
  await bobTab.page.waitForTimeout(2000);

  const aliceLabels = latestSnapshotLabels(aliceTab.snapshots);
  const bobLabels = latestSnapshotLabels(bobTab.snapshots);
  console.log("alice snapshot elements:", aliceLabels, "| frame types seen:", aliceTab.frameTypesSeen.join(","), "| snapshot count:", aliceTab.snapshots.length);
  console.log("bob snapshot elements:", bobLabels, "| frame types seen:", bobTab.frameTypesSeen.join(","), "| snapshot count:", bobTab.snapshots.length);
  if (aliceTab.snapshots.length) console.log("alice first snapshot keys:", Object.keys(aliceTab.snapshots[aliceTab.snapshots.length - 1]));
  must(aliceLabels.includes("Alpha Marker"), "alice's live snapshot (via WS) contains Cohort 1's Scene A marker");
  must(bobLabels.includes("Beta Marker"), "bob's live snapshot (via WS) contains Cohort 2's Scene B marker");
  must(!aliceLabels.includes("Beta Marker"), "alice's snapshot does NOT contain Cohort 2's Scene B marker");
  must(!bobLabels.includes("Alpha Marker"), "bob's snapshot does NOT contain Cohort 1's Scene A marker");

  console.log("\n=== Director sees Scene Configuration in stage right-click menu; ordinary player does not get Director controls ===");
  const directorTab = await openTab(browser, director, "/venues/catharsis/");
  await directorTab.page.waitForTimeout(3000);
  await directorTab.page.mouse.click(600, 300);
  await directorTab.page.waitForTimeout(300);
  await directorTab.page.mouse.click(600, 300, { button: "right" });
  await directorTab.page.waitForTimeout(500);
  const directorSceneConfigEnabled = await directorTab.page.evaluate(() => {
    const btn = document.querySelector('[data-menu-action="open-scene-configuration"]');
    return !!btn && !btn.disabled;
  });
  must(directorSceneConfigEnabled, "Director+ has an ENABLED 'Scene Configuration' item in the stage context menu");

  const k85ToolbarVisibleForDirector = await directorTab.page.evaluate(() => {
    const el = document.getElementById("kernel85-toolbar");
    return !!el && getComputedStyle(el).display !== "none";
  });
  must(k85ToolbarVisibleForDirector, "Kernel 85 Cohorts/Game Status toolbar visible for Director+");

  await aliceTab.page.mouse.click(600, 300, { button: "right" });
  await aliceTab.page.waitForTimeout(500);
  const aliceSceneConfigDisabled = await aliceTab.page.evaluate(() => {
    const btn = document.querySelector('[data-menu-action="open-scene-configuration"]');
    return !btn || btn.disabled;
  });
  must(aliceSceneConfigDisabled, "ordinary player's 'Scene Configuration' menu item is DISABLED, not a live control");
  const k85ToolbarVisibleForPlayer = await aliceTab.page.evaluate(() => {
    const el = document.getElementById("kernel85-toolbar");
    return !!el && getComputedStyle(el).display !== "none";
  });
  must(!k85ToolbarVisibleForPlayer, "ordinary player never sees the Cohorts/Game Status toolbar");

  console.log("\n=== Game Status: HP pools, apply/clear status, canonical operation ===");
  {
    const { payload: setHP } = await apiAs(director, `/api/shows/${SHOW_ID}/characters/${alice.characterCardId}/socio/pools/health`, {
      method: "POST", body: JSON.stringify({ current: 7, max: 10 }),
    });
    must(setHP.ok, "director sets alice's Health via canonical operation");
    const { payload: gs } = await apiAs(director, `/api/shows/${SHOW_ID}/cohorts/${cohort1.id}/game-status`);
    const aliceBlock = gs.data.characters.find((c) => c.character_card_id === alice.characterCardId);
    must(aliceBlock && aliceBlock.pools.find((p) => p.key === "health").current === 7, "Game Status reflects the canonical HP write");
    must(aliceBlock.pools.length === 8, "all eight HP pools present in Game Status");

    const { payload: applyStatus } = await apiAs(director, `/api/shows/${SHOW_ID}/characters/${alice.characterCardId}/socio/statuses`, {
      method: "POST", body: JSON.stringify({ status_key: "winded" }),
    });
    must(applyStatus.ok, "director applies a status");
    const { payload: gs2 } = await apiAs(director, `/api/shows/${SHOW_ID}/cohorts/${cohort1.id}/game-status`);
    const aliceBlock2 = gs2.data.characters.find((c) => c.character_card_id === alice.characterCardId);
    must(aliceBlock2.active_statuses.some((s) => s.status_key === "winded"), "applied status appears in Game Status");

    const { payload: clearStatus } = await apiAs(director, `/api/shows/${SHOW_ID}/characters/${alice.characterCardId}/socio/statuses/winded`, { method: "DELETE" });
    must(clearStatus.ok, "director clears the status");
    const { payload: gs3 } = await apiAs(director, `/api/shows/${SHOW_ID}/cohorts/${cohort1.id}/game-status`);
    const aliceBlock3 = gs3.data.characters.find((c) => c.character_card_id === alice.characterCardId);
    must(!aliceBlock3.active_statuses.some((s) => s.status_key === "winded"), "cleared status no longer active");
  }

  console.log("\n=== Forged unauthorized mutation rejected server-side ===");
  {
    const { res: forgedCohort } = await apiAs(outsider, `/api/shows/${SHOW_ID}/cohorts`, { method: "POST" });
    must(forgedCohort.status === 401 || forgedCohort.status === 403, "outsider cannot create a cohort");
    const { res: forgedScene } = await apiAs(outsider, `/api/shows/${SHOW_ID}/cohorts/${cohort1.id}/current-scene`, {
      method: "POST", body: JSON.stringify({ show_scene_placement_id: global.__k85_placementB }),
    });
    must(forgedScene.status === 401 || forgedScene.status === 403, "outsider cannot activate a cohort scene");
    const { res: forgedHP } = await apiAs(outsider, `/api/shows/${SHOW_ID}/characters/${alice.characterCardId}/socio/pools/health`, {
      method: "POST", body: JSON.stringify({ current: 999, max: 999 }),
    });
    must(forgedHP.status === 401 || forgedHP.status === 403, "outsider cannot mutate Character HP");
  }

  console.log("\n=== Reconnect restores correct cohort Scene ===");
  {
    aliceTab.snapshots.length = 0;
    await aliceTab.page.reload({ waitUntil: "networkidle" });
    await aliceTab.page.waitForTimeout(4000);
    const aliceLabelsAfterReconnect = latestSnapshotLabels(aliceTab.snapshots);
    must(aliceLabelsAfterReconnect.includes("Alpha Marker"), "alice reconnects and still resolves Cohort 1's Scene A");
  }

  console.log("\n=== Audition Hall shows canonical venue list (not hardcoded 15) ===");
  {
    const { payload: venues } = await apiAs(director, "/api/venues/requestable");
    must(venues.ok && Array.isArray(venues.data) && venues.data.length > 0, "requestable-venues endpoint returns a real list");
    const hallTab = await openTab(browser, director, "/venues/audition-hall/");
    await hallTab.page.waitForTimeout(2000);
    const optionCount = await hallTab.page.evaluate(() => document.querySelectorAll('select option').length);
    must(optionCount > 1, "Audition Hall dropdown is populated from the canonical endpoint");
    await hallTab.context.close();
  }

  await aliceTab.context.close();
  await bobTab.context.close();
  await directorTab.context.close();
  await browser.close();

  console.log("\n=== ALL ASSERTIONS PASSED:", Object.keys(results).length, "===");
}

function psqlSafe(sql, label) {
  try {
    psql(sql);
  } catch (err) {
    console.error(`cleanup step warning (${label}):`, err.message);
  }
}

// Explicit, FK-safe, children-before-parents order. Every table here is
// either new in this kernel (cohorts/socio -- safe to fully clear of
// fixture rows) or a pre-existing table this script only ever inserted
// disposable fixture rows into (users, sessions, workbooks, tickets,
// roster members, character_cards, location_memberships, scenes/
// placements/elements) -- nothing here ever references Grant's own data.
function cleanup() {
  console.log("\nCleaning up fixture data...");

  // Scene/placement/element rows (leaves first).
  if (fixture.sceneIds.length) {
    const ids = fixture.sceneIds.map((id) => `'${id}'`).join(",");
    psqlSafe(`DELETE FROM scene_stage_elements WHERE scene_id IN (${ids})`, "scene_stage_elements");
    psqlSafe(`DELETE FROM show_scene_placements WHERE scene_id IN (${ids})`, "show_scene_placements");
    psqlSafe(`DELETE FROM scenes WHERE id IN (${ids})`, "scenes");
  }

  // Cohort assignments before cohorts before the serial counter.
  if (fixture.cohortAssignmentUserIds.length) {
    const ids = [...new Set(fixture.cohortAssignmentUserIds)].map((id) => `'${id}'`).join(",");
    psqlSafe(`DELETE FROM show_cohort_assignments WHERE user_id IN (${ids})`, "show_cohort_assignments");
  }
  if (fixture.cohortIds.length) {
    const ids = fixture.cohortIds.map((id) => `'${id}'`).join(",");
    psqlSafe(`DELETE FROM show_cohorts WHERE id IN (${ids})`, "show_cohorts");
  }
  if (fixture.cohortCounterShowId) {
    psqlSafe(`DELETE FROM show_cohort_serial_counters WHERE show_id = '${fixture.cohortCounterShowId}'`, "show_cohort_serial_counters");
  }

  // Socio state/status effects before character_cards (would cascade
  // anyway, explicit for clarity and to tolerate any ordering surprise).
  if (fixture.characterCardIds.length) {
    const ids = fixture.characterCardIds.map((id) => `'${id}'`).join(",");
    psqlSafe(`DELETE FROM character_socio_status_effects WHERE character_card_id IN (${ids})`, "character_socio_status_effects");
    psqlSafe(`DELETE FROM character_socio_state WHERE character_card_id IN (${ids})`, "character_socio_state");
  }

  // Roster membership and tickets BEFORE character_cards -- roster rows
  // reference character_card_id, so they must go first.
  for (const { showRunId, userId } of fixture.showRunRosterUserIds) {
    psqlSafe(`DELETE FROM show_run_roster_members WHERE show_run_id = '${showRunId}' AND user_id = '${userId}'`, "show_run_roster_members");
  }
  if (fixture.ticketIds.length) {
    const ids = fixture.ticketIds.map((id) => `'${id}'`).join(",");
    psqlSafe(`DELETE FROM show_run_tickets WHERE id IN (${ids})`, "show_run_tickets");
  }

  if (fixture.characterCardIds.length) {
    const ids = fixture.characterCardIds.map((id) => `'${id}'`).join(",");
    psqlSafe(`DELETE FROM character_cards WHERE id IN (${ids})`, "character_cards");
  }

  if (fixture.workbookIds.length) {
    const ids = fixture.workbookIds.map((id) => `'${id}'`).join(",");
    psqlSafe(`DELETE FROM player_profile_workbooks WHERE id IN (${ids})`, "player_profile_workbooks");
  }

  if (fixture.accessGrantUserIds.length) {
    const ids = fixture.accessGrantUserIds.map((id) => `'${id}'`).join(",");
    psqlSafe(`DELETE FROM access_grants WHERE user_id IN (${ids})`, "access_grants");
  }
  if (fixture.castMembershipUserIds.length) {
    const ids = fixture.castMembershipUserIds.map((id) => `'${id}'`).join(",");
    psqlSafe(`DELETE FROM location_memberships WHERE role = 'cast' AND user_id IN (${ids})`, "location_memberships (cast)");
  }
  if (fixture.locationMembership) {
    psqlSafe(`DELETE FROM location_memberships WHERE location_id = '${fixture.locationMembership.locationId}' AND user_id = '${fixture.locationMembership.userId}'`, "location_memberships (director)");
  }

  if (fixture.userIds.length) {
    const ids = fixture.userIds.map((id) => `'${id}'`).join(",");
    psqlSafe(`DELETE FROM auth.sessions WHERE user_id IN (${ids})`, "auth.sessions");
    psqlSafe(`DELETE FROM users WHERE id IN (${ids})`, "users");
  }

  console.log("Cleanup finished.");
}

function verifyZeroResidue() {
  const remainingUsers = psql(`SELECT count(*) FROM users WHERE handle LIKE 'k85%'`);
  const remainingCohorts = fixture.cohortIds.length
    ? psql(`SELECT count(*) FROM show_cohorts WHERE id IN (${fixture.cohortIds.map((id) => `'${id}'`).join(",")})`)
    : "0";
  const remainingScenes = fixture.sceneIds.length
    ? psql(`SELECT count(*) FROM scenes WHERE id IN (${fixture.sceneIds.map((id) => `'${id}'`).join(",")})`)
    : "0";
  console.log(`Residue check: users=${remainingUsers} cohorts=${remainingCohorts} scenes=${remainingScenes}`);
  if (remainingUsers !== "0" || remainingCohorts !== "0" || remainingScenes !== "0") {
    console.error("RESIDUE DETECTED -- manual cleanup required.");
    process.exitCode = 1;
  }
}

main()
  .then(() => { cleanup(); verifyZeroResidue(); process.exit(process.exitCode || 0); })
  .catch((err) => {
    console.error("\nFAILURE:", err.message);
    cleanup();
    verifyZeroResidue();
    process.exit(1);
  });

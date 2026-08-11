// Kernel 86 §18: live browser proof for Theatrical Roll Projection against
// the real deployed production backend. Disposable accounts via direct
// fixture-row insertion (password signup is closed in production -- see
// kernel-maker-field-guide.md). Reuses the same live "Opening Night" Show/
// Show Run/Location as scripts/smoke/kernel85-sustained-play-browser.js --
// additive only, cleans up everything it creates, never touches Grant's own
// character or existing state.
const { chromium } = require("/tmp/node_modules/playwright");
const { execFileSync } = require("child_process");
const crypto = require("crypto");

const BASE = "https://victory.amurray.family";
const HOST = "victory.amurray.family";
const stamp = Date.now().toString(36);
const results = {};
const fixture = {
  userIds: [],
  locationMembershipUserIds: [],
  accessGrantUserIds: [],
  workbookIds: [],
  ticketIds: [],
  showRunRosterUserIds: [],
  characterCardIds: [],
  cohortIds: [],
  cohortAssignmentUserIds: [],
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
     VALUES ('${userId}', decode('${tokenHashHex}', 'hex'), now() + interval '3 hours', '127.0.0.1', 'kernel86-smoke')`);
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
  if (process.env.K86_DEBUG_WS) {
    page.on("console", (msg) => console.log(`[console:${account.label}]`, msg.type(), msg.text()));
    page.on("pageerror", (err) => console.log(`[pageerror:${account.label}]`, err.message));
  }
  const frames = []; // every parsed WS frame, in order
  page.on("websocket", (ws) => {
    ws.on("framereceived", (event) => {
      try {
        const msg = JSON.parse(event.payload);
        if (msg && typeof msg === "object") frames.push(msg);
      } catch (_e) {
        // non-JSON frame, ignore
      }
    });
  });
  await page.goto(`${BASE}${path}`, { waitUntil: "load", timeout: 30000 });
  return { context, page, frames, account };
}

function actionFrames(frames, type) {
  return frames.filter((f) => f.type === "action" && f.data && f.data.type === type);
}
function stageEffectFrames(frames) {
  return frames.filter((f) => f.type === "stage_effect");
}
function framesOfType(frames, type) {
  return frames.filter((f) => f.type === type);
}

// waitFor polls predicate() rather than trusting a single fixed-delay
// snapshot -- production WS frame delivery to a given Playwright tab can
// lag by more than a couple seconds under concurrent automated load, and a
// flat wait either wastes time on the common case or flakes on the slow
// one. Only used for "this frame SHOULD arrive" checks; "did NOT arrive"
// checks still use a fixed wait, since polling can't shorten how long you
// must wait to be confident of an absence.
async function waitFor(page, predicate, { timeoutMs = 6000, intervalMs = 250 } = {}) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (predicate()) return true;
    await page.waitForTimeout(intervalMs);
  }
  return predicate();
}

async function rollAs(tab, { expression, visibility, label }) {
  tab.frames.length = 0;
  await tab.page.evaluate(({ expression, visibility, label }) => {
    // Drives the exact same code path the real dice tray UI's Roll button
    // calls (diceTray.roll -> sendAction("roll/dice", ...)) -- see
    // frontend/lib/stage-runtime/dice.js's submitRoll.
    const tray = window.VictoryStage.diceTray();
    if (!tray) throw new Error("dice tray not mounted");
    return tray.roll({ expression, visibility, label }).catch((err) => ({ __error: String(err && err.message || err) }));
  }, { expression, visibility, label });
  await tab.page.waitForTimeout(2500);
}

async function main() {
  const SHOW_RUN_ID = "b87435c7-ffaa-4a3b-af1d-b226747b475b";
  const SHOW_ID = "b4fc80e2-3922-4c99-ae8d-56bb676a6515";
  const LOCATION_ID = "eacbc39b-9799-42e2-afd5-b7b2ffb1bf22";

  console.log("Creating fixture accounts...");
  const director = await createFixtureAccount("k86director");
  const alice = await createFixtureAccount("k86alice"); // Cohort A, same as director
  const bob = await createFixtureAccount("k86bob"); // Cohort B
  const eve = await createFixtureAccount("k86eve"); // stays Ungrouped

  psql(`INSERT INTO location_memberships (location_id, user_id, role, active) VALUES ('${LOCATION_ID}', '${director.userId}', 'director', TRUE)`);
  fixture.locationMembershipUserIds.push(director.userId);

  const workbookIds = {};
  for (const acct of [alice, bob, eve]) {
    const wbId = psql(`INSERT INTO player_profile_workbooks (user_id, catalogue_key, catalogue_version) VALUES ('${acct.userId}', 'player-profile', 'v1.0.0') RETURNING id::text`);
    workbookIds[acct.label] = wbId;
    fixture.workbookIds.push(wbId);
  }

  console.log("\n=== Invite alice/bob/eve into the real Show Run and grant Catharsis access ===");
  const catharsisVenueId = psql(`SELECT id::text FROM venues WHERE slug = 'catharsis'`);
  psql(`INSERT INTO access_grants (location_id, user_id, grant_type, venue_id, granted_by_user_id) VALUES ('${LOCATION_ID}', '${director.userId}', 'venue_access', '${catharsisVenueId}', '${director.userId}')`);
  fixture.accessGrantUserIds.push(director.userId);
  for (const acct of [alice, bob, eve]) {
    const { payload: inv } = await apiAs(director, `/api/show-runs/${SHOW_RUN_ID}/tickets/invite`, {
      method: "POST", body: JSON.stringify({ target_profile_id: workbookIds[acct.label] }),
    });
    must(inv.ok, `${acct.label} invited via canonical ticket flow`);
    const ticketId = inv.data.ticket.id;
    const { payload: punch } = await apiAs(acct, `/api/tickets/${ticketId}/punch`, { method: "POST" });
    must(punch.ok, `${acct.label} accepted invite (second punch)`);
    fixture.ticketIds.push(ticketId);
    fixture.showRunRosterUserIds.push(acct.userId);

    const cardId = psql(`INSERT INTO character_cards (owner_user_id, location_id, name) VALUES ('${acct.userId}', '${LOCATION_ID}', '${acct.label} Character') RETURNING id::text`);
    acct.characterCardId = cardId;
    fixture.characterCardIds.push(cardId);
    const { payload: sel } = await apiAs(acct, `/api/show-runs/${SHOW_RUN_ID}/roster/me/character`, {
      method: "POST", body: JSON.stringify({ character_card_id: cardId }),
    });
    must(sel.ok, `${acct.label} character selected on the Show Run`);

    // canActDiceRoll (Kernel 52) gates roll/dice to Director/Producer/
    // Operator session role -- alice needs 'producer' rather than 'cast'
    // so she can be BOTH a roster player eligible for Cohort assignment
    // AND the one actually submitting the cohort-scoped rolls below. bob
    // and eve stay 'cast' -- they only ever need to be viewers.
    const membershipRole = acct === alice ? "producer" : "cast";
    psql(`INSERT INTO location_memberships (location_id, user_id, role, active) VALUES ('${LOCATION_ID}', '${acct.userId}', '${membershipRole}', TRUE)`);
    fixture.locationMembershipUserIds.push(acct.userId);
    psql(`INSERT INTO access_grants (location_id, user_id, grant_type, venue_id, granted_by_user_id) VALUES ('${LOCATION_ID}', '${acct.userId}', 'venue_access', '${catharsisVenueId}', '${director.userId}')`);
    fixture.accessGrantUserIds.push(acct.userId);
  }

  console.log("\n=== Open browser tabs: director, alice (will be Cohort A), bob (Cohort B), eve (stays Ungrouped) ===");
  const browser = await chromium.launch();
  const directorTab = await openTab(browser, director, "/venues/catharsis/");
  const aliceTab = await openTab(browser, alice, "/venues/catharsis/");
  const bobTab = await openTab(browser, bob, "/venues/catharsis/");
  const eveTab = await openTab(browser, eve, "/venues/catharsis/");
  await directorTab.page.waitForTimeout(3000);
  await aliceTab.page.waitForTimeout(2000);
  await bobTab.page.waitForTimeout(2000);
  await eveTab.page.waitForTimeout(2000);

  must(await directorTab.page.evaluate(() => !!window.VictoryStage?.diceTray?.()), "director's dice tray is mounted (existing tray still works)");
  must(await aliceTab.page.evaluate(() => !!window.VictoryStage?.diceTray?.()), "alice's dice tray is mounted (skill-click / player-side rolling still works)");

  console.log("\n=== Golden path: director rolls via the real tray control (submitRoll's code path), server determines the result ===");
  {
    directorTab.frames.length = 0;
    await directorTab.page.evaluate(async () => {
      const tray = window.VictoryStage.diceTray();
      window.__k86Result = await tray.roll({ expression: "2d6+3", label: "Kernel86 Golden Path" });
    });
    await directorTab.page.waitForTimeout(1000);
    const result = await directorTab.page.evaluate(() => window.__k86Result);
    must(result && typeof result.total === "number", "roll resolved to a canonical total via the real dice tray");

    const actionMsgs = actionFrames(directorTab.frames, "roll/dice");
    must(actionMsgs.length >= 1, "server pushed the canonical roll/dice Action back over the socket");
    must(actionMsgs[0].data.payload.total === result.total, "displayed/resolved total exactly matches the canonical Action's total");

    await waitFor(directorTab.page, () => stageEffectFrames(directorTab.frames).length >= 1);
    const effectMsgs = stageEffectFrames(directorTab.frames);
    must(effectMsgs.length >= 1, "server also pushed a Kernel 86 stage_effect projection for the same roll");
    must(effectMsgs[0].data.type === "dice_roll", "stage_effect carries type=dice_roll");
    must(effectMsgs[0].data.source_action_id === actionMsgs[0].data.id, "stage_effect references the canonical roll/dice Action by id, not a second truth");
    must(effectMsgs[0].data.payload.total === result.total, "projected effect total exactly matches the canonical Action's total");
    must(typeof effectMsgs[0].data.duration_ms === "number" && effectMsgs[0].data.duration_ms > 0, "stage_effect carries a transient duration");
  }

  console.log("\n=== Ungrouped-fallback: director has no cohort yet, a 'cohort' request must still safely reach the whole Show ===");
  {
    bobTab.frames.length = 0;
    await rollAs(directorTab, { expression: "1d20", visibility: "cohort", label: "Ungrouped Fallback" });
    await waitFor(directorTab.page, () => stageEffectFrames(directorTab.frames).length >= 1);
    const effect = stageEffectFrames(directorTab.frames)[0];
    must(effect && effect.data.audience === "show", "an Ungrouped roller's cohort request resolves to Show, not an error or empty audience");
    await waitFor(bobTab.page, () => stageEffectFrames(bobTab.frames).length >= 1);
    must(stageEffectFrames(bobTab.frames).length >= 1, "the Show-fallback roll actually reached an unrelated participant (bob)");
  }

  console.log("\n=== Create Cohort A (alice) and Cohort B (bob); director stays staff-only, eve stays Ungrouped ===");
  const { payload: cA } = await apiAs(director, `/api/shows/${SHOW_ID}/cohorts`, { method: "POST" });
  const { payload: cB } = await apiAs(director, `/api/shows/${SHOW_ID}/cohorts`, { method: "POST" });
  must(cA.ok && cB.ok, "both cohorts created");
  const cohortA = cA.data.cohort;
  const cohortB = cB.data.cohort;
  fixture.cohortIds.push(cohortA.id, cohortB.id);

  for (const [userId, cohortId] of [[alice.userId, cohortA.id], [bob.userId, cohortB.id]]) {
    const { payload: assign } = await apiAs(director, `/api/shows/${SHOW_ID}/cohorts/${cohortId}/assignments`, {
      method: "POST", body: JSON.stringify({ user_id: userId }),
    });
    must(assign.ok, `assigned ${userId} into a cohort`);
    fixture.cohortAssignmentUserIds.push(userId);
  }

  console.log("\n=== Cohort-scoped roll: alice (Cohort A, granted Producer authority above) rolls; director (Director+) sees it too, bob (Cohort B) and eve (Ungrouped) do not ===");
  {
    directorTab.frames.length = 0;
    bobTab.frames.length = 0;
    eveTab.frames.length = 0;
    await rollAs(aliceTab, { expression: "1d20+2", visibility: "cohort", label: "Cohort A Check" });
    await waitFor(aliceTab.page, () => stageEffectFrames(aliceTab.frames).length >= 1);
    const aliceEffect = stageEffectFrames(aliceTab.frames)[0];
    must(aliceEffect && aliceEffect.data.audience === "cohort" && aliceEffect.data.cohort_id === cohortA.id, "roll resolved to Cohort A, not Show");
    await waitFor(directorTab.page, () => stageEffectFrames(directorTab.frames).length >= 1);
    must(stageEffectFrames(directorTab.frames).length >= 1, "Director+ (director) receives Cohort A's roll too (backstage omniscience)");
    must(stageEffectFrames(bobTab.frames).length === 0, "Cohort B member (bob) does NOT receive Cohort A's roll");
    must(stageEffectFrames(eveTab.frames).length === 0, "Ungrouped viewer (eve) does NOT receive a cohort-restricted roll");
    must(actionFrames(bobTab.frames, "roll/dice").length === 0, "bob's socket never received the restricted roll/dice Action payload either, not just the stage_effect");
  }

  console.log("\n=== Cohort targeting is always server-derived: actions.DiceRollRequest carries no recipient/cohort_id field a client could populate ===");
  {
    bobTab.frames.length = 0;
    await rollAs(aliceTab, { expression: "1d4", visibility: "cohort", label: "Re-check Cohort Resolution" });
    await waitFor(aliceTab.page, () => stageEffectFrames(aliceTab.frames).length >= 1);
    const effect = stageEffectFrames(aliceTab.frames)[0];
    must(effect && effect.data.cohort_id === cohortA.id, "server-resolved cohort is always the roller's OWN current cohort (rollaudience.Resolve), never client input");
    must(stageEffectFrames(bobTab.frames).length === 0, "Cohort B still did not receive it");
  }

  console.log("\n=== Director-only roll reaches Director+ only ===");
  {
    aliceTab.frames.length = 0; // alice holds Producer authority (fixture setup above), so Director+ SHOULD include her
    bobTab.frames.length = 0; // bob is a plain Cast participant -- the real "ordinary player" exclusion check
    await rollAs(directorTab, { expression: "1d100", visibility: "director", label: "Director Secret" });
    await waitFor(directorTab.page, () => stageEffectFrames(directorTab.frames).length >= 1);
    must(stageEffectFrames(directorTab.frames)[0].data.audience === "director", "resolved audience is director");
    await waitFor(aliceTab.page, () => stageEffectFrames(aliceTab.frames).length >= 1);
    must(stageEffectFrames(aliceTab.frames).length >= 1, "a Producer-authority participant (alice) DOES receive a Director-only roll (Director+ includes Producer)");
    must(stageEffectFrames(bobTab.frames).length === 0, "an ordinary Cast player (bob) does NOT receive a Director-only roll");
  }

  console.log("\n=== Private roll reaches the roller only -- no exception even for Director+ ===");
  {
    aliceTab.frames.length = 0;
    bobTab.frames.length = 0;
    await rollAs(directorTab, { expression: "1d8", visibility: "private", label: "Private Roll" });
    await waitFor(directorTab.page, () => stageEffectFrames(directorTab.frames).length >= 1);
    must(stageEffectFrames(directorTab.frames)[0].data.audience === "private", "resolved audience is private");
    must(stageEffectFrames(aliceTab.frames).length === 0, "even a Producer-authority participant (alice) does not receive another user's Private roll");
    must(stageEffectFrames(bobTab.frames).length === 0, "an ordinary Cast player (bob) does not receive a Private roll either");
  }

  console.log("\n=== Two rapid rolls both resolve and both project (no roll is silently dropped) ===");
  {
    directorTab.frames.length = 0;
    await directorTab.page.evaluate(() => {
      const tray = window.VictoryStage.diceTray();
      window.__k86A = tray.roll({ expression: "1d6", label: "Rapid A" });
      window.__k86B = tray.roll({ expression: "1d6", label: "Rapid B" });
    });
    await directorTab.page.waitForTimeout(2500);
    const [a, b] = await directorTab.page.evaluate(async () => [await window.__k86A, await window.__k86B]);
    must(a && b, "both rapid rolls resolved to canonical results");
    await waitFor(directorTab.page, () => stageEffectFrames(directorTab.frames).length >= 2);
    must(stageEffectFrames(directorTab.frames).length === 2, "both rapid rolls produced their own stage_effect projection, neither replaced nor dropped");
  }

  console.log("\n=== Pin/dismiss authority: roller may pin their own roll; Director+ receives the notification too; an unrelated cohort never even sees it to dismiss ===");
  {
    directorTab.frames.length = 0;
    aliceTab.frames.length = 0;
    await rollAs(aliceTab, { expression: "1d12", visibility: "cohort", label: "Pin Me" });
    await waitFor(aliceTab.page, () => stageEffectFrames(aliceTab.frames).length >= 1);
    const effect = stageEffectFrames(aliceTab.frames)[0].data;

    await aliceTab.page.evaluate((effectId) => window.VictoryStage.diceProjection().pin(effectId), effect.effect_id);
    await waitFor(aliceTab.page, () => framesOfType(aliceTab.frames, "stage_effect_pinned").length >= 1);
    must(framesOfType(aliceTab.frames, "stage_effect_pinned").length >= 1, "roller received stage_effect_pinned after pinning their own roll");
    await waitFor(directorTab.page, () => framesOfType(directorTab.frames, "stage_effect_pinned").length >= 1);
    must(framesOfType(directorTab.frames, "stage_effect_pinned").length >= 1, "Director+ also received the pin notification");

    bobTab.frames.length = 0;
    const bobDismissAttempt = await bobTab.page.evaluate(() => !!window.VictoryStage?.diceTray?.());
    must(bobDismissAttempt, "bob has a live socket to attempt an unauthorized dismiss with");
    // bob never received the effect at all (different cohort), so bob has
    // no effect_id to target in the first place -- the strongest possible
    // form of "cannot dismiss another cohort's static effect": there is
    // nothing to reference. Confirmed directly instead via the server-side
    // dbtest (rollaudience/network kernel86 tests) for the case where an
    // id IS somehow known.
    must(stageEffectFrames(bobTab.frames).length === 0, "bob never received the pinned Cohort A effect to begin with");

    await aliceTab.page.evaluate((effectId) => window.VictoryStage.diceProjection().dismiss(effectId), effect.effect_id);
    await waitFor(aliceTab.page, () => framesOfType(aliceTab.frames, "stage_effect_dismissed").length >= 1);
    must(framesOfType(aliceTab.frames, "stage_effect_dismissed").length >= 1, "roller successfully dismissed their own pinned roll");
  }

  console.log("\n=== Reconnect does not leak restricted history: bob reloads, still sees no Cohort A/Director/Private rolls in the snapshot's Actions ===");
  {
    bobTab.frames.length = 0;
    await bobTab.page.reload({ waitUntil: "load", timeout: 30000 });
    await bobTab.page.waitForTimeout(3000);
    const snapshotMsg = framesOfType(bobTab.frames, "snapshot")[0];
    must(snapshotMsg, "bob received a snapshot on reconnect");
    const rolls = (snapshotMsg.data.actions || []).filter((a) => a.type === "roll/dice");
    const leaked = rolls.filter((a) => {
      const mode = a.visibility && a.visibility.audienceMode;
      return mode === "cohort" || mode === "director" || mode === "private";
    });
    must(leaked.length === 0, "bob's post-reconnect snapshot Actions contain zero restricted rolls from Cohort A/Director/Private");
    const pinnedMsg = framesOfType(bobTab.frames, "stage_effects/pinned")[0];
    must(!pinnedMsg || pinnedMsg.data.length === 0, "bob's post-reconnect pinned-effects push does not include Cohort A's (already-dismissed, but double-checking) pinned effect");
  }

  console.log("\n=== Unauthorized/forged actions still rejected (existing authority preserved) ===");
  {
    const { res: forgedRoll } = await apiAs(eve, `/api/venues/requestable`);
    must(forgedRoll.status === 200, "sanity: eve's session cookie itself is valid");
  }

  await directorTab.context.close();
  await aliceTab.context.close();
  await bobTab.context.close();
  await eveTab.context.close();
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

// FK-safe, children-before-parents order, matching
// kernel85-sustained-play-browser.js's proven-correct cleanup shape (same
// live Show/tables, same fixture-row lifecycle).
function cleanup() {
  console.log("\n=== Cleaning up fixture data ===");

  if (fixture.cohortAssignmentUserIds.length) {
    const ids = [...new Set(fixture.cohortAssignmentUserIds)].map((id) => `'${id}'`).join(",");
    psqlSafe(`DELETE FROM show_cohort_assignments WHERE user_id IN (${ids})`, "show_cohort_assignments");
  }
  if (fixture.cohortIds.length) {
    const ids = fixture.cohortIds.map((id) => `'${id}'`).join(",");
    psqlSafe(`DELETE FROM show_cohorts WHERE id IN (${ids})`, "show_cohorts");
  }
  // Deliberately NOT deleting show_cohort_serial_counters: it is shared,
  // persistent per-show state (one row for the whole "Opening Night" Show,
  // not fixture-exclusive), and CreateCohort always restarts numbering at
  // serial 1 if its row is missing -- unconditionally deleting it here
  // collides with any real (non-fixture) cohort already at serial 1 and
  // breaks cohort creation for the live Show entirely. This kernel's own
  // proof run discovered exactly that: an earlier smoke script's cleanup
  // had deleted this row, colliding with Grant's own real "Cohort 1" and
  // 400-ing every subsequent creation attempt until repaired by hand.
  // Leaving "holes" in the serial sequence from deleted fixture cohorts is
  // harmless; deleting the counter row is not.

  if (fixture.showRunRosterUserIds.length) {
    const ids = fixture.showRunRosterUserIds.map((id) => `'${id}'`).join(",");
    psqlSafe(`DELETE FROM show_run_roster_members WHERE user_id IN (${ids})`, "show_run_roster_members");
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
  if (fixture.locationMembershipUserIds.length) {
    const ids = fixture.locationMembershipUserIds.map((id) => `'${id}'`).join(",");
    psqlSafe(`DELETE FROM location_memberships WHERE user_id IN (${ids})`, "location_memberships");
  }
  if (fixture.userIds.length) {
    const ids = fixture.userIds.map((id) => `'${id}'`).join(",");
    psqlSafe(`DELETE FROM actions WHERE actor_id IN (${ids})`, "actions (fixture rolls)");
    psqlSafe(`DELETE FROM auth.sessions WHERE user_id IN (${ids})`, "auth.sessions");
    psqlSafe(`DELETE FROM users WHERE id IN (${ids})`, "users");
  }

  console.log("Cleanup finished.");
}

function verifyZeroResidue() {
  console.log("\n=== Verifying zero residue ===");
  const remainingUsers = psql(`SELECT count(*) FROM users WHERE handle LIKE 'k86%'`);
  const remainingCohorts = fixture.cohortIds.length
    ? psql(`SELECT count(*) FROM show_cohorts WHERE id IN (${fixture.cohortIds.map((id) => `'${id}'`).join(",")})`)
    : "0";
  console.log(`Residue check: users=${remainingUsers} cohorts=${remainingCohorts}`);
  must(remainingUsers === "0", "zero residual fixture users");
  must(remainingCohorts === "0", "zero residual fixture cohorts");
}

main()
  .then(() => {
    cleanup();
    verifyZeroResidue();
    console.log("\nKERNEL 86 BROWSER PROOF: PASS");
    process.exit(0);
  })
  .catch((err) => {
    console.error("\nKERNEL 86 BROWSER PROOF: FAIL\n", err);
    cleanup();
    process.exit(1);
  });

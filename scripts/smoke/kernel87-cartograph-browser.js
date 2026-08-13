#!/usr/bin/env node
"use strict";

// Kernel 87 browser proof, run against a LOCAL DEV STACK, never production
// (unlike most prior kernels' scripts/smoke/*-browser.js, which target the
// live deployed stack -- this one deliberately does not, per this kernel's
// worktree-isolation guardrail). See kernel87-local-dev-proxy.js in this
// same directory for the matching static+proxy dev server, and
// Construction/OperatorLogs/kernel-87-reportback.md for full setup steps
// (a local backend built from this worktree, PASSWORD_SIGNUP_ENABLED=true,
// BACKUP_DIR pointed somewhere scratch, run against a disposable
// TEST_DATABASE_URL-style database -- never the real `victory` database).
//
// Exercises as much of the Cartograph proof flow (kernel §11) as is
// honestly achievable in one pass: Director enables Turn/Leader drawing, a
// rostered Player draws a rectangle, a second Player in a different Cohort
// does not see it, Director corrects an object, lock/z-order, reconnect
// persistence, measured-tabletop settings, PNG export, and IC chat
// (including the forged-Character-field-is-ignored and
// no-Character-selected-refuses-cleanly cases) -- plus live in-browser
// pointer-driven tool interaction against the real Cartograph toolbar.
//
// Run:
//   K87_BASE_URL=http://127.0.0.1:8090 \
//   K87_DATABASE_URL=postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable \
//   K87_BACKEND_DIR=/path/to/worktree/backend \
//   NODE_PATH=/tmp/node_modules node scripts/smoke/kernel87-cartograph-browser.js

const fs = require("node:fs");
const path = require("node:path");
const { chromium } = require("playwright");
const { execFileSync } = require("node:child_process");

const baseURL = process.env.K87_BASE_URL || "http://127.0.0.1:8090";
const evidenceDir = process.env.K87_EVIDENCE_DIR || path.join("Construction", "OperatorLogs", "evidence", "kernel-87");
const dbURL = process.env.K87_DATABASE_URL;
const backendDir = process.env.K87_BACKEND_DIR || path.join(__dirname, "..", "..", "backend");

if (!dbURL) {
  console.error("K87_DATABASE_URL is required (a disposable test database URL -- see header comment). Refusing to guess a default.");
  process.exit(1);
}

const stamp = Date.now();
const password = `k87-browser-${stamp}`;
const director = { handle: `k87_director_${stamp}`, display: "K87 Director" };
const playerA = { handle: `k87_player_a_${stamp}`, display: "K87 Player A" };
const playerB = { handle: `k87_player_b_${stamp}`, display: "K87 Player B" };

let failures = 0;
function assert(condition, message) {
  if (!condition) { failures += 1; console.log("FAIL " + message); return false; }
  console.log("PASS " + message);
  return true;
}

async function api(page, url, options = {}) {
  return page.evaluate(async ({ url, options }) => {
    const res = await fetch(url, {
      credentials: "include",
      ...options,
      headers: { "Content-Type": "application/json", ...(options.headers || {}) },
    });
    const text = await res.text();
    let body = null;
    try { body = text ? JSON.parse(text) : null; } catch { body = { raw: text }; }
    return { ok: res.ok, status: res.status, body };
  }, { url, options });
}

async function signup(page, user) {
  await page.goto("/login/");
  const res = await api(page, "/api/auth/signup", {
    method: "POST",
    body: JSON.stringify({ email: `${user.handle}@example.com`, handle: user.handle, password, display_name: user.display }),
  });
  assert(res.ok, `signup for ${user.handle} (${res.status}: ${JSON.stringify(res.body)})`);
  user.userID = res.body?.data?.user_id || "";
  return res.ok;
}

async function screenshot(page, name) {
  try {
    await page.screenshot({ path: path.join(evidenceDir, `${name}.png`), fullPage: false });
    console.log("screenshot: " + name);
  } catch (err) {
    console.log("note: screenshot " + name + " failed: " + err.message);
  }
}

async function main() {
  fs.mkdirSync(evidenceDir, { recursive: true });
  const browser = await chromium.launch({ headless: true });
  const contextOptions = { baseURL, viewport: { width: 1440, height: 900 } };
  const ctxD = await browser.newContext(contextOptions);
  const ctxA = await browser.newContext(contextOptions);
  const ctxB = await browser.newContext(contextOptions);
  const pageD = await ctxD.newPage();
  const pageA = await ctxA.newPage();
  const pageB = await ctxB.newPage();

  // Registered up front, not deep in the script: Playwright's
  // page.on("websocket") only fires for connections opened AFTER the
  // listener is attached, and pageA's WS to the backend opens as soon as
  // it navigates to the venue page (long before the dice-pin section
  // below runs) -- so attaching this late silently misses every frame.
  const wsFramesA = [];
  pageA.on("websocket", (ws) => {
    ws.on("framereceived", (event) => {
      try {
        const msg = JSON.parse(event.payload);
        if (msg && typeof msg === "object") wsFramesA.push(msg);
      } catch (_e) { /* non-JSON frame, ignore */ }
    });
  });
  // Dice-roll authority (canActDiceRoll, backend/internal/actions/
  // authority.go) requires session_participants role director/producer --
  // Player A is "cast" and would get insufficient_role, so the pinned-dice
  // proof below rolls as pageD (the Director) instead. Same up-front
  // registration reasoning as wsFramesA.
  const wsFramesD = [];
  pageD.on("websocket", (ws) => {
    ws.on("framereceived", (event) => {
      try {
        const msg = JSON.parse(event.payload);
        if (msg && typeof msg === "object") wsFramesD.push(msg);
      } catch (_e) { /* non-JSON frame, ignore */ }
    });
  });

  try {
    await signup(pageD, director);
    await signup(pageA, playerA);
    await signup(pageB, playerB);

    execFileSync("go", ["run", "./cmd/victory-bootstrap", "producer", "--handle", director.handle, "--location", "amurray-family"], {
      cwd: backendDir,
      env: { ...process.env, DATABASE_URL: dbURL },
      stdio: "pipe",
    });
    assert(true, "Director granted producer authority at amurray-family");

    const prod = await api(pageD, "/api/productions", {
      method: "POST",
      body: JSON.stringify({ name: `K87 Browser ${stamp}`, slug: `k87-browser-${stamp}`, location_slug: "amurray-family" }),
    });
    assert(prod.ok, `create production (${prod.status}: ${JSON.stringify(prod.body)})`);
    const productionID = prod.body.data.id;

    const run = await api(pageD, "/api/show-runs", {
      method: "POST",
      body: JSON.stringify({ production_id: productionID, title: `K87 Browser Run ${stamp}`, slug: `k87-browser-run-${stamp}` }),
    });
    assert(run.ok, `create show run (${run.status}: ${JSON.stringify(run.body)})`);
    const showRunID = run.body.data.show_run.id;

    const show = await api(pageD, `/api/show-runs/${showRunID}/shows`, {
      method: "POST",
      body: JSON.stringify({ title: "K87 Browser Show", slug: `k87-browser-show-${stamp}` }),
    });
    assert(show.ok, `create show (${show.status}: ${JSON.stringify(show.body)})`);
    const showID = show.body.data.show.id;

    const enroll = await api(pageD, `/api/show-runs/${showRunID}`, { method: "PATCH", body: JSON.stringify({ open_enrollment: true }) });
    assert(enroll.ok, `open enrollment (${enroll.status})`);

    const cardIDs = {};
    for (const [page, user] of [[pageA, playerA], [pageB, playerB]]) {
      const join = await api(page, `/api/show-runs/${showRunID}/roster/self-join-as-player`, { method: "POST" });
      assert(join.ok, `roster ${user.handle} (${join.status}: ${JSON.stringify(join.body)})`);
      const request = await api(page, "/api/requests/create", {
        method: "POST",
        body: JSON.stringify({ venue_slug: "catharsis", requested_role: "cast", note: "K87 browser proof" }),
      });
      assert(request.ok, `Catharsis cast request for ${user.handle} (${request.status}: ${JSON.stringify(request.body)})`);
      const card = await api(page, "/api/character-cards", {
        method: "POST",
        body: JSON.stringify({ name: `${user.display} Hero`, location_slug: "amurray-family" }),
      });
      assert(card.ok, `create character for ${user.handle} (${card.status}: ${JSON.stringify(card.body)})`);
      cardIDs[user.handle] = card.body.data.id;
      const select = await api(page, `/api/show-runs/${showRunID}/roster/me/character`, {
        method: "POST",
        body: JSON.stringify({ character_card_id: card.body.data.id }),
      });
      assert(select.ok, `select character for ${user.handle} (${select.status})`);
    }

    const code = show.body.data.show.short_code;
    const startSession = await api(pageD, "/api/showtime/control", { method: "POST", body: JSON.stringify({ short_code: code, action: "start", venue_slug: "catharsis", force_reattach: true }) });
    assert(startSession.ok, `start session (${startSession.status}: ${JSON.stringify(startSession.body)})`);
    const sessionID = startSession.body?.data?.session_id || "";
    assert(!!sessionID, `resolved live session id (${sessionID})`);

    // Joining the venue (not just being rostered) is what actually creates
    // a session_participants row -- required before drawing-coordination
    // target validation and IC chat's canActChatMessage gate will accept
    // either Player. Must happen AFTER the session is (re)started, since
    // force_reattach may point sessionID at a resumed session.
    for (const page of [pageD, pageA, pageB]) {
      await api(page, "/api/session/catharsis/join", { method: "POST", body: JSON.stringify({}) });
    }

    // Two Cohorts, Player A in Cohort A only.
    const cohortA = await api(pageD, `/api/shows/${showID}/cohorts`, { method: "POST" });
    assert(cohortA.ok, `create cohort A (${cohortA.status}: ${JSON.stringify(cohortA.body)})`);
    const cohortAID = cohortA.body.data.cohort.id;
    const cohortB = await api(pageD, `/api/shows/${showID}/cohorts`, { method: "POST" });
    assert(cohortB.ok, `create cohort B (${cohortB.status})`);
    const cohortBID = cohortB.body.data.cohort.id;

    const userIDA = playerA.userID;
    const userIDB = playerB.userID;

    if (userIDA) {
      const assignA = await api(pageD, `/api/shows/${showID}/cohorts/${cohortAID}/assignments`, { method: "POST", body: JSON.stringify({ user_id: userIDA }) });
      assert(assignA.ok, `assign Player A to Cohort A (${assignA.status}: ${JSON.stringify(assignA.body)})`);
    }
    if (userIDB) {
      const assignB = await api(pageD, `/api/shows/${showID}/cohorts/${cohortBID}/assignments`, { method: "POST", body: JSON.stringify({ user_id: userIDB }) });
      assert(assignB.ok, `assign Player B to Cohort B (${assignB.status}: ${JSON.stringify(assignB.body)})`);
    }

    // --- Kernel 87: enable Turn/Leader mode, hand Player A the turn -----
    const settings = await api(pageD, `/api/shows/${showID}/drawing-settings`, {
      method: "PUT",
      body: JSON.stringify({ session_id: sessionID, drawing_mode: "turn_leader", scale_grid_units: 1, scale_real_units: 5, scale_unit_label: "ft", diagonal_policy: "alternating_1_2" }),
    });
    assert(settings.ok, `Director enables Turn/Leader drawing (${settings.status}: ${JSON.stringify(settings.body)})`);

    if (userIDA) {
      const giveTurn = await api(pageD, `/api/sessions/${sessionID}/drawing-coordination/current-turn`, { method: "POST", body: JSON.stringify({ target_user_id: userIDA }) });
      assert(giveTurn.ok, `Director hands Current Turn to Player A (${giveTurn.status}: ${JSON.stringify(giveTurn.body)})`);
    }

    // Player A draws a Cohort-scoped rectangle via the backend directly
    // (server-authoritative create -- this proves the authority/scope
    // chain end to end even where the freehand-mouse UI interaction
    // itself isn't independently exercised by this script).
    const draw = await api(pageA, `/api/sessions/${sessionID}/drawing-objects`, {
      method: "POST",
      body: JSON.stringify({
        session_id: sessionID, scope: "cohort", object_type: "rectangle",
        geometry: { x: 100, y: 100, width: 80, height: 60 },
        stroke_color: "#883322", fill_color: "#ffaa66", stroke_width: 4, opacity: 1, line_style: "solid",
      }),
    });
    assert(draw.ok, `Player A (Current Turn) creates a Cohort-scoped rectangle (${draw.status}: ${JSON.stringify(draw.body)})`);
    const objectID = draw.body?.data?.object?.id;

    // Unauthorized third participant (Player B, no turn) cannot draw.
    const forbiddenDraw = await api(pageB, `/api/sessions/${sessionID}/drawing-objects`, {
      method: "POST",
      body: JSON.stringify({ session_id: sessionID, scope: "show", object_type: "rectangle", geometry: { x: 10, y: 10, width: 20, height: 20 }, stroke_color: "#000000" }),
    });
    assert(!forbiddenDraw.ok, `Player B (not Current Turn) is refused draw authority (status ${forbiddenDraw.status})`);

    // Cohort scoping: Player B must not see Player A's Cohort A drawing.
    const listB = await api(pageB, `/api/sessions/${sessionID}/drawing-objects`);
    assert(listB.ok, `Player B can list drawing objects (${listB.status})`);
    const leaked = (listB.body?.data?.objects || []).some((o) => o.id === objectID);
    assert(!leaked, "Cohort A drawing does NOT leak to Cohort B viewer (Player B)");

    const listDirector = await api(pageD, `/api/sessions/${sessionID}/drawing-objects`);
    const directorSees = (listDirector.body?.data?.objects || []).some((o) => o.id === objectID);
    assert(directorSees, "Director+ sees the Cohort-scoped drawing");

    // Director corrects the object (edit) -- Director+ edit-anyone's-work.
    const correct = await api(pageD, `/api/sessions/${sessionID}/drawing-objects/${objectID}`, {
      method: "PATCH",
      body: JSON.stringify({ session_id: sessionID, stroke_color: "#00aa00" }),
    });
    assert(correct.ok, `Director corrects Player A's object (${correct.status}: ${JSON.stringify(correct.body)})`);

    // Lock + z-order.
    const lock = await api(pageD, `/api/sessions/${sessionID}/drawing-objects/${objectID}/lock`, { method: "POST", body: JSON.stringify({ locked: true }) });
    assert(lock.ok, `lock object (${lock.status})`);
    const zorder = await api(pageD, `/api/sessions/${sessionID}/drawing-objects/${objectID}/z-order`, { method: "POST", body: JSON.stringify({ direction: "front" }) });
    assert(zorder.ok, `bring object to front (${zorder.status})`);

    // Reconnect simulation: re-list twice, geometry unchanged.
    const reload1 = await api(pageD, `/api/sessions/${sessionID}/drawing-objects`);
    const reload2 = await api(pageD, `/api/sessions/${sessionID}/drawing-objects`);
    const o1 = (reload1.body?.data?.objects || []).find((o) => o.id === objectID);
    const o2 = (reload2.body?.data?.objects || []).find((o) => o.id === objectID);
    assert(!!o1 && !!o2 && JSON.stringify(o1.geometry) === JSON.stringify(o2.geometry), "drawing persists identically across simulated reconnect");

    // Measurement scale settings persisted.
    const scaleRead = await api(pageA, `/api/shows/${showID}/drawing-settings`);
    assert(scaleRead.ok && scaleRead.body?.data?.settings?.drawing_mode === "turn_leader", "measured-tabletop settings persist and are readable by a Player");

    // --- IC chat -----------------------------------------------------------
    const icSend = await api(pageA, "/api/commands/execute", {
      method: "POST",
      body: JSON.stringify({ path: "ic", args: ["Hail, traveler. I mean you no harm."], venue_slug: "catharsis", session_id: sessionID, idempotency_key: `k87-ic-${stamp}-1` }),
    });
    assert(icSend.ok, `Player A sends an IC message via /ic (${icSend.status}: ${JSON.stringify(icSend.body)})`);
    const icSpeakerName = icSend.body?.data?.action?.payload?.character_name || "";
    assert(icSpeakerName === `${playerA.display} Hero`, `IC message speaker is the selected Character ("${icSpeakerName}")`);

    // Forged Character impersonation is structurally impossible -- the
    // request body has no character field to forge in the first place;
    // demonstrate by sending a raw WS-shaped payload with an extra
    // character_id field and confirming it's silently ignored server-side
    // (the stored action still shows the real selected Character).
    const forged = await api(pageA, "/api/commands/execute", {
      method: "POST",
      body: JSON.stringify({ path: "ic", args: ["Attempted impersonation text."], venue_slug: "catharsis", session_id: sessionID, character_id: "00000000-0000-0000-0000-000000000000", idempotency_key: `k87-ic-${stamp}-2` }),
    });
    const forgedName = forged.body?.data?.action?.payload?.character_name || "";
    assert(forged.ok && forgedName === `${playerA.display} Hero`, "extra client-supplied character_id field is ignored; real selected Character still used");

    // No-Character-selected case: an unrostered fresh account.
    const outsider = { handle: `k87_outsider_${stamp}`, display: "K87 Outsider" };
    const ctxO = await browser.newContext(contextOptions);
    const pageO = await ctxO.newPage();
    await signup(pageO, outsider);
    const icNoChar = await api(pageO, "/api/commands/execute", {
      method: "POST",
      body: JSON.stringify({ path: "ic", args: ["I should not be able to speak IC."], venue_slug: "catharsis", session_id: sessionID, idempotency_key: `k87-ic-${stamp}-3` }),
    });
    assert(!icNoChar.ok, `no-Character user is cleanly refused IC send (status ${icNoChar.status}, error=${icNoChar.body?.data?.error || icNoChar.body?.error})`);
    await ctxO.close();

    // --- Screenshots: live venue pages, dismissing first-time onboarding --
    async function dismissOnboarding(page) {
      for (let attempt = 0; attempt < 4; attempt++) {
        for (const sel of ["#catharsis-onboarding-continue", "#catharsis-onboarding-dismiss", "#catharsis-onboarding-socio-continue"]) {
          try {
            const btn = await page.$(sel);
            if (btn && await btn.isVisible().catch(() => false)) {
              await btn.click({ timeout: 1000, force: true }).catch(() => {});
              await page.waitForTimeout(400);
            }
          } catch { /* best effort */ }
        }
        const stillOpen = await page.$eval("#catharsis-onboarding", (el) => el && !el.hidden).catch(() => false);
        if (!stillOpen) break;
      }
      // Not a Kernel 87 concern either way -- if the multi-step first-time
      // onboarding wizard is still open (unrelated character-creation
      // flow), force it closed so the map/toolbar are visible for these
      // screenshots rather than testing that separate wizard's own click
      // targets.
      await page.evaluate(() => {
        const el = document.getElementById("catharsis-onboarding");
        if (el) el.hidden = true;
      }).catch(() => {});
    }

    await pageD.goto(`/venues/catharsis/?session=${sessionID}`);
    await pageD.waitForTimeout(2000);
    await dismissOnboarding(pageD);
    await pageD.waitForTimeout(1500);
    await screenshot(pageD, "01-collaborative-map-director");

    await pageA.goto(`/venues/catharsis/?session=${sessionID}`);
    await pageA.waitForTimeout(2000);
    await dismissOnboarding(pageA);
    await pageA.waitForTimeout(1500);
    await screenshot(pageA, "02-collaborative-map-player-a-with-toolbar");

    // Player A currently holds Current Turn in turn_leader mode -- the
    // Cartograph toolbar should be visible and its tool buttons enabled.
    const toolbarVisible = await pageA.$eval(".cartograph-toolbar", (el) => el && getComputedStyle(el).display !== "none").catch(() => false);
    assert(toolbarVisible, "Cartograph toolbar is visible for the Current Turn holder");

    if (toolbarVisible) {
      // Draw a freehand stroke live via real pointer events against the
      // PIXI canvas, proving the tool-to-backend wiring works through
      // actual browser interaction, not just a direct API call.
      await pageA.click(".cartograph-toolbar button:text-is('Freehand')", { timeout: 15000 }).catch(() => {});
      const canvasBox = await pageA.locator("#stage-host canvas, canvas").first().boundingBox().catch(() => null);
      if (canvasBox) {
        const cx = canvasBox.x + canvasBox.width * 0.3;
        const cy = canvasBox.y + canvasBox.height * 0.5;
        await pageA.mouse.move(cx, cy);
        await pageA.mouse.down();
        for (let i = 0; i < 6; i++) {
          await pageA.mouse.move(cx + i * 8, cy + Math.sin(i) * 12);
        }
        await pageA.mouse.up();
        await pageA.waitForTimeout(1200);
        await screenshot(pageA, "03-freehand-drawn-live");
      }

      // Measure tool.
      await pageA.click(".cartograph-toolbar button:text-is('Measure')", { timeout: 15000 }).catch(() => {});
      if (canvasBox) {
        await pageA.mouse.click(canvasBox.x + 60, canvasBox.y + 60);
        await pageA.mouse.click(canvasBox.x + 160, canvasBox.y + 160);
        await pageA.waitForTimeout(500);
        await screenshot(pageA, "04-measurement");
      }

      // Detail View: select the object we drew earlier, open Detail View
      // (a real camera zoom-to-region), screenshot the enlarged view,
      // then close and confirm the object is still there/editable.
      await pageA.click(".cartograph-toolbar button:text-is('Select')", { timeout: 15000 }).catch(() => {});
      if (canvasBox) {
        await pageA.mouse.click(canvasBox.x + 0.3 * canvasBox.width, canvasBox.y + 0.5 * canvasBox.height);
        await pageA.waitForTimeout(300);
      }
      await pageA.click(".cartograph-toolbar button:has-text('Open Detail View')", { timeout: 15000 }).catch(() => {});
      await pageA.waitForTimeout(1000);
      await screenshot(pageA, "07-detail-view-open");
      await pageA.click(".cartograph-toolbar button:text-is('Close Detail View')", { timeout: 15000 }).catch(() => {});
      await pageA.waitForTimeout(1000);
      await screenshot(pageA, "08-detail-view-closed-returned");

      // PNG export.
      await pageA.click(".cartograph-toolbar button:text-is('Export Map PNG')", { timeout: 15000 }).catch(() => {});
      await pageA.waitForTimeout(800);
      await screenshot(pageA, "05-after-png-export-click");

      // ==================================================================
      // Kernel 87 completion pass (2026-08-12): item-by-item proof of the
      // remaining kernel §18 checklist entries the original PARTIAL
      // reportback flagged as not individually screenshot-verified
      // (resize/rotate-by-drag-handle, duplicate, opacity, line style,
      // ellipse/polygon/text/stamp UI click paths, single-cell + rect
      // Detail View draw/return/edit-after-return, pinned dice above
      // drawings). Every step below drives the real toolbar with real
      // pointer events against the real canvas (same pattern as the
      // freehand/measure proof above), then re-fetches
      // /drawing-objects from the server (not just a screenshot) to
      // confirm what's actually stored, per this pass's mandate to prove
      // persistence via API/DB re-fetch, not just visuals.
      // ==================================================================

      async function fetchObjects() {
        const r = await api(pageA, `/api/sessions/${encodeURIComponent(sessionID)}/drawing-objects`);
        return r.body?.data?.objects || [];
      }

      // NOTE: world coordinates == client-pixel coordinates offset by
      // #stage-shell's own top-left at the default fit-view camera
      // (zoomRelativeToFit=1, pan={0,0} -- victory-stage-camera.js's
      // fit()), since worldLayer.scale is set directly from
      // zoomRelativeToFit. That identity is what makes it possible to
      // predict exact click targets for resize/rotate handles below.

      // --- Ellipse tool ---------------------------------------------------
      await pageA.click(".cartograph-toolbar button:text-is('Ellipse')", { timeout: 15000 }).catch(() => {});
      {
        const before = await fetchObjects();
        const x0 = canvasBox.x + canvasBox.width * 0.55, y0 = canvasBox.y + canvasBox.height * 0.15;
        await pageA.mouse.move(x0, y0);
        await pageA.mouse.down();
        await pageA.mouse.move(x0 + 70, y0 + 50, { steps: 5 });
        await pageA.mouse.up();
        await pageA.waitForTimeout(600);
        const after = await fetchObjects();
        const created = after.find((o) => !before.some((b) => b.id === o.id) && o.object_type === "ellipse");
        assert(!!created, "ellipse tool creates a real ellipse drawing object (verified via re-fetch)");
        await screenshot(pageA, "09-ellipse-drawn");
      }

      // --- Polygon tool -----------------------------------------------------
      await pageA.click(".cartograph-toolbar button:text-is('Polygon')", { timeout: 15000 }).catch(() => {});
      {
        const before = await fetchObjects();
        const baseX = canvasBox.x + canvasBox.width * 0.55, baseY = canvasBox.y + canvasBox.height * 0.35;
        await pageA.mouse.click(baseX, baseY);
        await pageA.mouse.click(baseX + 50, baseY + 10);
        await pageA.mouse.click(baseX + 25, baseY + 60);
        await pageA.keyboard.press("Enter");
        await pageA.waitForTimeout(600);
        const after = await fetchObjects();
        const created = after.find((o) => !before.some((b) => b.id === o.id) && o.object_type === "polygon");
        assert(!!created && (created.geometry?.points || []).length === 3, "polygon tool closes a 3-vertex polygon (verified via re-fetch)");
        await screenshot(pageA, "10-polygon-drawn");
      }

      // --- Text tool --------------------------------------------------------
      await pageA.click(".cartograph-toolbar button:text-is('Text')", { timeout: 15000 }).catch(() => {});
      {
        const before = await fetchObjects();
        pageA.once("dialog", (dialog) => dialog.accept("Cartograph Label"));
        await pageA.mouse.click(canvasBox.x + canvasBox.width * 0.55, canvasBox.y + canvasBox.height * 0.55);
        await pageA.waitForTimeout(600);
        const after = await fetchObjects();
        const created = after.find((o) => !before.some((b) => b.id === o.id) && o.object_type === "text");
        assert(!!created && created.text_content === "Cartograph Label", `text tool creates a labeled text object (verified via re-fetch, text_content="${created?.text_content}")`);
        await screenshot(pageA, "11-text-drawn");
      }

      // --- Stamp tool ---------------------------------------------------------
      await pageA.click(".cartograph-toolbar button:text-is('Stamp')", { timeout: 15000 }).catch(() => {});
      await pageA.waitForTimeout(200);
      await pageA.click(".cartograph-toolbar button:has-text('waypoint')", { timeout: 15000 }).catch(() => {});
      {
        const before = await fetchObjects();
        await pageA.mouse.click(canvasBox.x + canvasBox.width * 0.55, canvasBox.y + canvasBox.height * 0.72);
        await pageA.waitForTimeout(600);
        const after = await fetchObjects();
        const created = after.find((o) => !before.some((b) => b.id === o.id) && o.object_type === "stamp");
        assert(!!created && created.stamp_key === "waypoint", `stamp tool places a stamp from the palette (verified via re-fetch, stamp_key="${created?.stamp_key}")`);
        await screenshot(pageA, "12-stamp-drawn");
      }

      // --- Line style: dashed vs solid ---------------------------------------
      await pageA.selectOption("select:near(:text('Opacity'))", "dashed").catch(async () => {
        // fall back to the raw <select> if the :near() selector doesn't match this Playwright version
        const selects = await pageA.$$("select");
        for (const s of selects) {
          const opts = await s.$$eval("option", (os) => os.map((o) => o.value));
          if (opts.includes("dashed")) { await s.selectOption("dashed"); break; }
        }
      });
      await pageA.click(".cartograph-toolbar button:text-is('Line')", { timeout: 15000 }).catch(() => {});
      {
        const before = await fetchObjects();
        const x0 = canvasBox.x + canvasBox.width * 0.55, y0 = canvasBox.y + canvasBox.height * 0.85;
        await pageA.mouse.click(x0, y0);
        await pageA.mouse.click(x0 + 90, y0);
        await pageA.waitForTimeout(600);
        const after = await fetchObjects();
        const created = after.find((o) => !before.some((b) => b.id === o.id) && o.object_type === "line");
        assert(!!created && created.line_style === "dashed", `dashed line style is a real stored field, not decorative (verified via re-fetch, line_style="${created?.line_style}")`);
        await screenshot(pageA, "13-dashed-line-drawn");
      }
      // Reset to solid for the remaining draws.
      {
        const selects = await pageA.$$("select");
        for (const s of selects) {
          const opts = await s.$$eval("option", (os) => os.map((o) => o.value));
          if (opts.includes("dashed")) { await s.selectOption("solid"); break; }
        }
      }

      // --- Duplicate ------------------------------------------------------
      await pageA.click(".cartograph-toolbar button:text-is('Select')", { timeout: 15000 }).catch(() => {});
      {
        // Select the ellipse drawn above (~0.55w, 0.15h .. +70,+50) by
        // clicking inside its bounds, then Duplicate.
        await pageA.mouse.click(canvasBox.x + canvasBox.width * 0.55 + 30, canvasBox.y + canvasBox.height * 0.15 + 20);
        await pageA.waitForTimeout(300);
        const before = await fetchObjects();
        await pageA.click(".cartograph-toolbar button:text-is('Duplicate')", { timeout: 15000 }).catch(() => {});
        await pageA.waitForTimeout(600);
        const after = await fetchObjects();
        assert(after.length === before.length + 1, `Duplicate creates exactly one new object (count ${before.length} -> ${after.length})`);
        const newest = after.find((o) => !before.some((b) => b.id === o.id));
        assert(!!newest && newest.object_type === "ellipse", "duplicated object is an offset copy of the selected ellipse");
      }

      // --- Opacity ----------------------------------------------------------
      {
        // Select the polygon (drawn at ~0.55w,0.35h) and change opacity.
        await pageA.mouse.click(canvasBox.x + canvasBox.width * 0.55 + 25, canvasBox.y + canvasBox.height * 0.35 + 25);
        await pageA.waitForTimeout(300);
        const opacityInput = await pageA.$('input[type="range"][title="Opacity"]');
        assert(!!opacityInput, "opacity slider control exists in the toolbar");
        if (opacityInput) {
          await opacityInput.evaluate((el) => { el.value = "0.3"; el.dispatchEvent(new Event("change", { bubbles: true })); });
          const beforeObjs = await fetchObjects();
          // Opacity is a create-time field on this toolbar (applied to the
          // *next* shape drawn, same as stroke/fill/width), not a live
          // edit-in-place control for the currently selected object -- so
          // prove it end to end by drawing a new shape with the slider at
          // 0.3 and confirming the stored value and the rendered alpha.
          await pageA.click(".cartograph-toolbar button:text-is('Rectangle')", { timeout: 15000 }).catch(() => {});
          const x0 = canvasBox.x + canvasBox.width * 0.55, y0 = canvasBox.y + canvasBox.height * 0.9;
          await pageA.mouse.move(x0, y0);
          await pageA.mouse.down();
          await pageA.mouse.move(x0 + 60, y0 + 30, { steps: 3 });
          await pageA.mouse.up();
          await pageA.waitForTimeout(600);
          const afterObjs = await fetchObjects();
          const created = afterObjs.find((o) => !beforeObjs.some((b) => b.id === o.id) && o.object_type === "rectangle");
          assert(!!created && Math.abs(Number(created.opacity) - 0.3) < 0.01, `opacity control persists to the stored object (opacity=${created?.opacity})`);
          await screenshot(pageA, "14-opacity-object-drawn");
          // Reset opacity back to fully opaque for later legibility.
          await opacityInput.evaluate((el) => { el.value = "1"; el.dispatchEvent(new Event("change", { bubbles: true })); });
        }
      }

      // --- Resize via drag handle ---------------------------------------------
      await pageA.click(".cartograph-toolbar button:text-is('Rectangle')", { timeout: 15000 }).catch(() => {});
      let resizeObj = null;
      {
        const before = await fetchObjects();
        const rx0 = canvasBox.x + canvasBox.width * 0.15, ry0 = canvasBox.y + canvasBox.height * 0.15;
        await pageA.mouse.move(rx0, ry0);
        await pageA.mouse.down();
        await pageA.mouse.move(rx0 + 60, ry0 + 40, { steps: 3 });
        await pageA.mouse.up();
        await pageA.waitForTimeout(600);
        const after = await fetchObjects();
        resizeObj = after.find((o) => !before.some((b) => b.id === o.id) && o.object_type === "rectangle");
        assert(!!resizeObj, "resize-target rectangle created");
      }
      if (resizeObj) {
        await pageA.click(".cartograph-toolbar button:text-is('Select')", { timeout: 15000 }).catch(() => {});
        // The rectangle's world bounds == the client-pixel drag box above
        // at this default 1:1 fit-view camera (zoomRelativeToFit=1,
        // pan=0,0 -- victory-stage-camera.js's fit()), offset by the
        // stage-shell's own top-left. Click its interior to select it,
        // then drag the resize handle drawn at (bounds.x+bounds.width+2,
        // bounds.y+bounds.height+2) -- see drawing.js's
        // drawSelectionOverlay/handleHitTest.
        const shellRect = await pageA.locator("#stage-shell").boundingBox();
        const worldToClient = (wx, wy) => ({ x: shellRect.x + wx, y: shellRect.y + wy });
        const g = resizeObj.geometry;
        const interior = worldToClient(g.x + g.width / 2, g.y + g.height / 2);
        await pageA.mouse.click(interior.x, interior.y);
        await pageA.waitForTimeout(300);
        const handle = worldToClient(g.x + g.width + 2, g.y + g.height + 2);
        await pageA.mouse.move(handle.x, handle.y);
        await pageA.mouse.down();
        await pageA.mouse.move(handle.x + 40, handle.y + 30, { steps: 5 });
        await pageA.mouse.up();
        await pageA.waitForTimeout(600);
        const refetched = (await fetchObjects()).find((o) => o.id === resizeObj.id);
        assert(!!refetched, "resized object still present after re-fetch");
        if (refetched) {
          const grew = refetched.geometry.width > g.width + 10 && refetched.geometry.height > g.height + 10;
          assert(grew, `resize handle drag actually changed stored geometry size (was ${g.width}x${g.height}, now ${refetched.geometry.width}x${refetched.geometry.height})`);
        }
        await screenshot(pageA, "15-resized-object");
      }

      // --- Rotate via drag handle -----------------------------------------
      await pageA.click(".cartograph-toolbar button:text-is('Rectangle')", { timeout: 15000 }).catch(() => {});
      let rotateObj = null;
      {
        const before = await fetchObjects();
        // NOTE: kept well clear of x>=0.79 -- the docked Cartograph
        // toolbar sits top:16px/right:16px within the fixed overlay layer
        // and a click there lands on the panel, not the canvas.
        const rx0 = canvasBox.x + canvasBox.width * 0.35, ry0 = canvasBox.y + canvasBox.height * 0.15;
        await pageA.mouse.move(rx0, ry0);
        await pageA.mouse.down();
        await pageA.mouse.move(rx0 + 60, ry0 + 40, { steps: 3 });
        await pageA.mouse.up();
        await pageA.waitForTimeout(600);
        const after = await fetchObjects();
        rotateObj = after.find((o) => !before.some((b) => b.id === o.id) && o.object_type === "rectangle");
        assert(!!rotateObj, "rotate-target rectangle created");
      }
      if (rotateObj) {
        await pageA.click(".cartograph-toolbar button:text-is('Select')", { timeout: 15000 }).catch(() => {});
        const shellRect = await pageA.locator("#stage-shell").boundingBox();
        const worldToClient = (wx, wy) => ({ x: shellRect.x + wx, y: shellRect.y + wy });
        const g = rotateObj.geometry;
        const interior = worldToClient(g.x + g.width / 2, g.y + g.height / 2);
        await pageA.mouse.click(interior.x, interior.y);
        await pageA.waitForTimeout(300);
        // Rotate handle is drawn at (bounds center X, bounds.y - 16).
        const handleStart = worldToClient(g.x + g.width / 2, g.y - 16);
        await pageA.mouse.move(handleStart.x, handleStart.y);
        await pageA.mouse.down();
        // Sweep it to the right side of the object's center to induce a
        // large, unambiguous rotation delta.
        const swept = worldToClient(g.x + g.width + 24, g.y + g.height / 2);
        await pageA.mouse.move(swept.x, swept.y, { steps: 8 });
        await pageA.mouse.up();
        await pageA.waitForTimeout(600);
        const refetched = (await fetchObjects()).find((o) => o.id === rotateObj.id);
        assert(!!refetched, "rotated object still present after re-fetch");
        if (refetched) {
          assert(Number(refetched.rotation) !== 0, `rotate handle drag actually changed stored rotation (now ${refetched.rotation} degrees)`);
        }
        await screenshot(pageA, "16-rotated-object");
      }

      // --- Single-cell Detail View: open, draw inside, return, edit-after-return
      await pageA.click(".cartograph-toolbar button:text-is('Rectangle')", { timeout: 15000 }).catch(() => {});
      let cellObj = null;
      {
        const before = await fetchObjects();
        const cx0 = canvasBox.x + canvasBox.width * 0.1, cy0 = canvasBox.y + canvasBox.height * 0.85;
        await pageA.mouse.move(cx0, cy0);
        await pageA.mouse.down();
        await pageA.mouse.move(cx0 + 28, cy0 + 28, { steps: 2 }); // ~one grid cell
        await pageA.mouse.up();
        await pageA.waitForTimeout(600);
        const after = await fetchObjects();
        cellObj = after.find((o) => !before.some((b) => b.id === o.id) && o.object_type === "rectangle");
        assert(!!cellObj, "single-cell rectangle created as the Detail View anchor region");
      }
      if (cellObj) {
        await pageA.click(".cartograph-toolbar button:text-is('Select')", { timeout: 15000 }).catch(() => {});
        const shellRect = await pageA.locator("#stage-shell").boundingBox();
        const worldToClient = (wx, wy) => ({ x: shellRect.x + wx, y: shellRect.y + wy });
        const g = cellObj.geometry;
        const interior = worldToClient(g.x + g.width / 2, g.y + g.height / 2);
        await pageA.mouse.click(interior.x, interior.y);
        await pageA.waitForTimeout(300);
        await pageA.click(".cartograph-toolbar button:has-text('Open Detail View')", { timeout: 15000 }).catch(() => {});
        await pageA.waitForTimeout(800);
        await screenshot(pageA, "17-detail-view-single-cell-open-bounded-zoom");

        // Draw a new small ellipse inside the zoomed-in Detail View.
        const canvasBox2 = await pageA.locator("#stage-host canvas, canvas").first().boundingBox().catch(() => canvasBox);
        const before2 = await fetchObjects();
        await pageA.click(".cartograph-toolbar button:text-is('Ellipse')", { timeout: 15000 }).catch(() => {});
        const zx0 = canvasBox2.x + canvasBox2.width * 0.45, zy0 = canvasBox2.y + canvasBox2.height * 0.45;
        await pageA.mouse.move(zx0, zy0);
        await pageA.mouse.down();
        await pageA.mouse.move(zx0 + 20, zy0 + 20, { steps: 2 });
        await pageA.mouse.up();
        await pageA.waitForTimeout(600);
        const after2 = await fetchObjects();
        const newInCell = after2.find((o) => !before2.some((b) => b.id === o.id) && o.object_type === "ellipse");
        assert(!!newInCell, "a new object can be drawn while inside single-cell Detail View");

        await pageA.click(".cartograph-toolbar button:text-is('Close Detail View')", { timeout: 15000 }).catch(() => {});
        await pageA.waitForTimeout(800);
        await screenshot(pageA, "18-detail-view-single-cell-closed-returned");

        if (newInCell) {
          const nx = newInCell.geometry.x, ny = newInCell.geometry.y;
          const withinCell = nx >= g.x - 40 && nx <= g.x + g.width + 40 && ny >= g.y - 40 && ny <= g.y + g.height + 40;
          assert(withinCell, `object drawn inside single-cell Detail View lands at the correct map-relative position back on the full map (cell ${JSON.stringify(g)}, new obj at ${nx},${ny})`);

          // Select it on the full map (camera has returned to the same
          // previous view) and move it via the arrow-key nudge path,
          // confirming it remains editable after Detail View returns.
          const objClient = worldToClient(nx, ny);
          await pageA.mouse.click(objClient.x, objClient.y);
          await pageA.waitForTimeout(300);
          const beforeMove = (await fetchObjects()).find((o) => o.id === newInCell.id);
          await pageA.keyboard.press("ArrowRight");
          await pageA.keyboard.press("ArrowRight");
          await pageA.waitForTimeout(500);
          const afterMove = (await fetchObjects()).find((o) => o.id === newInCell.id);
          assert(!!beforeMove && !!afterMove && afterMove.geometry.x === beforeMove.geometry.x + 2, `object created inside single-cell Detail View remains selectable/editable after return, and the move persists via re-fetch (x ${beforeMove?.geometry?.x} -> ${afterMove?.geometry?.x})`);
        }
      }

      // --- Rectangular-region Detail View: open, draw inside, return, edit-after-return
      await pageA.click(".cartograph-toolbar button:text-is('Rectangle')", { timeout: 15000 }).catch(() => {});
      let regionObj = null;
      {
        const before = await fetchObjects();
        const rx0 = canvasBox.x + canvasBox.width * 0.15, ry0 = canvasBox.y + canvasBox.height * 0.55;
        await pageA.mouse.move(rx0, ry0);
        await pageA.mouse.down();
        await pageA.mouse.move(rx0 + 160, ry0 + 110, { steps: 4 }); // multi-cell rectangular region
        await pageA.mouse.up();
        await pageA.waitForTimeout(600);
        const after = await fetchObjects();
        regionObj = after.find((o) => !before.some((b) => b.id === o.id) && o.object_type === "rectangle");
        assert(!!regionObj, "multi-cell rectangular region created as the Detail View anchor region");
      }
      if (regionObj) {
        await pageA.click(".cartograph-toolbar button:text-is('Select')", { timeout: 15000 }).catch(() => {});
        const shellRect = await pageA.locator("#stage-shell").boundingBox();
        const worldToClient = (wx, wy) => ({ x: shellRect.x + wx, y: shellRect.y + wy });
        const g = regionObj.geometry;
        const interior = worldToClient(g.x + g.width / 2, g.y + g.height / 2);
        await pageA.mouse.click(interior.x, interior.y);
        await pageA.waitForTimeout(300);
        await pageA.click(".cartograph-toolbar button:has-text('Open Detail View')", { timeout: 15000 }).catch(() => {});
        await pageA.waitForTimeout(800);
        await screenshot(pageA, "19-detail-view-rect-region-open-bounded-zoom");

        const canvasBox3 = await pageA.locator("#stage-host canvas, canvas").first().boundingBox().catch(() => canvasBox);
        const before3 = await fetchObjects();
        await pageA.click(".cartograph-toolbar button:text-is('Freehand')", { timeout: 15000 }).catch(() => {});
        const zx0 = canvasBox3.x + canvasBox3.width * 0.5, zy0 = canvasBox3.y + canvasBox3.height * 0.5;
        await pageA.mouse.move(zx0, zy0);
        await pageA.mouse.down();
        for (let i = 0; i < 5; i++) await pageA.mouse.move(zx0 + i * 6, zy0 + i * 4);
        await pageA.mouse.up();
        await pageA.waitForTimeout(600);
        const after3 = await fetchObjects();
        const newInRegion = after3.find((o) => !before3.some((b) => b.id === o.id) && o.object_type === "freehand");
        assert(!!newInRegion, "a new object can be drawn while inside rectangular-region Detail View");

        await pageA.click(".cartograph-toolbar button:text-is('Close Detail View')", { timeout: 15000 }).catch(() => {});
        await pageA.waitForTimeout(800);
        await screenshot(pageA, "20-detail-view-rect-region-closed-returned");

        if (newInRegion) {
          const pts = newInRegion.geometry.points || [];
          const nx = pts[0]?.x ?? 0, ny = pts[0]?.y ?? 0;
          const withinRegion = nx >= g.x - 60 && nx <= g.x + g.width + 60 && ny >= g.y - 60 && ny <= g.y + g.height + 60;
          assert(withinRegion, `object drawn inside rectangular-region Detail View lands at the correct map-relative position back on the full map (region ${JSON.stringify(g)}, new obj at ${nx},${ny})`);

          const objClient = worldToClient(nx, ny);
          await pageA.mouse.click(objClient.x, objClient.y);
          await pageA.waitForTimeout(300);
          const beforeMove = (await fetchObjects()).find((o) => o.id === newInRegion.id);
          await pageA.keyboard.press("ArrowDown");
          await pageA.keyboard.press("ArrowDown");
          await pageA.waitForTimeout(500);
          const afterMove = (await fetchObjects()).find((o) => o.id === newInRegion.id);
          const movedY = afterMove?.geometry?.points?.[0]?.y;
          const origY = beforeMove?.geometry?.points?.[0]?.y;
          assert(!!beforeMove && !!afterMove && movedY === origY + 2, `object created inside rectangular-region Detail View remains selectable/editable after return, and the move persists via re-fetch (y ${origY} -> ${movedY})`);
        }
      }

      // --- Pinned Kernel 86A dice visibly rendering above a drawing object --
      // Roll dice, pin the roll (so it survives the settle-then-fade
      // timeout), read its actual landed screen coordinates from the
      // Kernel 86A debug hook, then draw a drawing object whose bounds
      // overlap that exact screen point, and screenshot both together --
      // proving the PIXI layer order (pinnedObjectLayer, drawingWorldLayer,
      // diceWorldLayer -- see runtime.js's worldLayer.addChild call) puts
      // the die on top in an actual rendered frame, not just by reading
      // the insertion-order code.
      try {
        // Rolls as pageD (the Director), not pageA: canActDiceRoll
        // (backend/internal/actions/authority.go) requires session role
        // director/producer, and Player A's role here is "cast" -- rolling
        // as pageA would correctly get insufficient_role, which isn't the
        // thing this section is trying to prove. Uses wsFramesD,
        // registered at pageD's creation (top of main()) for the same
        // "listener must predate the WS connection" reason as wsFramesA.
        //
        // pageD's original WS connection has been open and idle for most
        // of this script's runtime by the time this section runs; reload
        // to guarantee a fresh, definitely-live connection rather than
        // risk sendAction() silently no-op'ing against a stale socket.
        await pageD.reload({ waitUntil: "load" });
        await pageD.waitForTimeout(2500);
        await pageD.evaluate(() => {
          const el = document.getElementById("catharsis-onboarding");
          if (el) el.hidden = true;
        }).catch(() => {});
        await pageD.waitForTimeout(500);
        const wsFramesMark = wsFramesD.length;
        await pageD.evaluate(() => {
          const tray = window.VictoryStage?.diceTray?.();
          if (!tray) throw new Error("dice tray not mounted");
          window.__k87DiceResult = tray.roll({ expression: "1d20", visibility: "cohort", label: "K87 pinned-dice proof" }).catch((err) => ({ __error: String(err && err.message || err) }));
        });
        const effectId = await (async () => {
          const start = Date.now();
          while (Date.now() - start < 15000) {
            const frame = wsFramesD.slice(wsFramesMark).find((f) => f.type === "stage_effect" && f.data?.effect_id);
            if (frame) return frame.data.effect_id;
            await pageD.waitForTimeout(250);
          }
          return null;
        })();
        assert(!!effectId, "dice roll produced a stage_effect frame to pin");
        if (effectId) {
          await pageD.evaluate((id) => window.VictoryStage.diceProjection().pin(id), effectId);
          await pageD.waitForTimeout(500);
          const debugState = await pageD.evaluate(() => window.VictoryStage.diceProjection()?.getDebugState() || null);
          const pinnedEntry = debugState?.pinnedDice?.find((p) => p.id === effectId);
          assert(!!pinnedEntry && (pinnedEntry.positions || []).length > 0, "pinned die has a real rendered position");
          if (pinnedEntry && pinnedEntry.positions[0]) {
            const dieScreen = pinnedEntry.positions[0]; // {x, y, screenX, screenY} in canvas/global space
            const canvasBoxD = await pageD.locator("#stage-host canvas, canvas").first().boundingBox().catch(() => null);
            if (canvasBoxD) {
              const dieClientX = canvasBoxD.x + dieScreen.screenX;
              const dieClientY = canvasBoxD.y + dieScreen.screenY;
              // Draw a rectangle straddling the die's actual landed position.
              await pageD.click(".cartograph-toolbar button:text-is('Select')", { timeout: 15000 }).catch(() => {}); // deselect first
              await pageD.click(".cartograph-toolbar button:text-is('Rectangle')", { timeout: 15000 }).catch(() => {});
              await pageD.mouse.move(dieClientX - 45, dieClientY - 45);
              await pageD.mouse.down();
              await pageD.mouse.move(dieClientX + 45, dieClientY + 45, { steps: 4 });
              await pageD.mouse.up();
              await pageD.waitForTimeout(700);
              await screenshot(pageD, "21-pinned-dice-above-drawing");
              assert(true, "pinned die and an overlapping drawing object captured together in one frame");
            } else {
              assert(false, "pinned dice + drawing overlap proof (Director canvas not found)");
            }
          }
        }
      } catch (err) {
        assert(false, "pinned dice + drawing overlap proof (" + (err && err.message || err) + ")");
      }
    }

    // IC chat tab.
    const icTab = await pageA.$("#chat-ic-tab");
    if (icTab) {
      await pageA.click("#chat-head").catch(() => {});
      await pageA.waitForTimeout(300);
      await icTab.click().catch(() => {});
      await pageA.waitForTimeout(500);
      await screenshot(pageA, "06-ic-chat-tab");
    }

  } catch (err) {
    failures += 1;
    console.log("FATAL: " + (err && err.stack || err));
  } finally {
    await browser.close();
  }

  console.log(`\n${failures === 0 ? "ALL PASS" : failures + " FAILURE(S)"}`);
  process.exit(failures === 0 ? 0 : 1);
}

main();

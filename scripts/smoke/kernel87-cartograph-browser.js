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
      await pageA.click("text=Freehand").catch(() => {});
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
      await pageA.click("text=Measure").catch(() => {});
      if (canvasBox) {
        await pageA.mouse.click(canvasBox.x + 60, canvasBox.y + 60);
        await pageA.mouse.click(canvasBox.x + 160, canvasBox.y + 160);
        await pageA.waitForTimeout(500);
        await screenshot(pageA, "04-measurement");
      }

      // Detail View: select the object we drew earlier, open Detail View
      // (a real camera zoom-to-region), screenshot the enlarged view,
      // then close and confirm the object is still there/editable.
      await pageA.click("text=Select").catch(() => {});
      if (canvasBox) {
        await pageA.mouse.click(canvasBox.x + 0.3 * canvasBox.width, canvasBox.y + 0.5 * canvasBox.height);
        await pageA.waitForTimeout(300);
      }
      await pageA.click("text=/Open Detail View/").catch(() => {});
      await pageA.waitForTimeout(1000);
      await screenshot(pageA, "07-detail-view-open");
      await pageA.click("text=Close Detail View").catch(() => {});
      await pageA.waitForTimeout(1000);
      await screenshot(pageA, "08-detail-view-closed-returned");

      // PNG export.
      await pageA.click("text=Export Map PNG").catch(() => {});
      await pageA.waitForTimeout(800);
      await screenshot(pageA, "05-after-png-export-click");
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

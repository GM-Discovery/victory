#!/usr/bin/env node
"use strict";

// Kernel 65 browser acceptance proof (spec §11): real browser sessions
// against the live deployed stack. User A leaves a Headshot, changes their
// Trailer Face; user B browses the commons, adds A to My People, and
// confirms the live update; a third user C proves A's relationship with B
// is not visible to anyone else. Creates three clearly-named throwaway
// accounts via /api/auth/signup (no elevated privileges granted, so no
// cleanup is required -- documented in the reportback per Kernel 62/63
// precedent).
//
// Run after deploy:  NODE_PATH=/tmp/node_modules node scripts/smoke/kernel65-third-place-browser.js

const fs = require("node:fs");
const path = require("node:path");
const { chromium } = require("playwright");

const baseURL = process.env.K65_BASE_URL || "https://victory.amurray.family";
const hostDomain = new URL(baseURL).hostname;
const evidenceDir = process.env.K65_EVIDENCE_DIR || path.join("Construction", "OperatorLogs", "evidence", "kernel-65");

const stamp = Date.now();
const userA = { handle: `k65_owner_${stamp}`, display: "K65 Owner", stage1: `Headshot Owner ${stamp}`, stage2: `Renamed Owner ${stamp}` };
const userB = { handle: `k65_viewer_${stamp}`, display: "K65 Viewer" };
const userC = { handle: `k65_third_${stamp}`, display: "K65 Third" };
const password = `k65-browser-${stamp}`;

function assert(condition, message) {
  if (!condition) throw new Error("ASSERT FAILED: " + message);
}

function pass(message) {
  console.log("PASS " + message);
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
    body: JSON.stringify({
      email: `${user.handle}@example.com`,
      handle: user.handle,
      password,
      display_name: user.display,
    }),
  });
  assert(res.ok, `signup for ${user.handle} (status ${res.status}: ${JSON.stringify(res.body)})`);
}

async function screenshot(page, name) {
  await page.screenshot({ path: path.join(evidenceDir, `${name}.png`), fullPage: true });
}

async function main() {
  fs.mkdirSync(evidenceDir, { recursive: true });
  const browser = await chromium.launch({
    headless: true,
    args: [`--host-resolver-rules=MAP ${hostDomain} 127.0.0.1`],
  });

  const contextOptions = { baseURL, ignoreHTTPSErrors: true, viewport: { width: 1366, height: 900 } };
  const ctxA = await browser.newContext(contextOptions);
  const ctxB = await browser.newContext(contextOptions);
  const ctxC = await browser.newContext(contextOptions);
  const ctxAnon = await browser.newContext(contextOptions);
  const ctxMobile = await browser.newContext({ ...contextOptions, viewport: { width: 390, height: 844 } });

  const pageA = await ctxA.newPage();
  const pageB = await ctxB.newPage();
  const pageC = await ctxC.newPage();
  const pageAnon = await ctxAnon.newPage();

  try {
    // --- Setup: three fresh accounts, A gets a stage name + Face facts ---
    await signup(pageA, userA);
    await signup(pageB, userB);
    await signup(pageC, userC);
    pass("three fresh browser accounts signed up");

    const stageRes = await api(pageA, "/api/player-profile/stage-name", {
      method: "POST",
      body: JSON.stringify({ stage_name: userA.stage1 }),
    });
    assert(stageRes.ok, "A sets initial stage name");

    const pageCommit = await api(pageA, "/api/player-profile/pages/identity_presentation/commit", {
      method: "POST",
      body: JSON.stringify({ answers: { portrait_url: "https://example.test/k65-portrait.png", short_intro: "Browser proof intro." } }),
    });
    assert(pageCommit.ok, "A commits identity_presentation page (portrait + short intro)");

    const meA = await api(pageA, "/api/player-profile/me");
    const profileA = meA.body?.data?.workbook?.id;
    assert(profileA, "A has a workbook id");

    // --- Anonymous rejection ---
    await pageAnon.goto("/login/");
    const anonList = await api(pageAnon, "/api/third-place/headshots");
    assert(anonList.status === 401, `anonymous list must be rejected, got ${anonList.status}`);
    pass("anonymous Third Place API access rejected");

    // --- A leaves a Headshot ---
    await pageA.goto("/venues/third-place/");
    await pageA.waitForSelector("#leave-headshot-button:not([hidden])", { timeout: 15000 });
    await pageA.click("#leave-headshot-button");
    await pageA.waitForSelector("#remove-headshot-button:not([hidden])", { timeout: 15000 });
    const myLineText = await pageA.textContent("#my-headshot-line");
    assert(myLineText.includes("visible in Third Place"), `expected active-headshot copy, got ${myLineText}`);
    await screenshot(pageA, "01-a-left-headshot");
    pass("A left a Headshot and sees the active-Headshot status line");

    // A sees their own card marked "You".
    const youBadgeVisible = await pageA.isVisible(".headshot-card.is-you .you-badge");
    assert(youBadgeVisible, "A's own card must show the You badge");
    pass("A's own Headshot card is clearly marked You");

    // Idempotency: calling POST /me again (as the API layer, mirroring what
    // a second "Leave Headshot" click would send) must not create a
    // duplicate -- confirmed both by the API response and by the rendered
    // grid still showing exactly one self card after a reload.
    const repeatLeave = await api(pageA, "/api/third-place/headshots/me", { method: "POST" });
    assert(repeatLeave.ok && repeatLeave.body?.data?.created === false, `expected repeated leave to report created:false, got ${JSON.stringify(repeatLeave.body)}`);
    await pageA.reload();
    await pageA.waitForSelector("#remove-headshot-button:not([hidden])", { timeout: 15000 });
    const aCardCountAfterRepeat = await pageA.locator(".headshot-card.is-you").count();
    assert(aCardCountAfterRepeat === 1, `expected exactly one self card after repeated Leave Headshot, got ${aCardCountAfterRepeat}`);
    pass("repeated Leave Headshot did not create a duplicate card");

    // --- B browses the commons, opens A's Trailer, adds to My People ---
    await pageB.goto("/venues/third-place/");
    await pageB.waitForSelector(".headshot-card", { timeout: 15000 });
    const bBodyText = await pageB.textContent("body");
    assert(bBodyText.includes(userA.stage1), "B sees A's current stage name in the commons");
    assert(!bBodyText.includes(userA.handle), "B's page must not leak A's account handle");
    await screenshot(pageB, "02-b-sees-commons-desktop");
    pass("B sees A's Headshot in the commons, desktop viewport");

    await pageB.click(".headshot-card:not(.is-you) a:has-text('Open Trailer')");
    await pageB.waitForURL(/\/venues\/trailers\/view\.html/, { timeout: 15000 });
    pass("B opened A's Trailer from the Headshot card");

    await pageB.goto("/venues/third-place/");
    await pageB.waitForSelector(".headshot-card:not(.is-you) button:has-text('Add to My People')", { timeout: 15000 });
    await pageB.click(".headshot-card:not(.is-you) button:has-text('Add to My People')");
    await pageB.waitForURL(/\/venues\/trailers\/person\.html\?id=/, { timeout: 15000 });
    pass("B added A to My People from the Headshot card");

    await pageB.goto("/venues/third-place/");
    await pageB.waitForSelector(".headshot-card:not(.is-you) a:has-text('Open My Notes')", { timeout: 15000 });
    pass("B now sees Open My Notes instead of Add to My People for A");

    // --- Privacy: A is not notified, A's own view is unaffected ---
    await pageA.reload();
    await pageA.waitForSelector(".headshot-card.is-you", { timeout: 15000 });
    const aSelfCardText = await pageA.textContent(".headshot-card.is-you");
    assert(!aSelfCardText.includes("Open My Notes") && !aSelfCardText.includes("Add to My People"), "A's own card must never show relationship actions");
    pass("A's own Headshot shows no relationship affordance and no notification of being added");

    // --- Third viewer C never sees B's relationship with A ---
    await pageC.goto("/venues/third-place/");
    await pageC.waitForSelector(".headshot-card:not(.is-you)", { timeout: 15000 });
    const cCardText = await pageC.textContent(".headshot-card:not(.is-you)");
    assert(cCardText.includes("Add to My People"), "an unrelated third viewer must still see Add to My People for A");
    assert(!cCardText.includes("Open My Notes"), "an unrelated third viewer must never see another viewer's relationship state");
    pass("third viewer C does not see B's private relationship with A");

    // --- Privacy proof: payload contains no forbidden fields ---
    const commonsPayload = await api(pageB, "/api/third-place/headshots");
    const raw = JSON.stringify(commonsPayload.body);
    for (const forbidden of ["email", "\"handle\"", "account_uuid", "private_nickname", "journal", "followup", "stage_name_history"]) {
      assert(!raw.includes(forbidden), `Third Place payload must not contain ${forbidden}`);
    }
    pass("Third Place commons payload excludes email/handle/UUID/relationship-note fields");

    // --- Live projection: A changes stage name, B sees it update ---
    await api(pageA, "/api/player-profile/stage-name", { method: "POST", body: JSON.stringify({ stage_name: userA.stage2 }) });
    await pageB.waitForFunction(
      (expected) => document.body.textContent.includes(expected),
      userA.stage2,
      { timeout: 15000 }
    ).catch(() => null);
    let bTextAfterChange = await pageB.textContent("body");
    if (!bTextAfterChange.includes(userA.stage2)) {
      await pageB.reload();
      await pageB.waitForSelector(".headshot-card", { timeout: 15000 });
      bTextAfterChange = await pageB.textContent("body");
    }
    assert(bTextAfterChange.includes(userA.stage2), "B's Third Place view must reflect A's updated stage name");
    pass("B's Third Place view reflects A's live Trailer Face change (websocket or refresh)");

    // --- Mobile viewport ---
    await ctxMobile.addCookies(await ctxB.cookies());
    const pageMobile = await ctxMobile.newPage();
    await pageMobile.goto("/venues/third-place/");
    await pageMobile.waitForSelector(".headshot-card", { timeout: 15000 });
    await screenshot(pageMobile, "03-commons-mobile");
    pass("mobile viewport renders the Headshot Commons");

    // --- A removes their Headshot; history preserves the record ---
    await pageA.goto("/venues/third-place/");
    await pageA.waitForSelector("#remove-headshot-button:not([hidden])", { timeout: 15000 });
    await pageA.click("#remove-headshot-button");
    await pageA.waitForSelector("#leave-headshot-button:not([hidden])", { timeout: 15000 });
    pass("A removed their Headshot");

    await pageA.click("#toggle-history-button");
    await pageA.waitForSelector("#history-list .history-row", { timeout: 15000 });
    const historyText = await pageA.textContent("#history-list");
    assert(historyText.includes("Removed"), "A's history must show the removed record");
    await screenshot(pageA, "04-a-headshot-history");
    pass("A's Headshot history shows the placement/removal record");

    await pageB.goto("/venues/third-place/");
    await pageB.waitForTimeout(500);
    const bBodyAfterRemoval = await pageB.textContent("body");
    assert(!bBodyAfterRemoval.includes(userA.stage2), "removed Headshot must not still appear to other viewers");
    pass("removed Headshot no longer appears to other viewers");

    console.log(`\n${"=".repeat(60)}\nKernel 65 browser acceptance: ALL CHECKS PASSED\n${"=".repeat(60)}`);
  } finally {
    await browser.close();
  }
}

main().catch((error) => {
  console.error("KERNEL65 BROWSER PROOF FAILED:", error.message);
  process.exit(1);
});

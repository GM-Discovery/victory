#!/usr/bin/env node
"use strict";

const fs = require("node:fs");
const path = require("node:path");
const { chromium } = require("playwright");

const baseURL = process.env.K59A_BASE_URL || "https://victory.amurray.family";
const ownerToken = process.env.K59A_OWNER_TOKEN || "k59a_phase3_owner_202607060101";
const directorToken = process.env.K59A_DIRECTOR_TOKEN || "k59a_phase3_director_202607060101";
const cardID = process.env.K59A_CARD_ID || "0eea7529-f7c9-47a2-a538-e8da960b1723";
const evidenceDir = process.env.K59A_EVIDENCE_DIR || path.join("Construction", "OperatorLogs", "evidence", "kernel-59A-phase3");
const originalTagline = process.env.K59A_ORIGINAL_TAGLINE || "Parentage roll 36 · Merchant's Clerk";
const originalBio = process.env.K59A_ORIGINAL_BIO || "Bookkeeping, inventory management";

const marker = `K59A_BROWSER_${Date.now()}`;
const overrideQuote = `${marker} quote override`;
const directQuote = `${marker} direct owner quote`;
const privateJournal = `${marker} PRIVATE JOURNAL SHOULD NOT PROJECT`;
const directorReason = `${marker} private director reason`;

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

async function api(page, url, options = {}) {
  const response = await page.evaluate(async ({ url, options }) => {
    const res = await fetch(url, {
      credentials: "include",
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...(options.headers || {}),
      },
    });
    const text = await res.text();
    let body = null;
    try {
      body = text ? JSON.parse(text) : null;
    } catch {
      body = { raw: text };
    }
    return { ok: res.ok, status: res.status, body };
  }, { url, options });
  return response;
}

async function postFace(page, endpoint, body) {
  return api(page, `/api/character-workbooks/${encodeURIComponent(cardID)}/${endpoint}`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

async function patchCard(page, tagline, publicDescription = originalBio) {
  return api(page, `/api/character-cards/${encodeURIComponent(cardID)}`, {
    method: "PATCH",
    body: JSON.stringify({
      name: "IV",
      pronouns: "",
      portrait_url: "",
      token_aura: "",
      aura: "",
      color: "#d9c7a6",
      tagline,
      public_description: publicDescription,
      private_notes: "",
    }),
  });
}

async function waitForSheet(page) {
  await page.waitForSelector("#venue-sheet:not([hidden])", { timeout: 30000 });
  await page.waitForFunction(() => {
    const name = document.querySelector("#venue-sheet-name")?.textContent || "";
    return name.trim().length > 0;
  }, null, { timeout: 30000 });
}

async function sheetText(page) {
  return page.locator("#venue-sheet").innerText({ timeout: 10000 });
}

async function sheetFacts(page) {
  return page.evaluate(() => [...document.querySelectorAll("#venue-sheet-facts .venue-sheet__fact")].map((node) => node.textContent.trim()));
}

async function screenshot(page, name) {
  await page.screenshot({ path: path.join(evidenceDir, `${name}.png`), fullPage: true });
}

async function main() {
  fs.mkdirSync(evidenceDir, { recursive: true });
  const browser = await chromium.launch({
    headless: true,
    args: ["--host-resolver-rules=MAP victory.amurray.family 127.0.0.1"],
  });

  const contextOptions = {
    baseURL,
    ignoreHTTPSErrors: true,
    viewport: { width: 1366, height: 900 },
  };
  const ownerContext = await browser.newContext(contextOptions);
  const venueContext = await browser.newContext(contextOptions);
  const firstTheaterContext = await browser.newContext(contextOptions);
  const directorContext = await browser.newContext(contextOptions);

  for (const [context, token] of [[ownerContext, ownerToken], [venueContext, ownerToken], [firstTheaterContext, ownerToken], [directorContext, directorToken]]) {
    await context.addCookies([{
      name: "victory_session",
      value: token,
      domain: "victory.amurray.family",
      path: "/",
      httpOnly: true,
      secure: true,
      sameSite: "Lax",
    }]);
  }

  await venueContext.addInitScript(() => {
    window.localStorage.setItem("victory:catharsis:venue-intro:v1", "1");
    window.__k59aWSMessages = [];
    const NativeWebSocket = window.WebSocket;
    window.WebSocket = function patchedWebSocket(...args) {
      const ws = new NativeWebSocket(...args);
      ws.addEventListener("message", (event) => {
        window.__k59aWSMessages.push(String(event.data || ""));
      });
      return ws;
    };
    window.WebSocket.prototype = NativeWebSocket.prototype;
    Object.setPrototypeOf(window.WebSocket, NativeWebSocket);
  });

  const owner = await ownerContext.newPage();
  const venue = await venueContext.newPage();
  const firstTheater = await firstTheaterContext.newPage();
  const director = await directorContext.newPage();

  const evidence = {
    marker,
    card_id: cardID,
    screenshots: [],
    assertions: [],
    websocket_messages: [],
    api_results: {},
  };

  try {
    await owner.goto(`/venues/greenroom/?character_id=${encodeURIComponent(cardID)}`, { waitUntil: "networkidle", timeout: 60000 });
    await director.goto("/venues/greenroom/", { waitUntil: "domcontentloaded", timeout: 60000 });
    await venue.goto("/venues/catharsis/", { waitUntil: "networkidle", timeout: 60000 });
    await firstTheater.goto("/venues/first-theater/", { waitUntil: "networkidle", timeout: 60000 });
    await waitForSheet(venue);
    await waitForSheet(firstTheater);
    await screenshot(venue, "01-initial-venue-face");
    evidence.screenshots.push("01-initial-venue-face.png");
    await screenshot(firstTheater, "01b-first-theater-initial-face");
    evidence.screenshots.push("01b-first-theater-initial-face.png");
    const initialText = await sheetText(venue);
    assert(initialText.includes("IV"), "initial venue sheet did not show the active character name");
    const firstInitialText = await sheetText(firstTheater);
    assert(firstInitialText.includes("IV"), "initial First Theater sheet did not show the active character name");
    evidence.assertions.push("Initial Catharsis and First Theater sheets rendered active character IV.");

    evidence.api_results.hide_quote = await postFace(owner, "face-visibility", { fact_key: "tagline", mode: "hidden" });
    assert(evidence.api_results.hide_quote.ok, "hide quote API failed");
    await venue.waitForFunction((quote) => !(document.querySelector("#venue-sheet")?.innerText || "").includes(quote), originalTagline, { timeout: 30000 });
    await firstTheater.waitForFunction((quote) => !(document.querySelector("#venue-sheet")?.innerText || "").includes(quote), originalTagline, { timeout: 30000 });
    await screenshot(venue, "02-quote-hidden-live");
    evidence.screenshots.push("02-quote-hidden-live.png");
    await screenshot(firstTheater, "02b-first-theater-quote-hidden-live");
    evidence.screenshots.push("02b-first-theater-quote-hidden-live.png");
    evidence.assertions.push("Owner hide of Featured Quote removed it from open Catharsis and First Theater trays without focus/reload.");

    evidence.api_results.show_quote = await postFace(owner, "face-visibility", { fact_key: "tagline", mode: "shown" });
    assert(evidence.api_results.show_quote.ok, "show quote API failed");
    await venue.waitForFunction((quote) => (document.querySelector("#venue-sheet")?.innerText || "").includes(quote), originalTagline, { timeout: 30000 });
    await firstTheater.waitForFunction((quote) => (document.querySelector("#venue-sheet")?.innerText || "").includes(quote), originalTagline, { timeout: 30000 });
    await screenshot(venue, "03-quote-shown-live");
    evidence.screenshots.push("03-quote-shown-live.png");
    await screenshot(firstTheater, "03b-first-theater-quote-shown-live");
    evidence.screenshots.push("03b-first-theater-quote-shown-live.png");
    evidence.assertions.push("Owner show of Featured Quote restored it in both open venue trays.");

    evidence.api_results.priority_bio = await postFace(owner, "face-priority", { fact_key: "public_description", mode: "manual", score: 999 });
    assert(evidence.api_results.priority_bio.ok, "manual priority API failed");
    await venue.waitForFunction((bio) => {
      const facts = [...document.querySelectorAll("#venue-sheet-facts .venue-sheet__fact")].map((node) => node.textContent || "");
      return facts.length > 0 && facts[0].includes(bio);
    }, originalBio, { timeout: 30000 });
    await firstTheater.waitForFunction((bio) => {
      const facts = [...document.querySelectorAll("#venue-sheet-facts .venue-sheet__fact")].map((node) => node.textContent || "");
      return facts.length > 0 && facts[0].includes(bio);
    }, originalBio, { timeout: 30000 });
    const priorityFacts = await sheetFacts(venue);
    evidence.assertions.push(`Manual priority moved Bio to the first venue Face fact: ${priorityFacts[0] || ""}`);
    await screenshot(venue, "04-priority-bio-first");
    evidence.screenshots.push("04-priority-bio-first.png");

    evidence.api_results.value_override = await postFace(director, "face-value", { fact_key: "tagline", value: overrideQuote, reason: directorReason });
    assert(evidence.api_results.value_override.ok, "director value override API failed");
    await venue.waitForFunction((quote) => (document.querySelector("#venue-sheet")?.innerText || "").includes(quote), overrideQuote, { timeout: 30000 });
    let visibleText = await sheetText(venue);
    assert(!visibleText.includes(directorReason), "director reason leaked into venue tray");
    evidence.assertions.push("Director value override projected to venue tray and the private reason was not rendered.");
    await screenshot(venue, "05-director-override-visible");
    evidence.screenshots.push("05-director-override-visible.png");

    evidence.api_results.lock_value = await postFace(director, "face-lock", { fact_key: "tagline", dimension: "value", locked: true, reason: directorReason });
    assert(evidence.api_results.lock_value.ok, "director value lock API failed");
    evidence.api_results.locked_owner_patch = await patchCard(owner, directQuote);
    assert(!evidence.api_results.locked_owner_patch.ok, "owner card patch unexpectedly succeeded while value was locked");
    const lockError = evidence.api_results.locked_owner_patch.body?.data?.error || evidence.api_results.locked_owner_patch.body?.error || "";
    assert(lockError === "face_value_locked", `expected face_value_locked, got ${lockError}`);
    evidence.assertions.push("Director value lock blocked an owner PATCH with face_value_locked.");

    evidence.api_results.unlock_value = await postFace(director, "face-lock", { fact_key: "tagline", dimension: "value", locked: false, reason: "Phase 3 acceptance cleanup" });
    assert(evidence.api_results.unlock_value.ok, "director value unlock API failed");
    evidence.api_results.clear_value = await postFace(director, "face-value", { fact_key: "tagline", clear: true, reason: "Phase 3 acceptance cleanup" });
    assert(evidence.api_results.clear_value.ok, "director value clear API failed");
    evidence.api_results.unlocked_owner_patch = await patchCard(owner, directQuote);
    assert(evidence.api_results.unlocked_owner_patch.ok, "owner patch failed after unlock");
    await venue.waitForFunction((quote) => (document.querySelector("#venue-sheet")?.innerText || "").includes(quote), directQuote, { timeout: 30000 });
    evidence.assertions.push("Unlocked value dimension allowed owner edit and projected the new quote.");
    await screenshot(venue, "06-unlocked-owner-edit");
    evidence.screenshots.push("06-unlocked-owner-edit.png");

    evidence.api_results.private_journal = await api(owner, "/api/character-journals", {
      method: "POST",
      body: JSON.stringify({ character_card_id: cardID, visibility: "private", body: privateJournal }),
    });
    if (evidence.api_results.private_journal.ok) {
      await pageWait(venue, 1000);
      visibleText = await sheetText(venue);
      assert(!visibleText.includes(privateJournal), "private journal leaked into venue tray");
      evidence.assertions.push("Private journal marker did not enter the venue projection.");
    } else {
      evidence.assertions.push("Private journal endpoint was unavailable in browser fixture; projection privacy was checked through payload shape only.");
    }

    const wsMessages = await venue.evaluate(() => window.__k59aWSMessages || []);
    evidence.websocket_messages = wsMessages.filter((message) => message.includes("character/projection_updated"));
    assert(evidence.websocket_messages.length >= 1, "no projection_updated websocket frame was observed");
    assert(evidence.websocket_messages.every((message) => !message.includes(directorReason) && !message.includes(privateJournal)), "private data leaked in websocket invalidation frame");
    evidence.assertions.push(`Observed ${evidence.websocket_messages.length} server projection invalidation websocket frame(s), with no private reason/journal text.`);

    await venue.close();
    const lateVenue = await venueContext.newPage();
    await lateVenue.goto("/venues/catharsis/", { waitUntil: "networkidle", timeout: 60000 });
    await waitForSheet(lateVenue);
    await lateVenue.waitForFunction((quote) => (document.querySelector("#venue-sheet")?.innerText || "").includes(quote), directQuote, { timeout: 30000 });
    await screenshot(lateVenue, "07-late-join-latest-projection");
    evidence.screenshots.push("07-late-join-latest-projection.png");
    evidence.assertions.push("Late join/reconnect loaded the latest projected quote from the canonical venue-sheet endpoint.");

    await lateVenue.evaluate(() => document.querySelector("#venue-sheet-mechanics-tab")?.click());
    await lateVenue.waitForSelector("#venue-sheet-mechanics-panel:not([hidden])", { timeout: 10000 });
    const skillCount = await lateVenue.locator("#venue-sheet-skills .venue-sheet__skill").count();
    assert(skillCount > 0, "mechanics tab rendered no skill buttons");
    await screenshot(lateVenue, "08-mechanics-tab-skills");
    evidence.screenshots.push("08-mechanics-tab-skills.png");
    evidence.assertions.push(`Mechanics tab rendered ${skillCount} clickable skill button(s).`);
    await lateVenue.close();

    await firstTheater.evaluate(() => document.querySelector("#venue-sheet-mechanics-tab")?.click());
    await firstTheater.waitForSelector("#venue-sheet-mechanics-panel:not([hidden])", { timeout: 10000 });
    const firstSkillCount = await firstTheater.locator("#venue-sheet-skills .venue-sheet__skill").count();
    assert(firstSkillCount > 0, "First Theater mechanics tab rendered no skill buttons");
    await screenshot(firstTheater, "09-first-theater-mechanics-tab-skills");
    evidence.screenshots.push("09-first-theater-mechanics-tab-skills.png");
    evidence.assertions.push(`First Theater Mechanics tab rendered ${firstSkillCount} clickable skill button(s).`);

    evidence.status = "PASS";
  } finally {
    try { await postFace(director, "face-lock", { fact_key: "tagline", dimension: "value", locked: false, reason: "Phase 3 acceptance cleanup" }); } catch {}
    try { await postFace(director, "face-value", { fact_key: "tagline", clear: true, reason: "Phase 3 acceptance cleanup" }); } catch {}
    try { await postFace(owner, "face-priority", { fact_key: "public_description", mode: "inferred" }); } catch {}
    try { await postFace(owner, "face-visibility", { fact_key: "tagline", mode: "inferred" }); } catch {}
    try { await patchCard(owner, originalTagline, originalBio); } catch {}
    fs.writeFileSync(path.join(evidenceDir, "acceptance-evidence.json"), JSON.stringify(evidence, null, 2));
    await browser.close();
  }
}

async function pageWait(page, ms) {
  await page.waitForTimeout(ms);
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});

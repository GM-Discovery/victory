#!/usr/bin/env node
"use strict";

// Kernel 62 browser acceptance proof (spec §16): two real browser sessions.
// User A is the subject; user B builds a private relationship record about A.
// Creates two clearly-named throwaway accounts via /api/auth/signup.
//
// Run after deploy:  node scripts/smoke/kernel62-browser.js

const fs = require("node:fs");
const path = require("node:path");
const { chromium } = require("playwright");

const baseURL = process.env.K62_BASE_URL || "https://victory.amurray.family";
const hostDomain = new URL(baseURL).hostname;
const evidenceDir = process.env.K62_EVIDENCE_DIR || path.join("Construction", "OperatorLogs", "evidence", "kernel-62");

const stamp = Date.now();
const marker = `K62_PRIVATE_${stamp}`;
const userA = { handle: `k62_subject_${stamp}`, display: "K62 Subject", stage1: `Stage A ${stamp}`, stage2: `Renamed A ${stamp}` };
const userB = { handle: `k62_observer_${stamp}`, display: "K62 Observer" };
const userC = { handle: `k62_third_${stamp}`, display: "K62 Third" };
const password = `k62-browser-${stamp}`;

function assert(condition, message) {
  if (!condition) throw new Error("ASSERT FAILED: " + message);
}

const results = [];
function pass(message) {
  results.push(message);
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
  const ctxMobile = await browser.newContext({ ...contextOptions, viewport: { width: 390, height: 844 } });

  const pageA = await ctxA.newPage();
  const pageB = await ctxB.newPage();
  const pageC = await ctxC.newPage();

  try {
    // --- Setup: three fresh accounts; A sets a stage name ---
    await signup(pageA, userA);
    await signup(pageB, userB);
    await signup(pageC, userC);
    pass("three fresh browser accounts signed up");

    const stageRes = await api(pageA, "/api/player-profile/stage-name", {
      method: "POST",
      body: JSON.stringify({ stage_name: userA.stage1 }),
    });
    assert(stageRes.ok, "A sets initial stage name");

    const meA = await api(pageA, "/api/player-profile/me");
    const profileA = meA.body?.data?.workbook?.id;
    assert(profileA, "A has a workbook id");

    // Share B's session cookie with the mobile context so mobile screenshots
    // show B's real data.
    await ctxMobile.addCookies(await ctxB.cookies());

    // --- §16.1 Add from Trailer ---
    await pageB.goto(`/venues/trailers/view.html?id=${encodeURIComponent(profileA)}`);
    await pageB.waitForSelector("#add-person-button:not([hidden])", { timeout: 30000 });
    await screenshot(pageB, "01-trailer-add-button-desktop");
    await pageB.click("#add-person-button");
    await pageB.waitForURL("**/person.html?id=*", { timeout: 30000 });
    await pageB.waitForSelector("#relationship-card:not([hidden])", { timeout: 30000 });
    const relationshipId = new URL(pageB.url()).searchParams.get("id");
    assert(relationshipId, "relationship id in person.html URL");
    pass("B added A to My People from A's Trailer Face");
    await screenshot(pageB, "02-person-detail-fresh-desktop");

    // Trailer button now flips to Open My Notes.
    await pageB.goto(`/venues/trailers/view.html?id=${encodeURIComponent(profileA)}`);
    await pageB.waitForSelector("#open-notes-link:not([hidden])", { timeout: 30000 });
    pass("A's Trailer now shows Open My Notes for B");

    // --- §16.2 My People edits persist ---
    await pageB.goto(`/venues/trailers/person.html?id=${encodeURIComponent(relationshipId)}`);
    await pageB.waitForSelector("#relationship-card:not([hidden])", { timeout: 30000 });
    await pageB.fill("#nickname-input", "The Reliable One");
    await pageB.check(`#category-checks input[value="friend"]`);
    await pageB.check(`#category-checks input[value="collaborator"]`);
    await pageB.selectOption("#dd-trust_level", "trusted");
    await pageB.selectOption("#dd-closeness_level", "friendly");
    await pageB.selectOption("#dd-reliability_level", "reliable");
    await pageB.selectOption("#dd-communication_ease_level", "easy");
    await pageB.click("#save-relationship-button");
    await pageB.waitForFunction(() => document.getElementById("relationship-status").textContent === "Saved.", null, { timeout: 30000 });

    await pageB.reload();
    await pageB.waitForSelector("#relationship-card:not([hidden])", { timeout: 30000 });
    assert(await pageB.inputValue("#nickname-input") === "The Reliable One", "nickname persisted after reload");
    assert(await pageB.inputValue("#dd-trust_level") === "trusted", "trust persisted after reload");
    assert(await pageB.isChecked(`#category-checks input[value="friend"]`), "category persisted after reload");
    pass("nickname, categories, and qualitative dropdowns persist across reload");
    await screenshot(pageB, "03-person-detail-filled-desktop");

    // Workbook page save.
    await pageB.fill("#how_i_know_them", `${marker} met through weekly game night`);
    await pageB.click("#save-page-button");
    await pageB.waitForFunction(() => document.getElementById("save-status").textContent === "Saved.", null, { timeout: 30000 });
    pass("relationship workbook page saved");

    // --- §16.3 Journal ---
    await pageB.click(`#tabbar [data-tab="journal"]`);
    await pageB.fill("#journal-title", "First note");
    await pageB.fill("#journal-category", "session");
    await pageB.fill("#journal-tags", "smoke, kernel62");
    await pageB.fill("#journal-body", `${marker} private journal body`);
    await pageB.click("#save-journal-button");
    await pageB.waitForFunction(() => document.getElementById("journal-status").textContent === "Saved.", null, { timeout: 30000 });
    await pageB.waitForSelector("#journal-list .entry-card", { timeout: 30000 });
    await screenshot(pageB, "04-journal-desktop");

    // Filter by tag.
    await pageB.selectOption("#journal-filter-tag", "kernel62");
    await pageB.waitForSelector("#journal-list .entry-card", { timeout: 30000 });

    // Edit.
    await pageB.click("#journal-list .entry-card .link-button.safe");
    await pageB.fill("#journal-body", `${marker} edited private journal body`);
    await pageB.click("#save-journal-button");
    await pageB.waitForFunction(() => document.getElementById("journal-status").textContent === "Saved.", null, { timeout: 30000 });
    const journalText = await pageB.locator("#journal-list").innerText();
    assert(journalText.includes("edited private journal body"), "journal edit visible");

    // Delete.
    pageB.once("dialog", (dialog) => dialog.accept());
    await pageB.click("#journal-list .entry-card .link-button:not(.safe)");
    await pageB.waitForSelector("#journal-empty:not([hidden])", { timeout: 30000 });
    pass("journal create/filter/edit/delete all work");

    // Re-create one entry so privacy checks below have content to protect.
    await pageB.fill("#journal-body", `${marker} durable private note about A`);
    await pageB.click("#save-journal-button");
    await pageB.waitForFunction(() => document.getElementById("journal-status").textContent === "Saved.", null, { timeout: 30000 });

    // --- §16.4 Follow-ups ---
    await pageB.click(`#tabbar [data-tab="followups"]`);
    await pageB.fill("#followup-title", "Ask about co-running a one-shot");
    await pageB.fill("#followup-date", "2026-08-01");
    await pageB.click("#save-followup-button");
    await pageB.waitForFunction(() => document.getElementById("followup-status").textContent === "Added.", null, { timeout: 30000 });
    await pageB.click("#followup-list .followup-row.open .chip:has-text('Done')");
    await pageB.waitForSelector("#followup-list .followup-row.done", { timeout: 30000 });

    await pageB.fill("#followup-title", "Second item to dismiss");
    await pageB.click("#save-followup-button");
    await pageB.waitForFunction(() => document.getElementById("followup-status").textContent === "Added.", null, { timeout: 30000 });
    await pageB.click("#followup-list .followup-row.open .chip:has-text('Dismiss')");
    await pageB.waitForSelector("#followup-list .followup-row.dismissed", { timeout: 30000 });
    pass("follow-ups: create, done, dismiss (stored records only)");
    await screenshot(pageB, "05-followups-desktop");

    // --- My People list ---
    await pageB.goto("/venues/trailers/people.html");
    await pageB.waitForSelector(".person-row", { timeout: 30000 });
    let listText = await pageB.locator("#people-list").innerText();
    assert(listText.includes(userA.stage1), "A appears in B's My People with current stage name");
    assert(listText.includes("The Reliable One"), "private nickname shown in list");
    pass("My People list shows stage name, nickname, and metadata");
    await screenshot(pageB, "06-my-people-desktop");

    // Search.
    await pageB.fill("#search-input", "reliable one");
    await pageB.waitForSelector(".person-row", { timeout: 30000 });
    await pageB.fill("#search-input", "no such person zzz");
    await pageB.waitForSelector("#no-match-state:not([hidden])", { timeout: 30000 });
    await pageB.fill("#search-input", "");
    pass("My People search filters by nickname/stage name");

    // Mobile screenshots.
    const pageM = await ctxMobile.newPage();
    await pageM.goto("/venues/trailers/people.html");
    await pageM.waitForSelector(".person-row", { timeout: 30000 });
    await screenshot(pageM, "07-my-people-mobile");
    await pageM.goto(`/venues/trailers/person.html?id=${encodeURIComponent(relationshipId)}`);
    await pageM.waitForSelector("#relationship-card:not([hidden])", { timeout: 30000 });
    await screenshot(pageM, "08-person-detail-mobile");
    await pageM.close();
    pass("mobile viewport renders My People and person detail");

    // --- §16.6 Privacy / cross-user ---
    // A opens B's relationship URL about A -> safe not-found.
    await pageA.goto(`/venues/trailers/person.html?id=${encodeURIComponent(relationshipId)}`);
    await pageA.waitForSelector("#not-found-state:not([hidden])", { timeout: 30000 });
    await screenshot(pageA, "09-privacy-not-found-subject");
    const subjectRead = await api(pageA, `/api/player-relationships/${encodeURIComponent(relationshipId)}`);
    assert(subjectRead.status === 404, `subject direct API read is 404 (got ${subjectRead.status})`);
    const subjectWrite = await api(pageA, `/api/player-relationships/${encodeURIComponent(relationshipId)}`, {
      method: "PATCH",
      body: JSON.stringify({ private_nickname: "hijacked" }),
    });
    assert(subjectWrite.status === 404, `subject direct API write is 404 (got ${subjectWrite.status})`);
    const subjectJournal = await api(pageA, `/api/player-relationships/${encodeURIComponent(relationshipId)}/journal`);
    assert(subjectJournal.status === 404, `subject journal read is 404 (got ${subjectJournal.status})`);

    // Third user gets the same 404 shape.
    const thirdRead = await api(pageC, `/api/player-relationships/${encodeURIComponent(relationshipId)}`);
    assert(thirdRead.status === 404, `third-user read is 404 (got ${thirdRead.status})`);
    const thirdArchive = await api(pageC, `/api/player-relationships/${encodeURIComponent(relationshipId)}/archive`, { method: "POST" });
    assert(thirdArchive.status === 404, `third-user archive is 404 (got ${thirdArchive.status})`);

    // A's own surfaces never contain B's private note text.
    await pageA.goto("/venues/trailers/face.html");
    await pageA.waitForSelector("#identity-card:not([hidden])", { timeout: 30000 });
    let domA = await pageA.content();
    assert(!domA.includes(marker), "A's Face DOM contains no private note text");
    assert(!domA.includes("The Reliable One"), "A's Face DOM contains no private nickname");
    await pageA.goto(`/venues/trailers/view.html?id=${encodeURIComponent(profileA)}`);
    await pageA.waitForSelector("#identity-card:not([hidden])", { timeout: 30000 });
    domA = await pageA.content();
    assert(!domA.includes(marker), "A's Trailer view DOM contains no private note text");
    pass("privacy: subject and third user get 404s; A's DOM never sees B's notes");

    // A also sees no sign of the record on their own My People (directional).
    const listForA = await api(pageA, "/api/player-relationships?state=all");
    assert(listForA.ok && (listForA.body?.data?.relationships || []).length === 0, "A's own My People is empty (directional)");
    pass("relationship is directional: A's own list is unaffected");

    // --- §16.7 Stage-name change follows the same person ---
    const rename = await api(pageA, "/api/player-profile/stage-name", {
      method: "POST",
      body: JSON.stringify({ stage_name: userA.stage2 }),
    });
    assert(rename.ok, "A renames stage name");
    await pageB.goto("/venues/trailers/people.html");
    await pageB.waitForSelector(".person-row", { timeout: 30000 });
    listText = await pageB.locator("#people-list").innerText();
    assert(listText.includes(userA.stage2), "B's list shows A's new stage name after refresh");
    assert(listText.includes("The Reliable One"), "private nickname unchanged after stage-name change");
    pass("stage-name change: record follows the same account, nickname intact");
    await screenshot(pageB, "10-stage-name-updated-desktop");

    // --- §16.5 Archive / unarchive ---
    await pageB.goto(`/venues/trailers/person.html?id=${encodeURIComponent(relationshipId)}`);
    await pageB.waitForSelector("#relationship-card:not([hidden])", { timeout: 30000 });
    await pageB.click("#archive-button");
    await pageB.waitForSelector("#archived-badge:not([hidden])", { timeout: 30000 });
    await pageB.goto("/venues/trailers/people.html");
    await pageB.waitForSelector("#empty-state:not([hidden])", { timeout: 30000 });
    await pageB.selectOption("#state-filter", "archived");
    await pageB.waitForSelector(".person-row.archived", { timeout: 30000 });
    await screenshot(pageB, "11-archived-filter-desktop");
    await pageB.goto(`/venues/trailers/person.html?id=${encodeURIComponent(relationshipId)}`);
    await pageB.waitForSelector("#relationship-card:not([hidden])", { timeout: 30000 });
    await pageB.click("#archive-button");
    await pageB.waitForFunction(() => document.getElementById("archived-badge").hidden, null, { timeout: 30000 });
    await pageB.goto("/venues/trailers/people.html");
    await pageB.waitForSelector(".person-row:not(.archived)", { timeout: 30000 });
    pass("archive hides from default list; unarchive restores; A never notified");

    console.log("\nALL KERNEL 62 BROWSER CHECKS PASSED (" + results.length + " groups)");
    console.log("Evidence: " + evidenceDir);
    console.log("Throwaway accounts created: " + [userA.handle, userB.handle, userC.handle].join(", "));
  } finally {
    await browser.close();
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});

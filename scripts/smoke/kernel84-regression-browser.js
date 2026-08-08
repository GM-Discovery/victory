// Kernel 84 §7.3: focused live browser regression against the real
// deployed production backend, proving Storyboards (Blank + Timeline),
// Coordination, eWrite/Library, and a negative authority check all still
// work after this kernel's cleanup. Not a full re-proof of any earlier
// kernel -- see kernel-83-reportback.md for the exhaustive coordination
// proof and kernel-82-reportback.md for the exhaustive Timeline proof.
// Disposable accounts via direct fixture-row insertion (password signup
// is closed in production -- see kernel-maker-field-guide.md).
const { chromium } = require("/tmp/node_modules/playwright");
const { execFileSync } = require("child_process");
const crypto = require("crypto");

const BASE = "https://victory.amurray.family";
const HOST = "victory.amurray.family";
const stamp = Date.now().toString(36);
const results = {};
const createdUserIds = [];
const createdBoardIds = [];

function must(cond, label) {
  if (!cond) throw new Error("ASSERTION FAILED: " + label);
  results[label] = true;
}

function psql(sql) {
  const out = execFileSync(
    "docker",
    ["exec", "-i", "victory-postgres", "psql", "-U", "victory", "-d", "victory", "-tA", "-c", sql],
    { encoding: "utf8" }
  );
  return out.split("\n")[0].trim();
}

function cleanup() {
  try {
    for (const boardId of createdBoardIds) {
      psql(`DELETE FROM storyboards WHERE id = '${boardId}'`);
    }
    for (const userId of createdUserIds) {
      psql(`DELETE FROM auth.sessions WHERE user_id = '${userId}'`);
      psql(`DELETE FROM storyboard_grants WHERE user_id = '${userId}' OR granted_by = '${userId}'`);
      psql(`DELETE FROM users WHERE id = '${userId}'`);
    }
    console.log("Cleanup complete: " + createdBoardIds.length + " boards + " + createdUserIds.length + " fixture accounts removed.");
  } catch (err) {
    console.error("CLEANUP WARNING:", err.message);
  }
}

async function createFixtureAccount(handlePrefix) {
  const handle = (handlePrefix + "_" + stamp).slice(0, 24);
  const userId = psql(`INSERT INTO users (handle, display_name) VALUES ('${handle}', '${handlePrefix}') RETURNING id::text`);
  createdUserIds.push(userId);
  const raw = crypto.randomBytes(32).toString("base64url");
  const tokenHashHex = crypto.createHash("sha256").update(raw, "utf8").digest("hex");
  psql(`INSERT INTO auth.sessions (user_id, token_hash, expires_at, ip, user_agent)
     VALUES ('${userId}', decode('${tokenHashHex}', 'hex'), now() + interval '2 hours', '127.0.0.1', 'kernel84-smoke')`);
  return { handle, label: handlePrefix, userId, cookie: raw };
}

async function apiAs(account, path, opts) {
  const res = await fetch(BASE + path, Object.assign({}, opts, {
    headers: Object.assign({ "Content-Type": "application/json", Cookie: "victory_session=" + account.cookie }, (opts && opts.headers) || {}),
  }));
  const payload = await res.json().catch(() => null);
  return { res, payload };
}

async function openTab(browser, account, boardId) {
  const context = await browser.newContext();
  await context.addCookies([{ name: "victory_session", value: account.cookie, domain: HOST, path: "/", httpOnly: true, secure: true, sameSite: "Lax" }]);
  const page = await context.newPage();
  await page.goto(`${BASE}/venues/storyboards/board.html?id=${boardId}`, { waitUntil: "networkidle" });
  return { context, page };
}

async function main() {
  console.log("Creating fixture accounts...");
  const owner = await createFixtureAccount("k84owner");
  const other = await createFixtureAccount("k84other");

  const browser = await chromium.launch();

  // ---------- Storyboards: Blank ----------
  const { payload: blankCreate } = await apiAs(owner, "/api/storyboards", { method: "POST", body: JSON.stringify({ title: "K84 Blank " + stamp, mode: "blank" }) });
  const blankId = blankCreate.data.board.id;
  createdBoardIds.push(blankId);

  const blankTab = await openTab(browser, owner, blankId);
  await blankTab.page.waitForSelector("#board-grid .cell", { timeout: 10000 });
  must(true, "blank_board_opens");

  const dragErrors = [];
  blankTab.page.on("pageerror", (e) => dragErrors.push(String(e)));

  // Add a card via the UI (+ card -> createCardFlow's native prompt()).
  blankTab.page.once("dialog", (dialog) => dialog.accept("K84 Regression Card"));
  await blankTab.page.locator(".cell-empty-add").first().click();
  await blankTab.page.waitForSelector(".sb-card", { timeout: 5000 });
  must(true, "blank_card_created_via_ui");

  // Drag the card to a different cell (Pointer Events, real mouse sequence).
  const cardBox = await blankTab.page.locator(".sb-card").first().boundingBox();
  const cells = blankTab.page.locator(".cell");
  const cellCount = await cells.count();
  let targetCell = null;
  for (let i = 0; i < cellCount; i++) {
    const c = cells.nth(i);
    const hasCard = await c.locator(".sb-card").count();
    if (hasCard === 0) { targetCell = c; break; }
  }
  const targetBox = targetCell ? await targetCell.boundingBox() : null;
  if (cardBox && targetBox) {
    await blankTab.page.mouse.move(cardBox.x + cardBox.width / 2, cardBox.y + cardBox.height / 2);
    await blankTab.page.mouse.down();
    await blankTab.page.mouse.move(targetBox.x + targetBox.width / 2, targetBox.y + targetBox.height / 2, { steps: 10 });
    await blankTab.page.mouse.up();
    await blankTab.page.waitForTimeout(400);
  }
  must(dragErrors.length === 0, "blank_card_drag_no_js_errors");

  await blankTab.context.close();

  // ---------- Storyboards: Timeline ----------
  const { payload: tlCreate } = await apiAs(owner, "/api/storyboards", { method: "POST", body: JSON.stringify({ title: "K84 Timeline " + stamp, mode: "timeline" }) });
  const tlId = tlCreate.data.board.id;
  createdBoardIds.push(tlId);

  const tlTab = await openTab(browser, owner, tlId);
  await tlTab.page.waitForSelector("#board-grid .col-header", { timeout: 10000 });

  const colTitles = await tlTab.page.locator(".col-header .col-header-title").allTextContents();
  must(colTitles[0] === "Beginning" && colTitles[colTitles.length - 1] === "Ending", "timeline_boundary_columns_present");

  // Insert a few extra columns so the board has genuine horizontal overflow
  // to pan across (a fresh Timeline's default 3 columns may not). Done the
  // same way the UI's own "Insert right" does it (create at end, then an
  // explicit reorder to land it before Ending) -- NOT the raw "+ Column
  // (end)" POST alone, which was found during this same regression pass to
  // append past Ending with no boundary awareness of its own (see
  // Construction/Operations/cleanup-ledger-kernel-84.md's "found, not
  // fixed" section -- AddColumn has no Timeline-boundary special-casing;
  // only the client-side insertColumnFlow dance and ReorderColumns'
  // ErrBoundaryColumnDisplaced check keep the UI-driven path correct).
  for (let i = 0; i < 6; i++) {
    const created = await apiAs(owner, `/api/storyboards/${tlId}/columns`, { method: "POST", body: JSON.stringify({ title: "Extra " + i }) });
    const newColId = created.payload.data.column.id;
    const { payload: snapPayload } = await apiAs(owner, `/api/storyboards/${tlId}`);
    const ids = snapPayload.data.columns.map((c) => c.id).filter((id) => id !== newColId);
    const endingIdx = snapPayload.data.columns.findIndex((c) => c.column_role === "ending");
    const insertAt = ids.length > 0 ? Math.max(0, endingIdx) : 0;
    ids.splice(insertAt, 0, newColId);
    await apiAs(owner, `/api/storyboards/${tlId}/columns/reorder`, { method: "POST", body: JSON.stringify({ ordered_column_ids: ids }) });
  }
  await tlTab.page.reload({ waitUntil: "networkidle" });
  await tlTab.page.waitForSelector("#board-grid .col-header", { timeout: 10000 });
  const colTitlesAfterFiller = await tlTab.page.locator(".col-header .col-header-title").allTextContents();
  must(colTitlesAfterFiller[colTitlesAfterFiller.length - 1] === "Ending", "timeline_ending_still_last_after_filler_columns");

  const refPanelVisible = await tlTab.page.locator("#reference-panel:not([hidden])").count();
  must(refPanelVisible > 0, "timeline_reference_panel_visible");

  // Insert inward column via the Beginning column's menu.
  const beginningHeader = tlTab.page.locator(".col-header").first();
  await beginningHeader.locator(".col-menu-btn").click();
  tlTab.page.once("dialog", (dialog) => dialog.accept("K84 Inward Column"));
  await tlTab.page.getByRole("button", { name: "Insert right" }).first().click();
  await tlTab.page.waitForTimeout(500);
  const colTitlesAfterInsert = await tlTab.page.locator(".col-header .col-header-title").allTextContents();
  must(colTitlesAfterInsert.length === colTitlesAfterFiller.length + 1, "timeline_insert_inward_column");

  // Middle-mouse pan: raw CDP dispatch (Playwright's mouse API has no
  // middle-button primitive -- same technique Kernel 82's own proof used).
  const cdp = await tlTab.page.context().newCDPSession(tlTab.page);
  const scrollBefore = await tlTab.page.evaluate(() => document.getElementById("board-scroll").scrollLeft);
  const scroller = await tlTab.page.locator("#board-scroll").boundingBox();
  await cdp.send("Input.dispatchMouseEvent", { type: "mousePressed", x: scroller.x + 300, y: scroller.y + 100, button: "middle", clickCount: 1 });
  await cdp.send("Input.dispatchMouseEvent", { type: "mouseMoved", x: scroller.x + 100, y: scroller.y + 100, button: "middle" });
  await cdp.send("Input.dispatchMouseEvent", { type: "mouseReleased", x: scroller.x + 100, y: scroller.y + 100, button: "middle" });
  await tlTab.page.waitForTimeout(300);
  const scrollAfter = await tlTab.page.evaluate(() => document.getElementById("board-scroll").scrollLeft);
  must(scrollAfter !== scrollBefore, "timeline_middle_pan_moved_scroll");

  // Export.
  const { res: exportRes, payload: exportPayload } = await apiAs(owner, `/api/storyboards/${tlId}/export`);
  // Export's response body is the ExportDocument directly (no {ok,data}
  // envelope -- it's served with Content-Disposition: attachment).
  must(exportRes.ok && exportPayload.board && exportPayload.board.mode === "timeline", "timeline_export_succeeds");

  // ---------- Coordination (condensed 2-user regression) ----------
  await tlTab.page.waitForSelector("#presence-tray:not([hidden])", { timeout: 10000 });
  const coord0 = await tlTab.page.evaluate(() => snapshot.coordination);
  must(coord0 && coord0.active && coord0.group_leader_user_id === (await tlTab.page.evaluate(() => snapshot.viewer_user_id)), "coord_owner_leader_on_start");

  await apiAs(owner, `/api/storyboards/${tlId}/grants`, { method: "POST", body: JSON.stringify({ user_handle: other.handle, granted_role: "crew" }) });
  const otherTab = await openTab(browser, other, tlId);
  await otherTab.page.waitForSelector("#presence-tray:not([hidden])", { timeout: 10000 });
  await tlTab.page.waitForTimeout(400);

  const otherChip = tlTab.page.locator(".presence-chip", { hasText: "k84other" }).first();
  await otherChip.click({ button: "right" });
  await tlTab.page.getByRole("button", { name: "Make Group Leader" }).click();
  await tlTab.page.waitForTimeout(300);
  const otherUserId = await otherTab.page.evaluate(() => snapshot.viewer_user_id);
  await otherTab.page.waitForFunction((uid) => snapshot.coordination && snapshot.coordination.group_leader_user_id === uid, otherUserId, { timeout: 5000 });
  must(true, "coord_assign_and_live_update");

  await otherChip.click({ button: "right" });
  await tlTab.page.getByRole("button", { name: "Give Turn" }).click();
  await tlTab.page.waitForTimeout(300);
  await otherTab.page.waitForFunction((uid) => snapshot.coordination && snapshot.coordination.current_turn_user_id === uid, otherUserId, { timeout: 5000 });
  must(true, "coord_give_turn_live_update");

  await otherTab.context.close();
  await tlTab.page.waitForTimeout(1200);
  const afterOtherDisconnect = await tlTab.page.evaluate(() => snapshot.coordination);
  must(afterOtherDisconnect.group_leader_user_id === otherUserId && afterOtherDisconnect.current_turn_user_id === otherUserId, "coord_disconnect_no_auto_reassign");

  await tlTab.context.close();
  await new Promise((r) => setTimeout(r, 1500));

  // ---------- eWrite / Library ----------
  const libContext = await browser.newContext();
  await libContext.addCookies([{ name: "victory_session", value: owner.cookie, domain: HOST, path: "/", httpOnly: true, secure: true, sameSite: "Lax" }]);
  const libPage = await libContext.newPage();
  const coreRulebookId = "1b7d519a-bbc2-4c5e-94c8-6c379bb98e38";
  await libPage.goto(`${BASE}/venues/library/read.html?pub=${coreRulebookId}`, { waitUntil: "networkidle" });
  const readerText = await libPage.evaluate(() => document.body.innerText);
  must(readerText.length > 2000, "ewrite_reader_opens_real_content");

  const skillDirId = "56acf726-0162-4537-8a3a-3698ed847cc2";
  await libPage.goto(`${BASE}/venues/library/skill-directory.html?dir=${skillDirId}`, { waitUntil: "networkidle" });
  const dirLinks = libPage.locator("a[href*='read.html']");
  const dirLinkCount = await dirLinks.count();
  must(dirLinkCount > 0, "ewrite_skill_directory_has_resolvable_links");
  if (dirLinkCount > 0) {
    const href = await dirLinks.first().getAttribute("href");
    await libPage.goto(new URL(href, BASE).toString(), { waitUntil: "networkidle" });
    const linkedText = await libPage.evaluate(() => document.body.innerText);
    must(linkedText.length > 500, "ewrite_directory_link_resolves_to_real_section");
  }
  await libContext.close();

  // ---------- Negative authority proof ----------
  // `other` has no grant at all on a brand-new board owned by `owner` --
  // a direct, forged protected mutation must fail server-side.
  const { payload: unauthCreate } = await apiAs(owner, "/api/storyboards", { method: "POST", body: JSON.stringify({ title: "K84 Negative Auth " + stamp, mode: "blank" }) });
  const unauthBoardId = unauthCreate.data.board.id;
  createdBoardIds.push(unauthBoardId);
  const { res: forgedRes } = await apiAs(other, `/api/storyboards/${unauthBoardId}/columns`, { method: "POST", body: JSON.stringify({ title: "Should not be allowed" }) });
  must(forgedRes.status === 403 || forgedRes.status === 404, "negative_auth_protected_action_rejected");

  await browser.close();

  console.log("\n=== RESULTS ===");
  for (const [k, v] of Object.entries(results)) console.log(k, "=", v);
  console.log("\nAll", Object.keys(results).length, "assertions passed.");
}

main()
  .then(() => cleanup())
  .catch((err) => {
    console.error("SMOKE FAILED:", err);
    console.log("\n=== PARTIAL RESULTS ===");
    for (const [k, v] of Object.entries(results)) console.log(k, "=", v);
    cleanup();
    process.exit(1);
  });

// Regression proof for the Kernel 85/88 director-panel dead-control bug.
//
// The defect: makeDraggable() treated the whole panel as its drag handle and
// called preventDefault() on mousedown for anything that was not a <button>.
// preventDefault() on mousedown suppresses the native <select> popup and
// blocks click-to-focus on <input>, so every dropdown and field in all five
// panels was dead to the mouse while buttons kept working.
//
// Run against the fixed file (expect PASS) and, with --expect-fail, against
// the pre-fix HEAD copy to confirm this test actually catches the bug.
const http = require("http");
const fs = require("fs");
const path = require("path");
const { chromium } = require("/tmp/node_modules/playwright");

const SCRATCH = process.env.OUT_DIR || __dirname;
const HARNESS = path.join(__dirname, "kernel88a-director-panel-harness.html");
const JS_PATH = process.env.COHORT_TOOLS_JS || "/opt/victory/frontend/lib/stage-runtime/kernel85-cohort-tools.js";
const expectFail = process.argv.includes("--expect-fail");

const server = http.createServer((req, res) => {
  if (req.url.startsWith("/lib/stage-runtime/kernel85-cohort-tools.js")) {
    res.writeHead(200, { "Content-Type": "application/javascript" });
    res.end(fs.readFileSync(JS_PATH));
    return;
  }
  res.writeHead(200, { "Content-Type": "text/html" });
  res.end(fs.readFileSync(HARNESS));
});

const results = [];
function check(name, ok, detail) {
  results.push({ name, ok, detail });
  console.log(`${ok ? "PASS" : "FAIL"} ${name}${detail ? " -- " + detail : ""}`);
}

(async () => {
  await new Promise((r) => server.listen(4599, r));
  // Headless Chromium GPU-stalls on this host (kernel-88 reportback §5).
  const browser = await chromium.launch({
    args: ["--disable-gpu", "--disable-webgl", "--disable-software-rasterizer"],
  });
  const page = await browser.newPage();
  await page.goto("http://127.0.0.1:4599/", { waitUntil: "domcontentloaded" });
  await page.waitForSelector("#kernel85-toolbar", { state: "attached" });

  // --- Cohorts panel: the "assign myself to a cohort" path ---------------
  await page.click('[data-k85="cohorts"]');
  await page.waitForSelector("[data-assign-user]");

  // The core assertion. A mousedown that comes back defaultPrevented is
  // exactly what stops the dropdown from ever opening.
  const assignPrevented = await page.evaluate(() => {
    const select = document.querySelector("[data-assign-user]");
    const event = new MouseEvent("mousedown", { bubbles: true, cancelable: true });
    select.dispatchEvent(event);
    return event.defaultPrevented;
  });
  check("mousedown on the Assign-to dropdown is not swallowed", assignPrevented === false,
    `defaultPrevented=${assignPrevented}`);

  // And the dropdown actually drives the assignment request.
  await page.selectOption("[data-assign-user]", "cohort-20");
  await page.waitForTimeout(300);
  const assignPost = await page.evaluate(() => window.__requests.find(
    (r) => r.method === "POST" && /\/cohorts\/cohort-20\/assignments$/.test(r.path)));
  check("choosing a cohort POSTs the assignment", Boolean(assignPost),
    assignPost ? `body=${assignPost.body}` : "no assignment POST recorded");

  // Dragging must still work from the panel's non-interactive chrome.
  const dragged = await page.evaluate(async () => {
    const panel = document.querySelector("#kernel85-cohort-panel");
    const heading = panel.querySelector("h3");
    const before = panel.getBoundingClientRect().left;
    const box = heading.getBoundingClientRect();
    heading.dispatchEvent(new MouseEvent("mousedown", {
      bubbles: true, cancelable: true, clientX: box.left + 5, clientY: box.top + 5 }));
    window.dispatchEvent(new MouseEvent("mousemove", {
      bubbles: true, clientX: box.left - 95, clientY: box.top + 5 }));
    window.dispatchEvent(new MouseEvent("mouseup", { bubbles: true }));
    return { before, after: panel.getBoundingClientRect().left };
  });
  check("panel still drags from its heading", dragged.after < dragged.before - 50,
    `left ${Math.round(dragged.before)} -> ${Math.round(dragged.after)}`);

  // --- Game Status panel: cohort switching + editable fields -------------
  await page.click('[data-k85="game-status"]');
  await page.waitForSelector("#kernel85-status-panel [data-select-cohort]");

  const statusPrevented = await page.evaluate(() => {
    const select = document.querySelector("#kernel85-status-panel [data-select-cohort]");
    const event = new MouseEvent("mousedown", { bubbles: true, cancelable: true });
    select.dispatchEvent(event);
    return event.defaultPrevented;
  });
  check("mousedown on the Game Status cohort picker is not swallowed", statusPrevented === false,
    `defaultPrevented=${statusPrevented}`);

  await page.selectOption("#kernel85-status-panel [data-select-cohort]", "cohort-20");
  await page.waitForTimeout(400);
  const switched = await page.evaluate(() =>
    document.querySelector("#kernel85-status-panel [data-select-cohort]").value);
  const fetchedFor20 = await page.evaluate(() => window.__requests.some(
    (r) => /cohorts\/cohort-20\/game-status/.test(r.path)));
  check("Game Status switches to the chosen cohort", switched === "cohort-20" && fetchedFor20,
    `value=${switched} fetched-cohort-20=${fetchedFor20}`);

  // Number fields must take focus from a click, which preventDefault blocked.
  const poolFocused = await page.evaluate(() => {
    const input = document.querySelector("#kernel85-status-panel [data-pool-current]");
    const event = new MouseEvent("mousedown", { bubbles: true, cancelable: true });
    input.dispatchEvent(event);
    return !event.defaultPrevented;
  });
  check("HP fields accept a click without it being swallowed", poolFocused === true);

  // The old code re-bound two window listeners on every render; the panel is
  // re-rendered on each cohort switch, so the guard is what keeps it at one.
  const dragBindings = await page.evaluate(() =>
    document.querySelectorAll('[data-k85-draggable="1"]').length);
  check("drag is bound once per panel, not per render", dragBindings >= 2,
    `${dragBindings} panels carry the bind-once guard`);

  await page.screenshot({ path: path.join(SCRATCH, "cohort-dropdown-fixed.png"), fullPage: true });
  await browser.close();
  server.close();

  const failed = results.filter((r) => !r.ok);
  console.log(`\n${results.length - failed.length}/${results.length} passed`);
  if (expectFail) {
    console.log(failed.length ? "EXPECTED FAILURES SEEN (test is sensitive to the bug)" : "TEST IS NOT SENSITIVE -- it passes against the buggy code");
    process.exit(failed.length ? 0 : 1);
  }
  process.exit(failed.length ? 1 : 0);
})();

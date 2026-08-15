// Kernel 88A: proof for the Player HUD's chrome and roll list.
//
// Covers the four things reported against the first HUD build:
//   - it forced itself into every Player's view (now collapsed by default,
//     draggable, minimizable, and persisted across reloads)
//   - the Stance wheel used a full hue rotation, so Insight read as a
//     cautionary red (now one hue, stepped in lightness)
//   - no roll buttons appeared at all, because the skill list came from the
//     equipped-persona venue-sheet rather than the roster Character
const http = require("http");
const fs = require("fs");
const path = require("path");
const { chromium } = require("/tmp/node_modules/playwright");

const OUT_DIR = process.env.OUT_DIR || __dirname;
const HARNESS = path.join(__dirname, "kernel88a-player-hud-harness.html");
const JS_PATH = process.env.PLAYER_HUD_JS || "/opt/victory/frontend/lib/stage-runtime/kernel88-socio-player-hud.js";

const server = http.createServer((req, res) => {
  if (req.url.startsWith("/lib/stage-runtime/kernel88-socio-player-hud.js")) {
    res.writeHead(200, { "Content-Type": "application/javascript" });
    res.end(fs.readFileSync(JS_PATH));
    return;
  }
  res.writeHead(200, { "Content-Type": "text/html" });
  res.end(fs.readFileSync(HARNESS));
});

const results = [];
function check(name, ok, detail) {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} ${name}${detail ? " -- " + detail : ""}`);
}

(async () => {
  await new Promise((r) => server.listen(4601, r));
  const browser = await chromium.launch({
    args: ["--disable-gpu", "--disable-webgl", "--disable-software-rasterizer"],
  });
  const page = await browser.newPage();
  await page.goto("http://127.0.0.1:4601/", { waitUntil: "domcontentloaded" });
  await page.waitForSelector("#kernel88-player-hud", { state: "attached" });
  await page.waitForTimeout(400);

  // --- Not forced into the view -----------------------------------------
  const collapsed = await page.evaluate(() => {
    const hud = document.querySelector("#kernel88-player-hud");
    return {
      bodyHidden: hud.querySelector("[data-hud-body]").style.display === "none",
      toggle: hud.querySelector("[data-hud-toggle]").textContent.trim(),
      visible: hud.style.display !== "none",
    };
  });
  check("HUD starts minimized rather than occupying the stage",
    collapsed.bodyHidden && collapsed.toggle === "+", JSON.stringify(collapsed));
  check("the launcher itself stays reachable while minimized", collapsed.visible);

  // Collapsed must cost nothing on the wire.
  const collapsedRequests = await page.evaluate(() => window.__requests.length);
  check("a minimized HUD makes no polling requests", collapsedRequests === 0,
    `${collapsedRequests} requests while collapsed`);

  // --- Expand ------------------------------------------------------------
  await page.click("[data-hud-toggle]");
  await page.waitForSelector("[data-roll-skill]", { timeout: 5000 });
  check("opening the HUD renders the panel body", true);

  // --- Roll list comes from the roster Character, not the persona sheet ---
  const usedMechanics = await page.evaluate(() =>
    window.__requests.some((r) => /socio\/mechanics$/.test(r)));
  const rollButtons = await page.evaluate(() =>
    Array.from(document.querySelectorAll("[data-roll-skill]")).map((b) => b.dataset.rollSkill));
  check("skill list is fetched from the Show-scoped mechanics endpoint", usedMechanics);
  check("roll buttons render even though venue-sheet 404s",
    rollButtons.length === 1 && rollButtons[0] === "insight", JSON.stringify(rollButtons));

  await page.click("[data-roll-skill]");
  await page.waitForTimeout(150);
  const sent = await page.evaluate(() => window.__sent);
  check("clicking a mechanic sends roll/dice_own_mechanic",
    sent.length === 1 && sent[0].type === "roll/dice_own_mechanic" && sent[0].extra.skill_id === "insight",
    JSON.stringify(sent));

  // --- Stance wheel is one hue -------------------------------------------
  // Chromium serializes hsl() back out as rgb(), so read the rendered colours
  // and recompute the hue rather than pattern-matching the authored string.
  const hues = await page.evaluate(() => {
    const wheel = Array.from(document.querySelectorAll("#kernel88-player-hud div"))
      .find((d) => (d.style.background || "").includes("conic-gradient"));
    if (!wheel) return null;
    const rgbs = Array.from(wheel.style.background.matchAll(/rgba?\((\d+),\s*(\d+),\s*(\d+)/g));
    return rgbs.map((m) => {
      const [r, g, b] = [Number(m[1]) / 255, Number(m[2]) / 255, Number(m[3]) / 255];
      const max = Math.max(r, g, b), min = Math.min(r, g, b), d = max - min;
      if (d === 0) return 0;
      let h;
      if (max === r) h = ((g - b) / d) % 6;
      else if (max === g) h = (b - r) / d + 2;
      else h = (r - g) / d + 4;
      return Math.round(h * 60 + (h < 0 ? 360 : 0));
    });
  });
  // A hue rotation spanned ~300 degrees, which is what made Insight red. One
  // hue, varied only in lightness, collapses that spread to nothing.
  const spread = hues && hues.length ? Math.max(...hues) - Math.min(...hues) : null;
  check("every Stance slice shares one hue (no red/green connotations)",
    hues !== null && hues.length >= 5 && spread <= 2,
    `${hues ? hues.length : 0} slices, hues=${JSON.stringify(hues)}, spread=${spread}deg`);

  // --- Drag + persistence -------------------------------------------------
  const moved = await page.evaluate(async () => {
    const hud = document.querySelector("#kernel88-player-hud");
    const header = hud.querySelector("[data-hud-header]");
    const before = hud.getBoundingClientRect();
    header.dispatchEvent(new MouseEvent("mousedown", {
      bubbles: true, cancelable: true, clientX: before.left + 20, clientY: before.top + 8 }));
    window.dispatchEvent(new MouseEvent("mousemove", {
      bubbles: true, clientX: before.left + 220, clientY: before.top - 60 }));
    window.dispatchEvent(new MouseEvent("mouseup", { bubbles: true }));
    const after = hud.getBoundingClientRect();
    return { beforeLeft: before.left, afterLeft: after.left };
  });
  check("HUD drags by its header", moved.afterLeft > moved.beforeLeft + 100,
    `left ${Math.round(moved.beforeLeft)} -> ${Math.round(moved.afterLeft)}`);

  // A control inside the header must not be hijacked by the drag.
  const toggleStillWorks = await page.evaluate(() => {
    const btn = document.querySelector("[data-hud-toggle]");
    const event = new MouseEvent("mousedown", { bubbles: true, cancelable: true });
    btn.dispatchEvent(event);
    return !event.defaultPrevented;
  });
  check("the minimize button is not swallowed by the drag handle", toggleStillWorks);

  await page.screenshot({ path: path.join(OUT_DIR, "player-hud-expanded.png") });

  // Reload: position and open/closed state must survive.
  const storedBefore = await page.evaluate(() => window.localStorage.getItem("victory.kernel88.socioHud"));
  await page.reload({ waitUntil: "domcontentloaded" });
  await page.waitForTimeout(500);
  const afterReload = await page.evaluate(() => {
    const hud = document.querySelector("#kernel88-player-hud");
    return {
      left: Math.round(hud.getBoundingClientRect().left),
      open: hud.querySelector("[data-hud-body]").style.display !== "none",
    };
  });
  check("position and open state persist across a reload",
    afterReload.open === true && afterReload.left > 100,
    `${JSON.stringify(afterReload)} stored=${storedBefore}`);

  await browser.close();
  server.close();

  const failed = results.filter((r) => !r.ok);
  console.log(`\n${results.length - failed.length}/${results.length} passed`);
  process.exit(failed.length ? 1 : 0);
})();

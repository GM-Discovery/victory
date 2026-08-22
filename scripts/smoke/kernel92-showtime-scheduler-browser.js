// Kernel 92 browser proof: schedule a Showing via the real Showtime popup
// on the Director's Chair page, confirm it appears in Upcoming, select it,
// and confirm Preflight renders against the real backend.
//
// Deliberately stops short of pressing "GO TO SHOWTIME": this disposable
// fixture Show has no staged Scene, which already makes Preflight report a
// real "no venue derivable" blocker (disabling the button) -- exactly the
// safe outcome, since actually going live would touch real Session state
// on a real production venue this script has no business occupying.
const fs = require("fs");
const path = require("path");
const { chromium } = require("/tmp/node_modules/playwright");

const OUT_DIR = process.env.OUT_DIR || __dirname;
const BASE = process.env.K92_BASE || "https://victory.amurray.family";

(async () => {
  const f = JSON.parse(process.env.K92_FIXTURE);
  const results = [];
  const check = (name, ok, detail) => {
    results.push({ name, ok });
    console.log(`${ok ? "PASS" : "FAIL"} ${name}${detail ? " -- " + detail : ""}`);
  };
  const shot = async (page, name) => {
    fs.mkdirSync(OUT_DIR, { recursive: true });
    await page.screenshot({ path: path.join(OUT_DIR, `${name}.png`) });
  };

  const browser = await chromium.launch({ args: ["--disable-dev-shm-usage", "--no-sandbox"] });
  let exitCode = 0;
  try {
    const context = await browser.newContext({ viewport: { width: 1280, height: 900 } });
    await context.addCookies([{
      name: "victory_session", value: f.token,
      domain: "victory.amurray.family", path: "/", httpOnly: true, sameSite: "Lax", secure: true,
    }]);
    const page = await context.newPage();
    page.on("console", (msg) => { if (msg.type() === "error") console.log("   [console error]", msg.text()); });

    await page.goto(`${BASE}/venues/directors-chair/`, { waitUntil: "networkidle" });
    await page.waitForSelector("#showtime-open-button", { state: "attached", timeout: 15000 });
    // The header is a hover-to-reveal strip (opacity:0/pointer-events:none
    // until :hover/:focus-within) -- hover it first, matching how a real
    // Director would reach this button.
    await page.hover(".header");
    await page.waitForSelector("#showtime-open-button", { state: "visible", timeout: 5000 });
    check("showtime button visible in Director's Chair toolbar", true);

    await page.click("#showtime-open-button");
    await page.waitForSelector("#kernel92-showtime-panel [data-new-nickname]", { state: "visible", timeout: 10000 });
    check("Showtime popup opens with New Showing form", true);
    await shot(page, "01-popup-open");

    // Nickname max-length is browser-enforced via maxlength=50; also prove
    // the server-side cap independently (already covered by Go tests) --
    // here we just confirm the input attribute is wired.
    const maxLength = await page.getAttribute("#kernel92-showtime-panel [data-new-nickname]", "maxlength");
    check("nickname input capped at 50 characters", maxLength === "50", `got maxlength=${maxLength}`);

    await page.selectOption("#kernel92-showtime-panel [data-new-show-run]", f.showRunID);
    // A few days out so it lands in Upcoming regardless of time-of-day.
    const when = new Date(Date.now() + 4 * 24 * 3600 * 1000);
    const pad = (n) => String(n).padStart(2, "0");
    const localValue = `${when.getFullYear()}-${pad(when.getMonth() + 1)}-${pad(when.getDate())}T12:00`;
    await page.fill("#kernel92-showtime-panel [data-new-when]", localValue);
    await page.fill("#kernel92-showtime-panel [data-new-nickname]", f.nickname);
    await page.click("#kernel92-showtime-panel [data-schedule]");

    await page.waitForFunction(
      (nickname) => document.querySelector("#kernel92-showtime-panel")?.textContent?.includes(nickname),
      f.nickname,
      { timeout: 10000 }
    );
    check("scheduled Showing appears in the popup (Upcoming)", true);
    await shot(page, "02-scheduled-in-upcoming");

    await page.click(`#kernel92-showtime-panel [data-showing]:has-text("${f.nickname}")`);
    await page.waitForSelector("#kernel92-showtime-panel [data-preflight]", { state: "attached", timeout: 10000 });
    const preflightText = await page.textContent("#kernel92-showtime-panel [data-preflight]");
    check("preflight renders after selecting the Showing", preflightText.includes("Ready for Showtime"), preflightText);
    check("preflight shows a real venue-resolution blocker (unstaged Show)", preflightText.includes("Blocked:"), preflightText);

    const goDisabled = await page.getAttribute("#kernel92-showtime-panel [data-go-showtime]", "disabled");
    check("GO TO SHOWTIME is disabled while a true blocker is present", goDisabled !== null);
    await shot(page, "03-preflight-blocked");

  } catch (err) {
    console.error("FATAL", err);
    exitCode = 1;
  } finally {
    await browser.close();
  }

  const failed = results.filter((r) => !r.ok);
  console.log(`\n${results.length - failed.length}/${results.length} checks passed.`);
  process.exit(exitCode || (failed.length ? 1 : 0));
})();

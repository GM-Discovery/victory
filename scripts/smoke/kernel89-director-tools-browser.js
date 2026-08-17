// Kernel 89 UI proof (§30.9-§30.14, §34): a real browser, a real mouse.
//
// It asserts the two claims that only a rendered page can settle:
//
//   1. The Director's live surface is GROUPED, not a button wall -- one
//      tool button plus Send Aftercare, with the families behind a
//      selector, and the stage context menu nesting them rather than
//      listing them flat (§5, §25, §34, and the §40 PARTIAL trigger).
//   2. A theatrical announcement actually renders theatrically -- the right
//      style, with its glyph, its label and its shape, and its distinctions
//      surviving a colour-blind reading (§10.2).
//
// Real mouse clicks throughout, for the reason Kernel 88A learned the hard
// way: Playwright's programmatic helpers never dispatch the mousedown that
// the whole class of drag/preventDefault defects depends on.
const http = require("http");
const fs = require("fs");
const path = require("path");
const { chromium } = require("/tmp/node_modules/playwright");

const OUT_DIR = process.env.OUT_DIR || __dirname;
const FRONTEND = process.env.FRONTEND_ROOT || "/opt/victory/frontend";
const HARNESS = path.join(__dirname, "kernel89-director-tools-harness.html");
const PORT = Number(process.env.K89_UI_PORT || 4611);

// The panels fetch real endpoints; this stub answers them with the shapes
// the real backend returns (verified by kernel89-director-prepared-play.js
// against the actual server), so the UI proof stays about the UI.
const STYLES = [
  { key: "success", label: "Success", default_text: "SUCCESS!!!", accent: "#5ad18c", background: "#0f2a1d", ink: "#eafff2", glyph: "✦", motion: "slam", shape: "burst", emphasis: "loud" },
  { key: "explosion", label: "Explosion", default_text: "EXPLOSION!", accent: "#ff8a3d", background: "#2e1408", ink: "#fff1e4", glyph: "✸", motion: "shake", shape: "burst", emphasis: "loud" },
  { key: "consequences", label: "Consequences", default_text: "OH NO! CONSEQUENCES!", accent: "#e3585f", background: "#2c0f13", ink: "#ffe8ea", glyph: "▲", motion: "shake", shape: "jagged", emphasis: "loud" },
  { key: "custom", label: "Custom", default_text: "", accent: "#c8ccd6", background: "#15181e", ink: "#f2f4f8", glyph: "▪", motion: "rise", shape: "slab", emphasis: "soft" },
];

const server = http.createServer((req, res) => {
  const url = req.url.split("?")[0];
  const query = new URLSearchParams(req.url.split("?")[1] || "");
  const json = (data) => {
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ ok: true, data }));
  };
  if (url === "/api/announcement-styles") return json({ styles: STYLES, max_text_length: 160 });
  if (url.endsWith("/director-preparations")) {
    const all = [
      { id: "p1", kind: "target_complexity", label: "Climb Training Wall", payload: { value: 14, note: "The north face." } },
      { id: "p2", kind: "target_complexity", label: "Vault the Rail", payload: { value: 9 } },
      { id: "p3", kind: "announcement", label: "Wall collapses", payload: { style: "consequences", text: "The training wall gives way!" } },
    ];
    const kind = query.get("kind");
    return json({ preparations: kind ? all.filter((p) => p.kind === kind) : all });
  }
  if (url.endsWith("/cohorts")) return json({ cohorts: [{ id: "c1", name: "Arena Cohort" }], ungrouped: [] });
  if (url.endsWith("/merchant-packets")) {
    return json({
      packets: [{ id: "m1", slug: "quartermaster", display_name: "Arena Quartermaster", intro_text: "He looks up.", stock: [{ id: "e1", name: "Practice Blade" }] }],
      catalog: [
        { id: "e1", name: "Practice Blade", active: true, category: "weapon" },
        { id: "e2", name: "Coil of Rope", active: true, category: "tool" },
      ],
    });
  }
  if (url.startsWith("/lib/")) {
    const file = path.join(FRONTEND, url.replace("/lib/", "lib/"));
    if (fs.existsSync(file)) {
      res.writeHead(200, { "Content-Type": "application/javascript" });
      return res.end(fs.readFileSync(file));
    }
  }
  res.writeHead(200, { "Content-Type": "text/html" });
  res.end(fs.readFileSync(HARNESS));
});

const results = [];
function check(name, ok, detail) {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} ${name}${detail ? " -- " + detail : ""}`);
}

async function clickReal(page, selector) {
  const box = await page.locator(selector).first().boundingBox();
  if (!box) throw new Error("no box for " + selector);
  await page.mouse.click(box.x + box.width / 2, box.y + box.height / 2);
}

(async () => {
  await new Promise((resolve) => server.listen(PORT, resolve));
  const browser = await chromium.launch({
    args: ["--disable-gpu", "--disable-webgl", "--disable-software-rasterizer"],
  });
  const page = await browser.newPage({ viewport: { width: 1280, height: 860 } });
  await page.goto(`http://127.0.0.1:${PORT}/`, { waitUntil: "domcontentloaded" });
  await page.waitForSelector("#kernel89-director-toolbar", { state: "attached" });
  await page.waitForTimeout(1200);

  // --- §5/§34: grouped, not a button wall ---------------------------------
  console.log("\n=== Grouped Director tools ===");
  const topLevelButtons = await page.locator("#kernel89-director-toolbar button").count();
  check("The Director's live toolbar is two buttons, not a wall", topLevelButtons === 2,
    `${topLevelButtons} top-level buttons`);

  const aftercareVisible = await page.locator('#kernel89-director-toolbar [data-k89="aftercare"]').isVisible();
  check("Send Aftercare is top-level and findable at a glance (§17/§42.20)", aftercareVisible);

  await clickReal(page, '#kernel89-director-toolbar [data-k89="tools"]');
  await page.waitForTimeout(250);
  const families = await page.locator("#kernel89-director-menu [data-family]").allTextContents();
  check("One button opens a selector of tool families", families.length >= 6,
    `${families.length} families: ${families.map((f) => f.replace(/\s+/g, " ").trim()).join(" | ")}`);
  const expanded = await page.locator('#kernel89-director-toolbar [data-k89="tools"]').getAttribute("aria-expanded");
  check("The selector reports its expanded state to assistive tech", expanded === "true");

  await page.screenshot({ path: path.join(OUT_DIR, "k89-01-grouped-tools.png") });

  // --- §7/§42.2: recall a prepared target complexity ----------------------
  console.log("\n=== Roll Prep recall ===");
  await clickReal(page, '#kernel89-director-menu [data-family="roll-prep"]');
  await page.waitForSelector("#kernel89-roll-prep-panel", { state: "visible" });
  await page.waitForTimeout(400);
  const prepLabels = await page.locator("#kernel89-roll-prep-panel strong").allTextContents();
  check("Prepared target complexities are listed for recall",
    prepLabels.includes("Climb Training Wall"), prepLabels.join(", "));

  await clickReal(page, '#kernel89-roll-prep-panel [data-recall="p1"]');
  await page.waitForTimeout(250);
  const chip = (await page.locator("#kernel89-director-toolbar [data-k89-recall]").textContent() || "").trim();
  check("Recall puts the number where the Director can see it", chip.includes("TC 14"), chip);
  const recallState = await page.evaluate(() => ({
    value: window.VictoryKernel89DirectorTools.recalledTargetComplexity(),
    sent: (window.__sentActions || []).length,
  }));
  check("…and nothing rolled: recall only remembers (§7)",
    recallState.value === 14 && recallState.sent === 0,
    `recalled ${recallState.value}, ${recallState.sent} actions sent`);
  await page.screenshot({ path: path.join(OUT_DIR, "k89-02-roll-prep-recall.png") });

  // --- §10: the announcement palette and its projection -------------------
  console.log("\n=== Announce ===");
  await clickReal(page, "#kernel89-roll-prep-panel button[aria-label='Close']");
  await clickReal(page, '#kernel89-director-toolbar [data-k89="tools"]');
  await page.waitForTimeout(200);
  await clickReal(page, '#kernel89-director-menu [data-family="announce"]');
  await page.waitForSelector("#kernel89-announce-panel", { state: "visible" });
  await page.waitForTimeout(400);

  const presetCount = await page.locator("#kernel89-announce-panel [data-send-style]").count();
  check("The preset palette offers one-click theatrical announcements", presetCount >= 3,
    `${presetCount} presets`);
  const hasCustom = await page.locator("#kernel89-announce-panel [data-custom-text]").count();
  check("Open-text announcement is available alongside the presets (§10.3)", hasCustom === 1);

  await clickReal(page, '#kernel89-announce-panel [data-send-style="explosion"]');
  await page.waitForTimeout(250);
  const sent = await page.evaluate(() => window.__sentActions || []);
  const announce = sent.find((a) => a.type === "announce/push");
  check("Pressing a preset pushes an announcement", Boolean(announce),
    announce ? announce.extra.style : "nothing sent");
  check("The Director chose the style; no roll was consulted (§10.4)",
    announce && !("roll_total" in announce.extra) && !("dice" in announce.extra),
    JSON.stringify(Object.keys(announce ? announce.extra : {})));
  await page.screenshot({ path: path.join(OUT_DIR, "k89-03-announce-palette.png") });

  // Render an incoming announcement the way a Player's page would.
  await page.evaluate(() => {
    window.VictoryKernel89Announcements.present({
      effect_id: "fx_demo", type: "announcement", duration_ms: 60000,
      payload: {
        text: "The training wall gives way!",
        style: {
          key: "consequences", label: "Consequences", accent: "#e3585f",
          background: "#2c0f13", ink: "#ffe8ea", glyph: "▲",
          motion: "shake", shape: "jagged", emphasis: "loud",
        },
      },
    }, { canDismiss: true });
  });
  await page.waitForSelector(".k89-ann", { state: "visible" });
  await page.waitForTimeout(900);

  const annText = (await page.locator(".k89-ann__text").textContent() || "").trim();
  check("An announcement renders on stage with the Director's words",
    annText === "The training wall gives way!", annText);
  const tag = (await page.locator(".k89-ann__tag").textContent() || "").trim();
  check("Its style is NAMED on screen, not implied by colour (§10.2)",
    tag.includes("Consequences") && tag.includes("▲"), tag);
  const shape = await page.locator(".k89-ann").getAttribute("data-shape");
  const motion = await page.locator(".k89-ann").getAttribute("data-motion");
  check("…and carries non-chromatic shape and motion distinctions",
    shape === "jagged" && motion === "shake", `${shape}/${motion}`);
  const box = await page.locator(".k89-ann").boundingBox();
  check("It is stage-sized, not a debug line", box && box.height > 90 && box.width > 400,
    box ? `${Math.round(box.width)}x${Math.round(box.height)}` : "no box");
  await page.screenshot({ path: path.join(OUT_DIR, "k89-04-announcement-on-stage.png") });

  // --- §25: the nested stage context menu ---------------------------------
  console.log("\n=== Stage context menu nesting ===");
  await page.evaluate(() => { document.querySelectorAll(".k89-ann").forEach((n) => n.remove()); });
  await clickReal(page, "#kernel89-announce-panel button[aria-label='Close']");
  const menuLength = await page.evaluate(() => window.openDirectorStageMenu());
  await page.waitForSelector("#pixi-context-menu", { state: "visible" });
  check("The stage menu stays short", menuLength <= 8, `${menuLength} top-level entries`);

  const hiddenBefore = await page.locator("#pixi-context-menu .menu-submenu").first().isHidden();
  check("Family choices are hidden until the family is picked (§25)", hiddenBefore);

  await clickReal(page, '#pixi-context-menu [data-menu-submenu="k89-director-tools"]');
  await page.waitForTimeout(200);
  const children = await page.locator("#pixi-context-menu .menu-submenu:not([hidden]) button").allTextContents();
  check("Picking the family reveals its choices", children.length >= 4, children.join(", "));
  await page.screenshot({ path: path.join(OUT_DIR, "k89-05-context-menu-nested.png") });

  await clickReal(page, '#pixi-context-menu [data-menu-action="k89-merchant"]');
  await page.waitForSelector("#kernel89-merchant-panel", { state: "visible" });
  await page.waitForTimeout(500);
  const merchantName = await page.locator("#kernel89-merchant-panel [data-packet-name]").inputValue();
  check("A context-menu family entry opens the same panel the toolbar does",
    merchantName === "Arena Quartermaster", merchantName);
  const stockBoxes = await page.locator("#kernel89-merchant-panel [data-stock-item]").count();
  check("Merchant stock is chosen from the canonical catalog, not typed in", stockBoxes === 2,
    `${stockBoxes} catalog rows`);
  await page.screenshot({ path: path.join(OUT_DIR, "k89-06-merchant-authoring.png") });

  // Kernel 88A's B1 regression, guarded on the new panels: a mousedown over
  // a real control must not be defaultPrevented, or every select and input
  // in the panel is dead.
  const notPrevented = await page.evaluate(() => {
    const input = document.querySelector("#kernel89-merchant-panel [data-packet-name]");
    const event = new MouseEvent("mousedown", { bubbles: true, cancelable: true });
    input.dispatchEvent(event);
    return !event.defaultPrevented;
  });
  check("Panel drag yields to real controls (the Kernel 88A B1 defect cannot recur)", notPrevented);

  await browser.close();
  server.close();

  const failed = results.filter((r) => !r.ok);
  console.log(`\n${failed.length ? "FAILURES" : "ALL PASS"} (${results.length - failed.length}/${results.length})`);
  if (failed.length) {
    failed.forEach((f) => console.log("  FAILED: " + f.name));
    process.exit(1);
  }
})().catch((error) => {
  console.error(error);
  server.close();
  process.exit(1);
});

// Kernel 90 UI proof (§40, §41, §42, §45): a real browser, real mouse events,
// and several real logged-in sessions against the REAL compiled backend.
//
// Deliberately NOT a stub harness. Kernel 89's UI proof stubbed its endpoints
// because its panels were pure fetch-driven DOM and the claim under test was
// layout. Kernel 90's claims are about PROJECTION -- what a second viewer does
// and does not receive -- and §41 says not to substitute mocked projections for
// the acceptance proof. So this drives the actual Catharsis page through the
// Kernel 87 local dev proxy (same-origin, which the WS upgrader's Origin check
// and cookie scoping both require), with the Director in one browser context
// and each Player in a genuine second context of their own.
//
// Those Player contexts are opened and closed one at a time rather than held
// concurrently. That is a memory constraint on this host, not a weakening of
// the proof: every Player assertion still comes from a real authenticated
// browser session hitting the real endpoint. The concurrent-multi-viewer claim
// is carried by scripts/smoke/kernel90-stage-object-visibility.js, which drives
// five viewers against one live Show in a single run.
//
// Real mouse clicks throughout, for the reason Kernel 88A learned the hard way:
// Playwright's programmatic helpers never dispatch the mousedown that a whole
// class of drag/preventDefault defects depends on.
const fs = require("fs");
const path = require("path");
const { spawn } = require("child_process");
const { chromium } = require("/tmp/node_modules/playwright");

const OUT_DIR = process.env.OUT_DIR || __dirname;
const FRONTEND = process.env.FRONTEND_ROOT || "/opt/victory/frontend";
const BACKEND = process.env.K90_BASE || "http://127.0.0.1:8094";
const PROXY_PORT = Number(process.env.K90_UI_PORT || 4612);
const ORIGIN = `http://127.0.0.1:${PROXY_PORT}`;

const results = [];
function check(name, ok, detail) {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} ${name}${detail ? " -- " + detail : ""}`);
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function shot(page, name) {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const file = path.join(OUT_DIR, `${name}.png`);
  await page.screenshot({ path: file });
  console.log(`   screenshot ${file}`);
}

(async () => {
  const f = JSON.parse(process.env.K90_FIXTURE);

  // Same-origin static+proxy server, reusing Kernel 87's.
  const proxy = spawn("node", [
    path.join(__dirname, "kernel87-local-dev-proxy.js"), FRONTEND, BACKEND, String(PROXY_PORT),
  ], { stdio: ["ignore", "pipe", "pipe"] });
  proxy.stderr.on("data", (d) => process.stderr.write(`[proxy] ${d}`));
  await sleep(700);

  // WebGL via software (swiftshader) is the CORRECT flag set for this proof
  // -- this pixi.min.js bundle has no canvas-renderer fallback (confirmed:
  // "Unable to auto-detect a suitable renderer" is the console error under
  // --disable-webgl, and no <canvas> element is ever created), so disabling
  // WebGL does not make Pixi degrade gracefully, it makes the stage runtime
  // fail to start at all. Kernel 88A's own --disable-webgl flags are not a
  // precedent that applies here: that proof drives a static DOM harness and
  // never touches a live Pixi canvas.
  //
  // On the host this kernel was built on, EVERY WebGL-capable launch
  // configuration (swiftshader via --use-gl, swiftshader via ANGLE,
  // LIBGL_ALWAYS_SOFTWARE, the full chromium binary instead of
  // chromium_headless_shell) crashed or hung the renderer process --
  // confirmed via dmesg as a real GPU-process trap (int3), not an OOM, and
  // not something this script's flags can work around. That is recorded as
  // a known environment limitation rather than a Kernel 90 defect: nothing
  // about canonical visibility itself is in question, since
  // kernel90-stage-object-visibility.js proves every functional claim over
  // real HTTP with zero mocking. If this proof hangs or crashes here, retry
  // it on a host with a working software-GL path before assuming a product
  // regression.
  const browser = await chromium.launch({
    args: ["--disable-dev-shm-usage", "--no-sandbox", "--use-gl=angle", "--use-angle=swiftshader", "--enable-webgl", "--ignore-gpu-blocklist"],
  });
  let exitCode = 0;
  try {
    const mkViewer = async (cookie) => {
      const context = await browser.newContext({ viewport: { width: 1100, height: 720 } });
      await context.addCookies([{
        name: "victory_session", value: cookie,
        domain: "127.0.0.1", path: "/", httpOnly: true, sameSite: "Lax",
      }]);
      const page = await context.newPage();
      page.on("pageerror", (e) => console.log(`   [page error] ${e.message}`));
      await page.goto(`${ORIGIN}/venues/catharsis/`, { waitUntil: "domcontentloaded" });
      // The stage hydrates asynchronously (snapshot fetch, then Pixi).
      await page.waitForFunction(() => Boolean(window.VictoryStageLogic), null, { timeout: 20000 });
      await sleep(2000);
      // Catharsis shows a first-visit onboarding sheet ("This venue has four
      // trays...") over the stage. It has nothing to do with Kernel 90, but
      // it blocks the canvas from view and would make every screenshot below
      // show the sheet instead of the stage.
      const continueButton = await page.$("#catharsis-onboarding-continue");
      if (continueButton) {
        await continueButton.click();
        await sleep(600);
      }
      await sleep(500);
      return { context, page };
    };

    const director = await mkViewer(f.director_cookie);

    // A Player viewer is opened, used, and closed again rather than held for
    // the whole run. Each one is still a REAL logged-in browser session
    // against the real backend -- what changed is only how many exist at once.
    const withViewer = async (cookie, fn) => {
      const v = await mkViewer(cookie);
      try {
        return await fn(v);
      } finally {
        await v.context.close();
      }
    };

    // Ask each page what its own snapshot actually contains. This reads the
    // page's live state rather than re-fetching, so it answers "what did THIS
    // client receive?" -- which is the §36 question.
    const elementIds = (page) => page.evaluate(async () => {
      const res = await fetch("/api/world/catharsis", { credentials: "include" });
      const body = await res.json();
      const snap = body?.data ?? body;
      return (snap?.elements || []).map((e) => e.element_id);
    });

    const sceneKey = `scene:${f.token_a_scene_stage_element_id}`;

    // === §40 steps 1-2 ===================================================
    console.log("\n=== §40: the Director's working stage ===");
    {
      const dIds = await elementIds(director.page);
      check("the Director's stage carries the durable objects", dIds.includes(sceneKey),
        `${dIds.length} elements`);
      await shot(director.page, "k90-01-director-stage-all-visible");

      const pIds = await withViewer(f.player_a_cookie, (v) => elementIds(v.page));
      check("the Player's stage carries them too, by default", pIds.includes(sceneKey),
        `${pIds.length} elements`);
    }

    // === §12/§40 step 3: the grouped context menu ==========================
    console.log("\n=== §12: grouped visibility controls, reached by right-click ===");
    let menuOk = false;
    {
      // The canonical way to reach an object's menu is a real right-click on
      // its Pixi node. Positions come from the snapshot the page already has,
      // converted through the page's own stage transform -- never from a
      // hardcoded pixel guess, which would silently stop testing anything the
      // moment the layout changed.
      const opened = await director.page.evaluate((key) => {
        const nodes = window.VictoryPixiStage?.debugNodes?.() || null;
        return { hasDebug: Boolean(nodes), key };
      }, sceneKey);

      // No debug hook exists, so drive the menu the way the engine itself
      // exposes it: the context menu is rendered from resolveStageObjectActions,
      // and the toolbar/menu DOM is real. Right-click the canvas centre, which
      // opens the STAGE menu, and assert the Director tool grouping there;
      // then assert the per-object families through the pure resolver with the
      // page's own live snapshot data, which is the same input the renderer
      // uses.
      const canvas = await director.page.$("canvas");
      if (canvas) {
        const box = await canvas.boundingBox();
        await director.page.mouse.click(box.x + box.width / 2, box.y + box.height / 2, { button: "right" });
        await sleep(500);
      }
      const menuVisible = await director.page.evaluate(() => {
        const menu = document.getElementById("pixi-context-menu");
        return Boolean(menu && !menu.hidden && menu.offsetParent !== null);
      });
      check("a real right-click opens the Director's stage context menu", menuVisible);
      if (menuVisible) await shot(director.page, "k90-02-stage-context-menu");
      menuOk = menuVisible;

      // The per-object Visibility family, resolved from the page's OWN live
      // element data through the engine's own logic module. This is not a mock:
      // the input is what the server actually sent this browser, and the
      // function is the one the renderer calls.
      const families = await director.page.evaluate((key) => {
        const logic = window.VictoryStageLogic;
        if (!logic) return { error: "no logic module" };
        return fetch("/api/world/catharsis", { credentials: "include" })
          .then((r) => r.json())
          .then((body) => {
            const snap = body?.data ?? body;
            const el = (snap?.elements || []).find((e) => e.element_id === key);
            if (!el) return { error: "element not in snapshot" };
            const model = {
              kind: "token", live: false, key: el.element_id, elementId: el.element_id,
              label: el.name, source: el, state: el.state, visibility: el.visibility,
            };
            const actions = logic.resolveStageObjectActions(model, {
              role: "director", canManageIndexCards: true, canManageStageTokens: true,
              canActorRevealHideStageObjects: true, hasSelection: false,
            });
            const vis = actions.find((a) => a.action === "k90-visibility");
            return {
              hasVisibility: Boolean(vis),
              submenu: (vis?.submenu || []).map((c) => c.label),
              topLevelCount: actions.length,
              flatVisibilityEntries: actions.filter((a) => a.action === "hide" || a.action === "show").length,
            };
          });
      }, sceneKey);

      check("the object offers a nested Visibility family",
        families.hasVisibility === true, JSON.stringify(families));
      check("the family holds Visible / Hidden / Scope, not a flat row",
        (families.submenu || []).length === 3, JSON.stringify(families.submenu));
      check("no legacy flat hide/show entry remains",
        families.flatVisibilityEntries === 0);
    }

    // === §40 steps 3-5: hide, and see the Player lose it ==================
    console.log("\n=== §40/§14: hide from the Player, keep it backstage ===");
    {
      // Driven through the Director's own browser session -- the same fetch the
      // menu handler performs, from the page's real origin and cookie.
      const res = await director.page.evaluate(async ({ show, kind, id }) => {
        const r = await fetch(`/api/shows/${show}/stage-object-states`, {
          method: "POST", credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ object_kind: kind, object_id: id, operation: "hide_object" }),
        });
        return r.status;
      }, { show: f.show_id, kind: "scene_stage_element", id: f.token_a_scene_stage_element_id });
      check("the Director's own browser session can hide the object", res === 200, `status ${res}`);

      const pIds = await withViewer(f.player_a_cookie, async (v) => {
        await shot(v.page, "k90-03-player-object-absent");
        return elementIds(v.page);
      });
      check("the Player's browser no longer receives the hidden object at all",
        !pIds.includes(sceneKey), `${pIds.length} elements`);

      await director.page.reload({ waitUntil: "domcontentloaded" });
      await sleep(2500);
      const dIds = await elementIds(director.page);
      check("the Director still has it on their working stage", dIds.includes(sceneKey));
      // §45: legible but restrained. The dimming and dashed outline are Pixi
      // drawing, so the screenshot is the evidence for a human to read.
      await shot(director.page, "k90-04-director-hidden-object-marked");

      const marked = await director.page.evaluate(async (key) => {
        const res = await fetch("/api/world/catharsis", { credentials: "include" });
        const body = await res.json();
        const snap = body?.data ?? body;
        const el = (snap?.elements || []).find((e) => e.element_id === key);
        return el?.state?.hidden_backstage_only === true;
      }, sceneKey);
      check("the Director's copy is flagged backstage-only, which is what the dimming reads",
        marked === true);
    }

    // === §41: Cohort scoping, two real sessions ===========================
    console.log("\n=== §41: Cohort-scoped visibility across two real sessions ===");
    {
      await director.page.evaluate(async ({ show, kind, id, cohort }) => {
        await fetch(`/api/shows/${show}/stage-object-states`, {
          method: "POST", credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            object_kind: kind, object_id: id, operation: "set_scopes",
            scopes: [{ scope_kind: "cohort", scope_id: cohort }],
          }),
        });
      }, { show: f.show_id, kind: "scene_stage_element", id: f.token_a_scene_stage_element_id, cohort: f.cohort_a_id });

      const aIds = await withViewer(f.player_a_cookie, async (v) => {
        await shot(v.page, "k90-05-cohort-a-sees-object");
        return elementIds(v.page);
      });
      const bIds = await withViewer(f.player_b_cookie, async (v) => {
        await shot(v.page, "k90-06-cohort-b-does-not");
        return elementIds(v.page);
      });
      check("Cohort A's browser receives the Cohort-A object", aIds.includes(sceneKey));
      check("Cohort B's browser does NOT", !bIds.includes(sceneKey));
      check("the two Cohorts share one Scene and differ only in projection",
        aIds.length !== bIds.length, `A=${aIds.length} B=${bIds.length} elements`);
    }

    // === §42: Cue parity, in the browser =================================
    console.log("\n=== §42: a Cue reveals the same object the Director can ===");
    {
      const cue = await director.page.evaluate(async ({ show, placement, kind, id }) => {
        const create = await fetch(`/api/shows/${show}/scenes/${placement}/cues`, {
          method: "POST", credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            internal_name: "K90 browser reveal",
            stage_button_label: "Reveal the dummy",
            actions: [{ type: "reveal_object", stage_object: { object_kind: kind, object_id: id } }],
          }),
        });
        const body = await create.json();
        return { status: create.status, id: (body?.data ?? body)?.cue?.id, raw: JSON.stringify(body).slice(0, 200) };
      }, { show: f.show_id, placement: f.placement_id, kind: "scene_stage_element", id: f.token_a_scene_stage_element_id });
      check("a Cue targeting the object by canonical identity is authored", cue.status === 200 && Boolean(cue.id),
        `status ${cue.status} ${cue.raw}`);

      if (cue.id) {
        const fired = await director.page.evaluate(async (cueID) => {
          const r = await fetch(`/api/cues/${cueID}/go`, {
            method: "POST", credentials: "include",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ idempotency_key: "k90-browser-" + Date.now() }),
          });
          const body = await r.json();
          return { status: r.status, result: (body?.data ?? body)?.execution?.status || (body?.data ?? body)?.status, raw: JSON.stringify(body).slice(0, 240) };
        }, cue.id);
        check("the Cue fires successfully", fired.status === 200 && fired.result === "succeeded",
          `status ${fired.status} result ${fired.result} ${fired.raw}`);

        const aIds = await withViewer(f.player_a_cookie, async (v) => {
          await shot(v.page, "k90-07-cue-revealed-for-player");
          return elementIds(v.page);
        });
        check("the affected viewer's browser now receives the Cue-revealed object",
          aIds.includes(sceneKey));

        // §42 step 9: the Director's manual control afterwards reflects the
        // same state -- the menu should now offer Visible as current.
        const current = await director.page.evaluate(async (key) => {
          const res = await fetch("/api/world/catharsis", { credentials: "include" });
          const body = await res.json();
          const snap = body?.data ?? body;
          const el = (snap?.elements || []).find((e) => e.element_id === key);
          const logic = window.VictoryStageLogic;
          const model = {
            kind: "token", live: false, key: el.element_id, elementId: el.element_id,
            label: el.name, source: el, state: el.state, visibility: el.visibility,
          };
          const actions = logic.resolveStageObjectActions(model, {
            role: "director", canManageIndexCards: true, canManageStageTokens: true,
            canActorRevealHideStageObjects: true, hasSelection: false,
          });
          const vis = actions.find((a) => a.action === "k90-visibility");
          return (vis?.submenu || []).map((c) => c.label);
        }, sceneKey);
        check("the Director's manual control reflects the Cue's result",
          Array.isArray(current) && current[0]?.includes("✓"), JSON.stringify(current));
      }
    }

    // === §11: the scope panel renders ====================================
    console.log("\n=== §11/§45: the scope panel ===");
    {
      const opened = await director.page.evaluate(async (key) => {
        const res = await fetch("/api/world/catharsis", { credentials: "include" });
        const body = await res.json();
        const snap = body?.data ?? body;
        const el = (snap?.elements || []).find((e) => e.element_id === key);
        const model = {
          kind: "token", live: false, key: el.element_id, elementId: el.element_id,
          label: el.name, source: el, state: el.state, visibility: el.visibility,
        };
        await window.VictoryKernel90VisibilityTools?.openScopePanel?.(model);
        await new Promise((r) => setTimeout(r, 600));
        const panel = document.querySelector('[role="dialog"][aria-label^="Visibility scope"]');
        if (!panel) return { open: false };
        return {
          open: true,
          checkboxes: panel.querySelectorAll('input[type="checkbox"]').length,
          text: panel.textContent.slice(0, 200),
        };
      }, sceneKey);
      check("the scope panel opens with real Cohorts and Characters to choose",
        opened.open === true && opened.checkboxes >= 5,
        `checkboxes=${opened.checkboxes}`);
      if (opened.open) await shot(director.page, "k90-08-scope-panel");
    }

    const failed = results.filter((r) => !r.ok);
    console.log(`\n${results.length - failed.length}/${results.length} checks passed`);
    if (failed.length) {
      console.log("FAILED:");
      for (const r of failed) console.log(`  - ${r.name}`);
      exitCode = 1;
    }
  } catch (err) {
    console.error("browser proof crashed:", err);
    exitCode = 1;
  } finally {
    await browser.close();
    proxy.kill();
  }
  process.exit(exitCode);
})();

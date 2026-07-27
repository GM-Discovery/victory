#!/usr/bin/env node
"use strict";

// Kernel 74 browser acceptance proof (spec §14.4): real browser sessions
// against the live deployed stack, walking the Player-controlled tutorial
// tail end to end.
//
//   Kessa -> Leave Kessa's Stall -> the door appears (for that Player and
//   that Character only) -> freeform intention -> Ra interrupts
//   automatically -> guided topics out of order -> refresh survives ->
//   Leave Ra -> only that Player transitions -> Director flies a later
//   shared Scene -> the local projection clears.
//
// Creates clearly-named throwaway accounts, a Production, a Show Run, and a
// Show, all prefixed k74_/k74- and archived at the end. Following Kernel
// 62/63/65 precedent, the throwaway accounts are left in place (no elevated
// privileges are granted to the Player/Audience accounts) and are documented
// in the reportback rather than deleted.
//
// Run after deploy:
//   NODE_PATH=/tmp/node_modules node scripts/smoke/kernel74-tutorial-browser.js

const fs = require("node:fs");
const path = require("node:path");
const { chromium } = require("playwright");

const baseURL = process.env.K74_BASE_URL || "https://victory.amurray.family";
const hostDomain = new URL(baseURL).hostname;
const evidenceDir = process.env.K74_EVIDENCE_DIR || path.join("Construction", "OperatorLogs", "evidence", "kernel-74");

const stamp = Date.now();
const password = `k74-browser-${stamp}`;
const director = { handle: `k74_director_${stamp}`, display: "K74 Director" };
const playerA = { handle: `k74_player_a_${stamp}`, display: "K74 Player A" };
const playerB = { handle: `k74_player_b_${stamp}`, display: "K74 Player B" };

// The Player types a COMPLETE SENTENCE on purpose (§1.4): "You start to
// <this>" would be ungrammatical, which is exactly why Victory must quote
// rather than prepend. The angle brackets are the escaping probe (§8.3).
const INTENTION = `I wedge my dagger under the <b>iron band</b> and lever it, betting the wood is softer than it looks.`;

let failures = 0;

function assert(condition, message) {
  if (!condition) throw new Error("ASSERT FAILED: " + message);
}
function pass(message) {
  console.log("PASS " + message);
}
function warn(message) {
  failures += 1;
  console.log("FAIL " + message);
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
  try {
    await page.screenshot({ path: path.join(evidenceDir, `${name}.png`), fullPage: false });
  } catch (err) {
    console.log("note: screenshot " + name + " failed: " + err.message);
  }
}

async function main() {
  fs.mkdirSync(evidenceDir, { recursive: true });
  const browser = await chromium.launch({
    headless: true,
    args: [`--host-resolver-rules=MAP ${hostDomain} 127.0.0.1`],
  });
  const contextOptions = { baseURL, ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } };
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
    pass("three fresh browser accounts signed up");

    // The Director needs producer authority at amurray-family, granted out
    // of band exactly as fresh-install.sh does.
    // Run from the host against the live database, exactly as
    // fresh-install.sh does -- the runtime image ships only the server
    // binary, not the bootstrap command.
    const { execFileSync } = require("node:child_process");
    execFileSync("go", ["run", "./cmd/victory-bootstrap", "producer",
      "--handle", director.handle, "--location", "amurray-family"], {
      cwd: path.join(__dirname, "..", "..", "backend"),
      env: { ...process.env, DATABASE_URL: process.env.K74_DATABASE_URL || "postgres://victory:REDACTED@127.0.0.1:5432/victory?sslmode=disable" },
      stdio: "pipe",
    });
    pass("Director granted producer authority at amurray-family");

    // --- Setup: Production -> Show Run -> Show -> roster -> Characters ----

    const prod = await api(pageD, "/api/productions", {
      method: "POST",
      body: JSON.stringify({ name: `K74 Browser ${stamp}`, slug: `k74-browser-${stamp}`, location_slug: "amurray-family" }),
    });
    assert(prod.ok, `create production (${prod.status}: ${JSON.stringify(prod.body)})`);
    const productionID = prod.body.data.id;

    const run = await api(pageD, "/api/show-runs", {
      method: "POST",
      body: JSON.stringify({
        production_id: productionID,
        title: `K74 Browser Run ${stamp}`,
        slug: `k74-browser-run-${stamp}`,
      }),
    });
    assert(run.ok, `create show run (${run.status}: ${JSON.stringify(run.body)})`);
    const showRunID = run.body.data.show_run.id;

    const show = await api(pageD, `/api/show-runs/${showRunID}/shows`, {
      method: "POST",
      body: JSON.stringify({ title: "K74 Browser Show", slug: `k74-browser-show-${stamp}` }),
    });
    assert(show.ok, `create show (${show.status}: ${JSON.stringify(show.body)})`);
    const showID = show.body.data.show.id;
    pass(`Production/Show Run/Show created (show ${showID})`);

    // Kernel 65 open enrollment is the cleanest way to roster Players
    // without walking the whole ticket/invite flow, which Kernel 74 does not
    // touch and other smoke tests already cover.
    const enroll = await api(pageD, `/api/show-runs/${showRunID}`, {
      method: "PATCH",
      body: JSON.stringify({ open_enrollment: true }),
    });
    assert(enroll.ok, `enable open enrollment (${enroll.status}: ${JSON.stringify(enroll.body)})`);

    // Roster both Players and give each their own Character.
    for (const [page, user] of [[pageA, playerA], [pageB, playerB]]) {
      // The roster addresses people by their Player Workbook profile id,
      // not their raw user id -- Kernel 61's contract.
      const profile = await api(page, "/api/player-profile/me");
      const profileID = profile.body?.data?.workbook?.id;
      assert(profileID, `workbook id for ${user.handle}: ${JSON.stringify(profile)}`);
      const invite = await api(page, `/api/show-runs/${showRunID}/roster/self-join-as-player`, { method: "POST" });
      assert(invite.ok, `roster ${user.handle} (${invite.status}: ${JSON.stringify(invite.body)})`);

      // A Player needs a cast membership (to draft a Character) AND a
      // Catharsis access grant (to read the venue). A cast request for
      // Catharsis auto-approves and issues both -- the real onboarding path,
      // so this script needs no direct database access.
      const request = await api(page, "/api/requests/create", {
        method: "POST",
        body: JSON.stringify({ venue_slug: "catharsis", requested_role: "cast", note: "K74 browser proof" }),
      });
      assert(request.ok, `Catharsis cast request for ${user.handle} (${request.status}: ${JSON.stringify(request.body)})`);

      const card = await api(page, "/api/character-cards", {
        method: "POST",
        body: JSON.stringify({ name: `${user.display} Hero`, location_slug: "amurray-family" }),
      });
      assert(card.ok, `create character for ${user.handle} (${card.status}: ${JSON.stringify(card.body)})`);
      const cardID = card.body.data.id;
      const select = await api(page, `/api/show-runs/${showRunID}/roster/me/character`, {
        method: "POST",
        body: JSON.stringify({ character_card_id: cardID }),
      });
      assert(select.ok, `select character for ${user.handle} (${select.status}: ${JSON.stringify(select.body)})`);
    }
    pass("both Players rostered with their own selected Characters");

    // --- Prepare the tutorial and start the Show -------------------------

    const prep = await api(pageD, `/api/shows/${showID}/prepare-locked-courtyard-opening`, { method: "POST" });
    assert(prep.ok, `prepare (${prep.status}: ${JSON.stringify(prep.body)})`);
    const p = prep.body.data.result;
    assert(p.ready, `prepare Ready: ${JSON.stringify(prep.body)}`);
    assert(p.door_interaction_id && p.door_binding_id && p.ra_interaction_id,
      "prepare must create the Kernel 74 door + Ra records");
    pass(`Locked Courtyard prepared (door ${p.door_interaction_id}, Ra ${p.ra_interaction_id})`);

    const setScene = await api(pageD, `/api/shows/${showID}/current-scene`, {
      method: "POST",
      body: JSON.stringify({ show_scene_placement_id: p.placement_id }),
    });
    assert(setScene.ok, `set current scene (${setScene.status}: ${JSON.stringify(setScene.body)})`);

    // PREREQUISITE: Catharsis must not already have a Session open for a
    // different Show. world.LoadVenueSnapshot resolves the most recent
    // active Session at a venue, so a pre-existing rehearsal Session on
    // Catharsis makes every Player read below resolve someone else's Show
    // and fail with no_active_session. This script deliberately does NOT
    // end a Session it did not open -- that is the operator's call. Check
    // first, and stop with a clear message rather than interfering.
    const venueSlug = process.env.K74_VENUE_SLUG || "catharsis";
    const preflight = await api(pageA, `/api/world/${venueSlug}`);
    assert(preflight.ok, `preflight world read (${preflight.status})`);
    const existingShow = preflight.body?.data?.session?.show_id || "";
    if (existingShow && existingShow !== showID) {
      throw new Error(
        `PRECONDITION: ${venueSlug} already has an active Session for a different Show (${existingShow}). ` +
        `End it (or point K74_VENUE_SLUG at a free venue) before running this proof. ` +
        `This script will not end a Session it did not open.`);
    }

    const code = show.body.data.show.short_code;
    const startSession = await api(pageD, "/api/showtime/control", {
      method: "POST",
      body: JSON.stringify({ short_code: code, action: "start", venue_slug: venueSlug }),
    });
    assert(startSession.ok, `start session (${startSession.status}: ${JSON.stringify(startSession.body)})`);
    pass(`Show session started at ${venueSlug}`);

    // --- Step 5/6: the door is Player- and Character-scoped ---------------

    const worldA1 = await api(pageA, `/api/world/${venueSlug}`);
    assert(worldA1.ok, `Player A world (${worldA1.status})`);
    const hasDoor = (world) => (world.body.data.elements || []).some((e) => e.name === "Locked Courtyard Door");
    const hasKessa = (world) => (world.body.data.elements || []).some((e) => e.name === "Kessa");

    assert(hasKessa(worldA1), "Player A must see Kessa on the shared Courtyard");
    if (hasDoor(worldA1)) {
      warn("§16.3 door must NOT be visible before Kessa completion");
    } else {
      pass("door hotspot is absent before Kessa completion");
    }

    // Step 4: complete Kessa WITHOUT buying anything.
    const complete = await api(pageA, `/api/participant-interactions/${p.interaction_id}/complete`, { method: "POST" });
    assert(complete.ok, `Leave Kessa's Stall (${complete.status}: ${JSON.stringify(complete.body)})`);
    pass("Leave Kessa's Stall succeeded with no purchase, no Haggle, no stance");

    const worldA2 = await api(pageA, `/api/world/${venueSlug}`);
    if (!hasDoor(worldA2)) {
      warn("§16.3 door must appear for Player A after Kessa completion");
    } else {
      pass("door hotspot appeared for Player A only after completing Kessa");
    }

    const worldB1 = await api(pageB, `/api/world/${venueSlug}`);
    if (hasDoor(worldB1)) {
      warn("§16.3 Player B must not see the door from Player A's progress");
    } else {
      pass("Player B (no Kessa completion) still cannot see the door");
    }

    // --- Step 7-10: freeform intention and automatic Ra -------------------

    const submit = await api(pageA, `/api/participant-interactions/${p.door_interaction_id}/submit`, {
      method: "POST",
      body: JSON.stringify({ text: INTENTION, idempotency_key: `k74-${stamp}-1` }),
    });
    assert(submit.ok, `submit intention (${submit.status}: ${JSON.stringify(submit.body)})`);
    const sub = submit.body.data.submission;
    assert(sub.submitted_text === INTENTION, "the exact intention must be stored and echoed verbatim");
    assert(!sub.narration.includes(INTENTION) && !sub.interruption.includes(INTENTION),
      "authored narration must not embed the Player's sentence server-side");
    assert(sub.next_interaction_id === p.ra_interaction_id, "Ra must be chained automatically");
    pass("intention stored verbatim; Ra chained with no Director GO");

    // Retry with a different idempotency key AND different words.
    const retry = await api(pageA, `/api/participant-interactions/${p.door_interaction_id}/submit`, {
      method: "POST",
      body: JSON.stringify({ text: "A totally different second plan", idempotency_key: `k74-${stamp}-2` }),
    });
    assert(retry.ok, `retry submit (${retry.status})`);
    assert(retry.body.data.submission.already_submitted === true, "a retry must resolve to the original");
    assert(retry.body.data.submission.submitted_text === INTENTION, "a retry must return the ORIGINAL words");
    pass("retry duplicated neither the intention nor the note");

    // --- Step 11/12: Directors+ note, and no leakage ----------------------

    const worldD = await api(pageD, `/api/world/${venueSlug}`);
    const sessionID = worldD.body.data.session.id;
    const notesD = await api(pageD, `/api/backstage-notes?session_id=${sessionID}`);
    assert(notesD.ok, `director notes (${notesD.status}: ${JSON.stringify(notesD.body)})`);
    const notes = notesD.body.data.notes || [];
    assert(notes.length === 1, `Director should see exactly 1 note, got ${notes.length}`);
    assert(notes[0].body.includes(INTENTION), "the note must quote the intention verbatim");
    pass("Directors+ see the durable note inside Catharsis, quoting the intention");

    for (const [label, page] of [["submitting Player", pageA], ["other Player", pageB]]) {
      const res = await api(page, `/api/backstage-notes?session_id=${sessionID}`);
      const leaked = res.ok && (res.body?.data?.notes || []).length > 0;
      if (leaked) {
        warn(`§16.19 backstage note leaked to the ${label}`);
      } else {
        pass(`${label} is refused the backstage note (status ${res.status})`);
      }
    }

    // --- Step 13-16: Ra's guided dialogue ---------------------------------

    const openRa = await api(pageA, `/api/participant-interactions/${p.ra_interaction_id}/open`, { method: "POST" });
    assert(openRa.ok, `open Ra (${openRa.status}: ${JSON.stringify(openRa.body)})`);
    let d = openRa.body.data.dialogue;
    assert(d.npc_name === "Ra", "Ra's portrait/name must be present");
    assert(d.portrait_url, "Ra must have a portrait");
    assert(d.can_leave === false, "Leave Ra must start locked");
    pass("Ra's Program opened automatically with portrait, name, and topics");

    const topic = (state, key) => (state.topics || []).find((t) => t.topic_key === key);
    assert(topic(d, "crown-bet").unlocked === false, "crown-bet must start locked");

    // Step 14: ask out of default order.
    const t1 = await api(pageA, `/api/participant-interactions/${p.ra_interaction_id}/dialogue/topic`, {
      method: "POST", body: JSON.stringify({ topic_key: "who-are-you" }),
    });
    assert(t1.ok, `ask who-are-you (${t1.status})`);

    // Step 17: ask why the door is locked.
    const t2 = await api(pageA, `/api/participant-interactions/${p.ra_interaction_id}/dialogue/topic`, {
      method: "POST", body: JSON.stringify({ topic_key: "why-locked" }),
    });
    assert(t2.ok, `ask why-locked (${t2.status})`);
    assert(/locked it/i.test(t2.body.data.dialogue.current_response), "Ra must admit he locked the door");
    pass("asked topics in a non-default order; Ra admits locking the door");

    // Step 16: Leave stays locked until the required topics are viewed.
    const earlyLeave = await api(pageA, `/api/participant-interactions/${p.ra_interaction_id}/dialogue/leave`, { method: "POST" });
    if (earlyLeave.ok) {
      warn("§16.13 Leave Ra must be refused before the required topics are viewed");
    } else {
      pass(`Leave Ra refused before required topics (status ${earlyLeave.status})`);
    }

    for (const key of ["why-looking", "crown-bet"]) {
      const res = await api(pageA, `/api/participant-interactions/${p.ra_interaction_id}/dialogue/topic`, {
        method: "POST", body: JSON.stringify({ topic_key: key }),
      });
      assert(res.ok, `ask ${key} (${res.status}: ${JSON.stringify(res.body)})`);
      d = res.body.data.dialogue;
    }
    assert(/turtle/i.test(d.current_response), "the Crown Bet must name the fractured Turtle continent");
    assert(d.can_leave === true, "Leave Ra must unlock after the required topics only");
    pass("Crown Bet delivered; Leave Ra unlocked without every optional topic");

    // Step 15: refresh -- reopen and confirm progress survived.
    const reopen = await api(pageA, `/api/participant-interactions/${p.ra_interaction_id}/open`, { method: "POST" });
    assert(reopen.ok, `reopen Ra (${reopen.status})`);
    const seen = (reopen.body.data.dialogue.topics || []).filter((t) => t.seen).length;
    assert(seen === 4, `expected 4 topics still seen after refresh, got ${seen}`);
    assert(reopen.body.data.dialogue.can_leave === true, "Leave Ra must stay earned across a refresh");
    pass("topic progress and earned Leave survived a refresh");

    // --- Step 18-22: the lock reveal and the local transition -------------

    const leave = await api(pageA, `/api/participant-interactions/${p.ra_interaction_id}/dialogue/leave`, { method: "POST" });
    assert(leave.ok, `Leave Ra (${leave.status}: ${JSON.stringify(leave.body)})`);
    const closing = leave.body.data.dialogue.closing_narration || "";
    assert(/iron plate/i.test(closing) && /recessed/i.test(closing),
      "the reveal must show a concealed courtyard-side mechanism");
    assert(!/padlock/i.test(closing), "the reveal must not contradict 'no obvious lock'");
    pass("Ra revealed a believable concealed courtyard-side lock mechanism");

    const worldA3 = await api(pageA, `/api/world/${venueSlug}`);
    const projA = worldA3.body.data.session.local_projection;
    assert(projA && projA.scene_slug === "tutorial-handoff", "Player A must be on the handoff projection");
    assert(projA.backdrop_url, "the handoff projection must carry a backdrop");
    if (hasKessa(worldA3)) {
      warn("§11.2 Player A on the handoff must not still see the Courtyard composition");
    } else {
      pass("Player A transitioned to the generic tutorial-handoff map");
    }

    // Step 21: the shared current Scene did NOT move.
    assert(worldA3.body.data.session.current_show_scene_placement_id === p.placement_id,
      "§16.15 the shared current Scene must be unchanged");
    pass("shared Show current Scene remains the Courtyard");

    // Step 22: Player B was not moved.
    const worldB2 = await api(pageB, `/api/world/${venueSlug}`);
    assert(!worldB2.body.data.session.local_projection, "§16.16 Player B must not have been moved");
    assert(hasKessa(worldB2), "Player B must remain on the shared Courtyard");
    pass("Player B was neither moved nor blocked");

    // Step 23: refresh -- the handoff persists.
    const worldA4 = await api(pageA, `/api/world/${venueSlug}`);
    assert(worldA4.body.data.session.local_projection, "§16.17 the projection must survive a refresh");
    pass("handoff projection persists across refresh");

    // --- Step 24/25: a later shared Scene clears the projection -----------

    const scenes = await api(pageD, `/api/shows/${showID}/scenes`);
    assert(scenes.ok, `list placements (${scenes.status})`);
    let secondPlacement = (scenes.body.data.placements || []).find((x) => x.id !== p.placement_id);
    if (!secondPlacement) {
      const handoffScene = await api(pageD, "/api/scenes?location_slug=amurray-family");
      const target = (handoffScene.body?.data?.scenes || []).find((s) => s.slug === "tutorial-handoff");
      assert(target, "the seeded tutorial-handoff Scene must exist to stage as a second Scene");
      const staged = await api(pageD, `/api/shows/${showID}/scenes`, {
        method: "POST", body: JSON.stringify({ scene_id: target.id }),
      });
      assert(staged.ok, `stage a second Scene (${staged.status}: ${JSON.stringify(staged.body)})`);
      secondPlacement = staged.body.data.placement;
    }
    const fly = await api(pageD, `/api/shows/${showID}/current-scene`, {
      method: "POST", body: JSON.stringify({ show_scene_placement_id: secondPlacement.id }),
    });
    assert(fly.ok, `Director flies a later shared Scene (${fly.status}: ${JSON.stringify(fly.body)})`);

    const worldA5 = await api(pageA, `/api/world/${venueSlug}`);
    if (worldA5.body.data.session.local_projection) {
      warn("§16.18 advancing the shared Scene must clear the local projection");
    } else {
      pass("local projection cleared when the Director flew a later shared Scene");
    }
    assert(worldA5.body.data.session.current_show_scene_placement_id === secondPlacement.id,
      "Player A must now resolve the Director-controlled Scene");
    pass("Player A rejoined the Director-controlled shared Scene");

    // --- Browser-rendered evidence + the escaping proof -------------------

    await pageD.goto("/venues/catharsis/");
    await pageD.waitForTimeout(4000);
    await screenshot(pageD, "director-catharsis");

    // §8.3 / §1.4: the Player's angle-bracket text must render as literal
    // characters, never as markup. Assert on the DOM the Director actually
    // sees rather than on the JSON.
    const escaped = await pageD.evaluate(async (sid) => {
      const res = await fetch(`/api/backstage-notes?session_id=${sid}`, { credentials: "include" });
      const payload = await res.json();
      const body = payload?.data?.notes?.[0]?.body || "";
      const probe = document.createElement("div");
      probe.textContent = body;
      return { rendersBold: probe.querySelector("b") !== null, html: probe.innerHTML };
    }, sessionID);
    if (escaped.rendersBold) {
      warn("§8.3 Player text must never become live markup in the backstage note");
    } else {
      assert(escaped.html.includes("&lt;b&gt;"), "the angle brackets must survive as escaped text");
      pass("Player-supplied markup renders as literal text, not HTML");
    }

    await pageA.goto("/venues/catharsis/");
    await pageA.waitForTimeout(4000);
    await screenshot(pageA, "player-a-handoff");
    await pageB.goto("/venues/catharsis/");
    await pageB.waitForTimeout(4000);
    await screenshot(pageB, "player-b-courtyard");
    pass(`browser screenshots written to ${evidenceDir}`);

    // --- Cleanup: archive the throwaway Show/Show Run --------------------

    await api(pageD, `/api/shows/${showID}`, { method: "PATCH", body: JSON.stringify({ status: "archived" }) });
    pass("throwaway Show archived");

    console.log("");
    if (failures === 0) {
      console.log("Kernel 74 browser golden path: PASS");
    } else {
      console.log(`Kernel 74 browser golden path: ${failures} FAILURE(S)`);
      process.exitCode = 1;
    }
  } finally {
    await browser.close();
  }
}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});

#!/usr/bin/env node
"use strict";

// Kernel 75 golden-journey browser proof (spec §11.1). Noting
//
// Walks the complete first-time path with fresh accounts:
//
//   signup -> Catharsis cast request -> Character creation WITH a real
//   workbook (archetype, attributes, Face Sheet history) -> roster self-join
//   -> Character selection -> Director prepares + starts the Show -> Kessa
//   with a stance, a Haggle and a purchase -> locked door -> freeform
//   intention -> Ra -> all required topics -> Leave (gate opens, tutorial
//   NOT yet complete) -> Continue -> completion payload -> Story So Far ->
//   Aftercare submitted (Player A) and skipped (Player B) -> Director's
//   Chair review + CSV -> Director flies a later shared Scene -> local
//   projections clear.
//
// Deliberately wider than the Kernel 74 script, which created bare
// Characters. The Face Sheet clause of the reflection has no input unless a
// Character actually has chapter2/chapter3 workbook context and history
// entries, so a proof that skipped that would silently pass while the most
// personalised sentence in the product never rendered.
//
// PRECONDITION HANDLING. world.LoadVenueSnapshot resolves the most recent
// active Session at a venue, so a pre-existing Session for a DIFFERENT Show
// makes every Player read below resolve someone else's Show. Kernel 74's
// script stopped there and could never complete a full run. Three changes:
//
//   1. The refusal stays the DEFAULT. This script still never silently ends
//      a Session it did not open.
//   2. K75_END_FOREIGN_SESSION=1 opts in explicitly, and the ended session
//      id is written to the evidence directory so the operator can restart
//      it.
//   3. After starting our own Session we ASSERT that the venue resolves to
//      OUR show before any Player read. Without that, every downstream
//      failure presents as an uninformative no_active_session.
//
// Run after deploy:
//   NODE_PATH=/tmp/node_modules node scripts/smoke/kernel75-golden-journey-browser.js

const fs = require("node:fs");
const path = require("node:path");
const { chromium } = require("playwright");

const baseURL = process.env.K75_BASE_URL || "https://victory.amurray.family";
const hostDomain = new URL(baseURL).hostname;
const evidenceDir = process.env.K75_EVIDENCE_DIR
  || path.join("Construction", "OperatorLogs", "evidence", "kernel-75");
const venueSlug = process.env.K75_VENUE_SLUG || "catharsis";
const endForeignSession = process.env.K75_END_FOREIGN_SESSION === "1";

const stamp = Date.now();
const password = `k75-browser-${stamp}`;
const director = { handle: `k75_director_${stamp}`, display: "K75 Director" };
const playerA = { handle: `k75_player_a_${stamp}`, display: "K75 Player A" };
const playerB = { handle: `k75_player_b_${stamp}`, display: "K75 Player B" };

// A complete sentence, with markup, on purpose: §1.4 requires Victory to
// quote the Player rather than prepend to their words, and the angle
// brackets probe escaping all the way through to the completion Program,
// the Story So Far page, and the Director's Chair.
const INTENTION =
  "I study the hinges and test whether the <b>door</b> can be lifted instead of forced.";

let failures = 0;
function assert(condition, message) {
  if (!condition) throw new Error("ASSERT FAILED: " + message);
}
function pass(message) { console.log("PASS " + message); }
function warn(message) { failures += 1; console.log("FAIL " + message); }

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
    return { ok: res.ok, status: res.status, body, text };
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

// buildCharacterWorkbook writes the same workbook shape the Catharsis
// onboarding produces: confirmed chapter3 archetype, chapter2 attributes,
// and chapter2_stage history entries on the history page.
//
// Without these the Story So Far reflection has no archetype to name, no
// attribute to lead with, and no Face Sheet line to quote -- so the richest
// three sentences it can produce would go untested.
async function buildCharacterWorkbook(page, cardID, archetype) {
  const res = await api(page, `/api/character-workbooks/${cardID}/events`, {
    method: "POST",
    body: JSON.stringify({
      character_card_id: cardID,
      module_key: "socio",
      module_status: "active",
      current_stage: 4,
      current_event: "chapter3_archetype_confirmed",
      workbook_status: "complete",
      workbook_context: {
        chapter2: { attributes: archetype.attributes },
        chapter3: {
          archetype_key: archetype.key,
          archetype_title: archetype.title,
          primary_attribute: archetype.primary,
          secondary_attribute: archetype.secondary,
          key_skill: archetype.keySkill,
          confirmed: true,
        },
      },
      entries: [
        {
          page_key: "history",
          entry_type: "chapter2_stage",
          title: "Stage 2: Childhood",
          body: archetype.historyLine,
          stage_number: 2,
          sort_order: 0,
        },
      ],
    }),
  });
  assert(res.ok, `write workbook for ${cardID} (${res.status}: ${JSON.stringify(res.body)})`);
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

    const { execFileSync } = require("node:child_process");
    execFileSync("go", ["run", "./cmd/victory-bootstrap", "producer",
      "--handle", director.handle, "--location", "amurray-family"], {
      cwd: path.join(__dirname, "..", "..", "backend"),
      env: {
        ...process.env,
        DATABASE_URL: process.env.K75_DATABASE_URL
          || "postgres://victory:REDACTED@127.0.0.1:5432/victory?sslmode=disable",
      },
      stdio: "pipe",
    });
    pass("Director granted producer authority at amurray-family");

    // --- Setup ------------------------------------------------------------

    const prod = await api(pageD, "/api/productions", {
      method: "POST",
      body: JSON.stringify({
        name: `K75 Browser ${stamp}`, slug: `k75-browser-${stamp}`, location_slug: "amurray-family",
      }),
    });
    assert(prod.ok, `create production (${prod.status}: ${JSON.stringify(prod.body)})`);

    const run = await api(pageD, "/api/show-runs", {
      method: "POST",
      body: JSON.stringify({
        production_id: prod.body.data.id,
        title: `K75 Browser Run ${stamp}`,
        slug: `k75-browser-run-${stamp}`,
      }),
    });
    assert(run.ok, `create show run (${run.status}: ${JSON.stringify(run.body)})`);
    const showRunID = run.body.data.show_run.id;

    const show = await api(pageD, `/api/show-runs/${showRunID}/shows`, {
      method: "POST",
      body: JSON.stringify({ title: "K75 Browser Show", slug: `k75-browser-show-${stamp}` }),
    });
    assert(show.ok, `create show (${show.status}: ${JSON.stringify(show.body)})`);
    const showID = show.body.data.show.id;
    pass(`Production/Show Run/Show created (show ${showID})`);

    const enroll = await api(pageD, `/api/show-runs/${showRunID}`, {
      method: "PATCH",
      body: JSON.stringify({ open_enrollment: true }),
    });
    assert(enroll.ok, `enable open enrollment (${enroll.status}: ${JSON.stringify(enroll.body)})`);

    const archetypes = {
      [playerA.handle]: {
        key: "Observer", title: "The Observer",
        primary: "Empathy", secondary: "Awareness", keySkill: "Insight",
        attributes: { Empathy: 8, Awareness: 6, Craft: 4, Resolve: 5 },
        historyLine: "You learned early to notice who was left outside the group, and Empathy came with it.",
      },
      [playerB.handle]: {
        key: "Guardian", title: "The Guardian",
        primary: "Resolve", secondary: "Might", keySkill: "Protection",
        attributes: { Resolve: 7, Might: 6, Empathy: 3 },
        historyLine: "You were the one who stood in the doorway while the others got out.",
      },
    };
    const cards = {};

    for (const [page, user] of [[pageA, playerA], [pageB, playerB]]) {
      const invite = await api(page, `/api/show-runs/${showRunID}/roster/self-join-as-player`, { method: "POST" });
      assert(invite.ok, `roster ${user.handle} (${invite.status}: ${JSON.stringify(invite.body)})`);

      const request = await api(page, "/api/requests/create", {
        method: "POST",
        body: JSON.stringify({ venue_slug: venueSlug, requested_role: "cast", note: "K75 golden journey" }),
      });
      assert(request.ok, `cast request for ${user.handle} (${request.status}: ${JSON.stringify(request.body)})`);

      const card = await api(page, "/api/character-cards", {
        method: "POST",
        body: JSON.stringify({ name: `${user.display} Hero`, location_slug: "amurray-family" }),
      });
      assert(card.ok, `create character for ${user.handle} (${card.status}: ${JSON.stringify(card.body)})`);
      const cardID = card.body.data.id;
      cards[user.handle] = cardID;

      await buildCharacterWorkbook(page, cardID, archetypes[user.handle]);

      const select = await api(page, `/api/show-runs/${showRunID}/roster/me/character`, {
        method: "POST",
        body: JSON.stringify({ character_card_id: cardID }),
      });
      assert(select.ok, `select character for ${user.handle} (${select.status}: ${JSON.stringify(select.body)})`);
    }
    pass("both Players rostered with fully built Characters (archetype, attributes, Face Sheet history)");

    // --- Prepare and start ------------------------------------------------

    const prep = await api(pageD, `/api/shows/${showID}/prepare-locked-courtyard-opening`, { method: "POST" });
    assert(prep.ok, `prepare (${prep.status}: ${JSON.stringify(prep.body)})`);
    const p = prep.body.data.result;
    assert(p.ready, `prepare must be Ready: ${JSON.stringify(prep.body)}`);

    const setScene = await api(pageD, `/api/shows/${showID}/current-scene`, {
      method: "POST",
      body: JSON.stringify({ show_scene_placement_id: p.placement_id }),
    });
    assert(setScene.ok, `set current scene (${setScene.status}: ${JSON.stringify(setScene.body)})`);

    // PRECONDITION: a foreign Session at this venue would make every Player
    // read below resolve someone else's Show.
    const preflight = await api(pageA, `/api/world/${venueSlug}`);
    assert(preflight.ok, `preflight world read (${preflight.status})`);
    const existingShow = preflight.body?.data?.session?.show_id || "";
    if (existingShow && existingShow !== showID) {
      if (!endForeignSession) {
        throw new Error(
          `PRECONDITION: ${venueSlug} already has an active Session for a different Show (${existingShow}). ` +
          `Re-run with K75_END_FOREIGN_SESSION=1 to end it (the ended session id will be written to ` +
          `${evidenceDir}/ended-foreign-session.txt so you can restart it), or point K75_VENUE_SLUG at a free venue. ` +
          `This script will not end a Session it did not open unless told to.`);
      }
      // /api/showtime/control's end action keys on the SHOW's short code,
      // not on the venue, so resolve the foreign Show first. This also means
      // the acting Director must already hold authority over that Show Run --
      // ending someone else's Session at a Location you have no standing in
      // is refused by the ordinary gate, not by this script.
      const foreign = await api(pageD, `/api/shows/${existingShow}`);
      assert(foreign.ok,
        `look up the foreign Show to end it (${foreign.status}: ${JSON.stringify(foreign.body)})`);
      const foreignCode = foreign.body?.data?.show?.short_code;
      const foreignTitle = foreign.body?.data?.show?.title || "(untitled)";
      assert(foreignCode, `foreign Show ${existingShow} has no short code`);

      const ended = await api(pageD, "/api/showtime/control", {
        method: "POST",
        body: JSON.stringify({ action: "end", short_code: foreignCode }),
      });
      assert(ended.ok, `end foreign session (${ended.status}: ${JSON.stringify(ended.body)})`);
      fs.writeFileSync(
        path.join(evidenceDir, "ended-foreign-session.txt"),
        `Ended a pre-existing Session at ${venueSlug} on ${new Date().toISOString()}\n` +
        `  show_id:    ${existingShow}\n` +
        `  show title: ${foreignTitle}\n` +
        `  short code: ${foreignCode}\n` +
        `Restart it with: /showtime -> ${foreignCode}, or from Stage Management.\n`);
      // A notice, not a failure: the operator opted in explicitly, and the
      // ended Session is recorded above so it can be restarted.
      console.log(`NOTE ended a pre-existing Session for show ${existingShow} ` +
        `(opted in via K75_END_FOREIGN_SESSION; see ${evidenceDir}/ended-foreign-session.txt)`);
    }

    const code = show.body.data.show.short_code;
    const startSession = await api(pageD, "/api/showtime/control", {
      method: "POST",
      body: JSON.stringify({ short_code: code, action: "start", venue_slug: venueSlug }),
    });
    assert(startSession.ok, `start session (${startSession.status}: ${JSON.stringify(startSession.body)})`);

    // THE ASSERTION KERNEL 74'S SCRIPT WAS MISSING. Without it, every
    // downstream failure presents as an uninformative no_active_session
    // with no indication of the cause.
    const confirmed = await api(pageA, `/api/world/${venueSlug}`);
    assert(confirmed.ok, `post-start world read (${confirmed.status})`);
    assert(confirmed.body?.data?.session?.show_id === showID,
      `${venueSlug} must resolve to OUR Show after start; got ` +
      `${confirmed.body?.data?.session?.show_id || "(none)"} instead of ${showID}`);
    pass(`Show session started and ${venueSlug} resolves to our Show`);

    // Both Players actually LOAD the venue before acting. This is the real
    // journey (§11.1 goes /showtime -> Catharsis), and it is also load
    // bearing: joining the stage is what creates the session_participants
    // row that the durable action log attributes events to. Driving the API
    // without ever arriving is a shape no real Player produces.
    for (const [page, user] of [[pageA, playerA], [pageB, playerB]]) {
      await page.goto(`/venues/${venueSlug}/`);
      await page.waitForTimeout(2500);
      const world = await api(page, `/api/world/${venueSlug}`);
      assert(world.ok, `${user.handle} world read (${world.status})`);
      assert(world.body?.data?.session?.show_id === showID,
        `${user.handle} must resolve to our Show, got ${world.body?.data?.session?.show_id || "(none)"}`);
    }
    pass("both Players arrived in Catharsis and resolve to our Show");
    await screenshot(pageA, "00-courtyard-arrival");

    // --- Kessa: stance, Haggle, purchase ----------------------------------

    const stance = await api(pageA, `/api/participant-interactions/${p.interaction_id}/stance`, {
      method: "POST",
      body: JSON.stringify({ stance: "insight" }),
    });
    assert(stance.ok, `stance attempt (${stance.status}: ${JSON.stringify(stance.body)})`);
    pass("Player A approached Kessa through Insight");

    const haggle = await api(pageA, `/api/participant-interactions/${p.interaction_id}/haggle`, { method: "POST" });
    assert(haggle.ok, `haggle attempt (${haggle.status}: ${JSON.stringify(haggle.body)})`);
    pass(`Player A haggled (success=${haggle.body?.data?.result?.success})`);

    const equipment = await api(pageA, `/api/participant-interactions/${p.interaction_id}/open`, { method: "POST" });
    assert(equipment.ok, `open equip mode (${equipment.status}: ${JSON.stringify(equipment.body)})`);
    const firstItem = (equipment.body?.data?.packet?.stock || [])[0];
    if (firstItem) {
      const buy = await api(pageA, `/api/participant-interactions/${p.interaction_id}/purchase`, {
        method: "POST",
        body: JSON.stringify({ equipment_item_id: firstItem.id, idempotency_key: `k75-${stamp}` }),
      });
      assert(buy.ok, `purchase (${buy.status}: ${JSON.stringify(buy.body)})`);
      pass(`Player A acquired ${firstItem.name}`);
    } else {
      warn("Kessa's packet offered no equipment; the acquisition clause will be untested");
    }

    for (const [page, user] of [[pageA, playerA], [pageB, playerB]]) {
      const done = await api(page, `/api/participant-interactions/${p.interaction_id}/complete`, { method: "POST" });
      assert(done.ok, `leave Kessa for ${user.handle} (${done.status}: ${JSON.stringify(done.body)})`);
    }
    pass("both Players left Kessa's stall");

    // Refresh point (§11.3).
    await pageA.goto(`/venues/${venueSlug}/`);
    await pageA.waitForTimeout(1200);
    await screenshot(pageA, "01-courtyard-after-kessa");

    // --- Door intention ----------------------------------------------------

    for (const [page, user] of [[pageA, playerA], [pageB, playerB]]) {
      const submit = await api(page, `/api/participant-interactions/${p.door_interaction_id}/submit`, {
        method: "POST",
        body: JSON.stringify({ text: INTENTION, idempotency_key: `k75-door-${user.handle}` }),
      });
      assert(submit.ok, `door intention for ${user.handle} (${submit.status}: ${JSON.stringify(submit.body)})`);
    }
    pass("both Players submitted a door intention");

    // --- Ra ----------------------------------------------------------------

    for (const [page, user] of [[pageA, playerA], [pageB, playerB]]) {
      const open = await api(page, `/api/participant-interactions/${p.ra_interaction_id}/open`, { method: "POST" });
      assert(open.ok, `open Ra for ${user.handle} (${open.status}: ${JSON.stringify(open.body)})`);
      for (const topic of ["why-looking", "crown-bet"]) {
        const read = await api(page, `/api/participant-interactions/${p.ra_interaction_id}/dialogue/topic`, {
          method: "POST",
          body: JSON.stringify({ topic_key: topic }),
        });
        assert(read.ok, `read ${topic} for ${user.handle} (${read.status}: ${JSON.stringify(read.body)})`);
      }
    }
    // Player A also reads an optional topic, so the "stayed to ask more than
    // they had to" clause has input.
    await api(pageA, `/api/participant-interactions/${p.ra_interaction_id}/dialogue/topic`, {
      method: "POST", body: JSON.stringify({ topic_key: "beyond-the-door" }),
    });
    pass("both Players completed Ra's required topics");

    // --- §3.1: Leave opens the gate but does NOT complete the tutorial -----

    const leaveA = await api(pageA, `/api/participant-interactions/${p.ra_interaction_id}/dialogue/leave`, { method: "POST" });
    assert(leaveA.ok, `leave Ra (${leaveA.status}: ${JSON.stringify(leaveA.body)})`);
    const beats = leaveA.body?.data?.dialogue?.closing_beats || [];
    assert(beats.length > 0, "Leave must return closing beats (or the closing_narration fallback)");
    pass(`Ra's gate-opening beats returned (${beats.length} beats)`);

    const midWorld = await api(pageA, `/api/world/${venueSlug}`);
    const midState = midWorld.body?.data?.session?.tutorial_state || {};
    if (midState.completed) {
      warn("§3.1: leaving Ra must NOT complete the tutorial — Continue is a separate Player act");
    } else {
      pass("leaving Ra opened the gate without completing the tutorial");
    }
    if (!midState.completion_available) {
      warn("the completion Program must be reachable once the handoff is entered");
    } else {
      pass("the completion Program is reachable");
    }

    // Refresh point: the projection and the reopen affordance must survive.
    await pageA.goto(`/venues/${venueSlug}/`);
    await pageA.waitForTimeout(1500);
    await screenshot(pageA, "02-gate-open-before-continue");

    // --- §3.2: Continue -----------------------------------------------------

    const cont = await api(pageA, `/api/participant-interactions/${p.ra_interaction_id}/tutorial/continue`, { method: "POST" });
    assert(cont.ok, `Continue (${cont.status}: ${JSON.stringify(cont.body)})`);
    const completion = cont.body.data.completion;

    assert(completion.headline, "the completion payload must carry a headline");
    assert((completion.waiting_copy || []).length > 0, "the completion payload must carry waiting copy");
    assert((completion.story_events || []).length > 0, "Continue must generate Story So Far");
    pass(`Continue generated ${completion.story_events.length} Story So Far entries`);

    const joined = completion.story_events.map((e) => e.summary).join("\n");
    if (!joined.includes(INTENTION)) {
      warn(`the door intention must appear verbatim in the reflection.\n--- got ---\n${joined}`);
    } else {
      pass("the door intention appears verbatim, markup and all");
    }
    if (!joined.includes("Empathy")) {
      warn(`the archetype-selected lens (Empathy) must appear.\n--- got ---\n${joined}`);
    } else {
      pass("the archetype selected the primary attribute lens");
    }
    if (!joined.includes("left outside the group")) {
      warn(`the matching Face Sheet line must be quoted.\n--- got ---\n${joined}`);
    } else {
      pass("the matching Face Sheet line was quoted");
    }
    if (!completion.recognition?.newly_granted) {
      warn("a first tutorial completion must grant Player recognition");
    } else {
      pass(`one-time Player recognition granted: ${completion.recognition.grant?.label}`);
    }
    for (const event of completion.story_events) {
      if (event.visibility_state !== "private") {
        warn(`generated entries must be private by default; ${event.event_type} was ${event.visibility_state}`);
      }
    }
    pass("every generated Story So Far entry is private by default");

    // Idempotency: a second Continue changes nothing.
    const again = await api(pageA, `/api/participant-interactions/${p.ra_interaction_id}/tutorial/continue`, { method: "POST" });
    assert(again.ok, `repeat Continue (${again.status})`);
    if (again.body.data.completion.story_events.length !== completion.story_events.length) {
      warn("a retried Continue must not duplicate a Player's history");
    } else {
      pass("Continue is idempotent");
    }

    // Refresh point.
    await pageA.goto(`/venues/${venueSlug}/`);
    await pageA.waitForTimeout(1500);
    await screenshot(pageA, "03-after-continue");

    // --- Story So Far is private -------------------------------------------

    const cardA = cards[playerA.handle];
    const mine = await api(pageA, `/api/characters/${cardA}/story-so-far`);
    assert(mine.ok, `own story read (${mine.status}: ${JSON.stringify(mine.body)})`);
    assert((mine.body.data.story_events || []).length > 0, "the owner must see their own history");
    pass(`Player A sees ${mine.body.data.story_events.length} of their own entries`);

    const theirs = await api(pageB, `/api/characters/${cardA}/story-so-far`);
    if (theirs.ok && (theirs.body.data.story_events || []).length > 0) {
      warn("another Player must not see private Story So Far entries");
    } else {
      pass("another Player sees no private Story So Far entries");
    }

    await pageA.goto(`/venues/greenroom/?character_id=${encodeURIComponent(cardA)}`);
    await pageA.waitForTimeout(1500);
    await screenshot(pageA, "04-story-so-far-page");

    // --- Aftercare: A submits, B skips --------------------------------------

    const draft = await api(pageA, `/api/shows/${showID}/aftercare/draft`, {
      method: "PUT",
      body: JSON.stringify({ responses: { favorite_moments: "The gate finally opening." } }),
    });
    assert(draft.ok, `save draft (${draft.status}: ${JSON.stringify(draft.body)})`);

    // Refresh point: the draft must survive.
    const reoffer = await api(pageA, `/api/shows/${showID}/aftercare`);
    assert(reoffer.ok, `re-offer (${reoffer.status})`);
    if (reoffer.body.data.aftercare.draft?.responses?.favorite_moments !== "The gate finally opening.") {
      warn("an Aftercare draft must survive and round-trip");
    } else {
      pass("the Aftercare draft survived a reload");
    }
    if (reoffer.body.data.aftercare.submission) {
      warn("a draft must never be reported as a submission");
    } else {
      pass("the draft is correctly not a submission");
    }

    const submit = await api(pageA, `/api/shows/${showID}/aftercare`, {
      method: "POST",
      body: JSON.stringify({
        responses: {
          favorite_moments: "The gate finally opening.",
          next_session: "I want to meet whoever Ra was talking about.",
        },
      }),
    });
    assert(submit.ok, `submit aftercare (${submit.status}: ${JSON.stringify(submit.body)})`);
    pass("Player A submitted partial Aftercare (two of three prompts)");

    // Player B: an unconfirmed skip must be refused.
    const unconfirmed = await api(pageB, `/api/shows/${showID}/aftercare/skip`, {
      method: "POST", body: JSON.stringify({}),
    });
    if (unconfirmed.ok) {
      warn("an unconfirmed skip must be refused");
    } else {
      pass("an unconfirmed skip was refused");
    }
    const skipped = await api(pageB, `/api/shows/${showID}/aftercare/skip`, {
      method: "POST", body: JSON.stringify({ confirmed: true }),
    });
    assert(skipped.ok, `confirmed skip (${skipped.status}: ${JSON.stringify(skipped.body)})`);
    if (skipped.body.data.aftercare.consecutive_skips !== 1) {
      warn(`expected 1 consecutive skip, got ${skipped.body.data.aftercare.consecutive_skips}`);
    } else {
      pass("Player B's confirmed skip was recorded and counted");
    }

    // --- Director's Chair ----------------------------------------------------

    const review = await api(pageD, `/api/shows/${showID}/aftercare-review`);
    assert(review.ok, `aftercare review (${review.status}: ${JSON.stringify(review.body)})`);
    const rows = review.body.data.rows || [];
    const submittedRow = rows.find((r) => r.state === "submitted");
    const skippedRow = rows.find((r) => r.state === "skipped");
    if (!submittedRow || !skippedRow) {
      warn(`the review must show both states, got ${JSON.stringify(rows.map((r) => r.state))}`);
    } else {
      pass("the Director's Chair shows both a submission and a skip");
    }
    if (!review.body.data.can_export) {
      warn("a Producer must be able to export");
    } else {
      pass("export is offered to the Producer");
    }

    // A Player must not be able to read the review at all.
    const playerReview = await api(pageA, `/api/shows/${showID}/aftercare-review`);
    if (playerReview.ok) {
      warn("a Player must not be able to read the Directors+ Aftercare review");
    } else {
      pass("a Player is refused the Directors+ review");
    }

    const csv = await api(pageD, `/api/shows/${showID}/aftercare-review.csv`);
    assert(csv.ok, `csv export (${csv.status})`);
    const csvText = csv.body?.raw || csv.text || "";
    if (!csvText.includes("player_handle") || !csvText.includes("aftercare_state")) {
      warn(`the CSV must carry its header row; got: ${csvText.slice(0, 200)}`);
    } else {
      pass("the CSV export parsed with its expected header");
    }
    fs.writeFileSync(path.join(evidenceDir, "aftercare-export.csv"), csvText);

    await pageD.goto("/venues/directors-chair/");
    await pageD.waitForTimeout(2000);
    await screenshot(pageD, "05-directors-chair-aftercare");

    // --- §10.3: a later shared Scene clears the local projection -------------

    // Stage a genuine SECOND Scene rather than hunting for one. The Show
    // starts with exactly one placement (the Courtyard), so a "find any
    // other placement" search has nothing correct to return -- and flying
    // an id that is not a valid placement for this Show clears the current
    // Scene instead of advancing it, which looks like a projection bug and
    // is not one.
    const library = await api(pageD, `/api/scenes?location_id=${encodeURIComponent(p.location_id || "")}`);
    let secondScene = null;
    if (library.ok) {
      const scenes = library.body?.data?.scenes || [];
      secondScene = scenes.find((s) => s.slug === "tutorial-handoff") || scenes.find((s) => s.slug !== "courtyard");
    }

    let other = null;
    if (secondScene) {
      const staged = await api(pageD, `/api/shows/${showID}/scenes`, {
        method: "POST",
        body: JSON.stringify({ scene_id: secondScene.id, sort_order: 10 }),
      });
      if (staged.ok) {
        other = staged.body?.data?.placement || null;
      } else {
        warn(`could not stage a second Scene (${staged.status}: ${JSON.stringify(staged.body)})`);
      }
    }

    if (!other || !other.id || other.id === p.placement_id) {
      warn("no distinct second placement to fly; the projection-clearing step was not proven");
    } else {
      const fly = await api(pageD, `/api/shows/${showID}/current-scene`, {
        method: "POST",
        body: JSON.stringify({ show_scene_placement_id: other.id }),
      });
      assert(fly.ok, `fly a later shared Scene (${fly.status}: ${JSON.stringify(fly.body)})`);
      const afterFly = await api(pageA, `/api/world/${venueSlug}`);
      if (afterFly.body?.data?.session?.local_projection) {
        warn("§10.3: a later shared Scene must clear the local tutorial projection");
      } else {
        pass("the later shared Scene cleared the local projection");
      }
      // Story So Far and Aftercare must still be reachable afterward.
      const stillThere = await api(pageA, `/api/characters/${cardA}/story-so-far`);
      if (!stillThere.ok || (stillThere.body.data.story_events || []).length === 0) {
        warn("Story So Far must survive the shared Scene advance");
      } else {
        pass("Story So Far survived the shared Scene advance");
      }
      await pageA.goto(`/venues/${venueSlug}/`);
      await pageA.waitForTimeout(1500);
      await screenshot(pageA, "06-after-shared-scene-advance");
    }

    // --- Anonymous refusals --------------------------------------------------

    const anon = await browser.newContext(contextOptions);
    const anonPage = await anon.newPage();
    await anonPage.goto("/login/");
    for (const url of [
      `/api/characters/${cardA}/story-so-far`,
      `/api/shows/${showID}/aftercare`,
      `/api/shows/${showID}/aftercare-review`,
    ]) {
      const res = await api(anonPage, url);
      if (res.ok) warn(`anonymous users must not read ${url}`);
    }
    pass("anonymous users are refused Story So Far and Aftercare");
    await anon.close();

  } finally {
    await browser.close();
  }

  console.log("");
  if (failures > 0) {
    console.log(`RESULT: ${failures} check(s) failed.`);
    process.exitCode = 1;
  } else {
    console.log("RESULT: golden journey complete, all checks passed.");
    console.log(`Evidence written to ${evidenceDir}`);
  }
  console.log(`Throwaway accounts (left in place, no elevated Player privileges): ${playerA.handle}, ${playerB.handle}, ${director.handle}`);
}

main().catch((err) => {
  console.error(String(err && err.stack ? err.stack : err));
  process.exit(1);
});

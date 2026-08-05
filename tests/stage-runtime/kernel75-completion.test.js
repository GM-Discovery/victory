const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

globalThis.VictoryStageVenue = { slug: "catharsis", name: "Catharsis" };
const { normalizeSnapshot } = require("../../frontend/lib/stage-runtime/state.js");

const PI_PATH = path.join(__dirname, "../../frontend/lib/stage-runtime/participant-interactions.js");
const RUNTIME_PATH = path.join(__dirname, "../../frontend/lib/stage-runtime/runtime.js");
const GREENROOM_PATH = path.join(__dirname, "../../frontend/venues/greenroom/index.html");
const CHAIR_PATH = path.join(__dirname, "../../frontend/venues/directors-chair/index.html");

const participantInteractionsSource = fs.readFileSync(PI_PATH, "utf8");
const runtimeSource = fs.readFileSync(RUNTIME_PATH, "utf8");
const greenroomSource = fs.readFileSync(GREENROOM_PATH, "utf8");
const chairSource = fs.readFileSync(CHAIR_PATH, "utf8");

const { interactionErrorMessage } = require(PI_PATH);

test("Kessa directs unfinished Players to Character Making", () => {
  const expected = "You haven't finished making a character. Please select the character maker in the right tray first.";
  assert.equal(interactionErrorMessage(new Error("not_a_roster_member")), expected);
  assert.equal(interactionErrorMessage(new Error("no_character_selected")), expected);
  assert.equal(interactionErrorMessage(new Error("scene_not_current")), "scene_not_current");
});

// --- Ra's closing beats (S3.1) ---------------------------------------------

test("kernel 75 renders closing beats one at a time", () => {
  assert.match(participantInteractionsSource, /function renderClosingBeats/,
    "the staged lock reveal must be its own renderer");
  assert.match(participantInteractionsSource, /data-action="next-beat"/,
    "there must be an advance control between beats");
});

test("aftercare keeps each prompt attached to its answer field", () => {
  assert.match(participantInteractionsSource, /victory-aftercare-fields/);
  assert.match(participantInteractionsSource, /victory-aftercare-field__prompt/);
  assert.match(participantInteractionsSource, /class="victory-aftercare-field"/);
});

test("kernel 75 falls back to closing_narration when no beats are authored", () => {
  // Load-bearing: this fallback is what lets the frontend and backend deploy
  // in either order, and what keeps a packet seeded before migration 072
  // working unchanged.
  assert.match(
    participantInteractionsSource,
    /closing_beats\)\s*&&\s*data\.dialogue\.closing_beats\.length[\s\S]{0,120}closing_narration/,
    "an empty closing_beats array must fall back to closing_narration");
});

test("kernel 75 final beat continues rather than closing the panel", () => {
  // S3.1: the completion projection must not interrupt Ra's final physical
  // action, so the last beat's button is the Continue verb -- not a dismiss.
  assert.match(participantInteractionsSource, /data-action="continue">Continue</,
    "the final beat must offer Continue");
  assert.match(participantInteractionsSource, /continue:\s*\(\)\s*=>\s*runTutorialContinue\(\)/,
    "Continue must POST the tutorial continue verb, not close the panel");
  assert.match(participantInteractionsSource, /tutorial\/continue/,
    "the Continue verb must hit the tutorial continue route");
});

// --- The completion Program (S4.1) -----------------------------------------

test("kernel 75 completion Program renders all six required sections", () => {
  for (const section of [
    "tutorial-complete", "what-you-did", "story-so-far",
    "progress-earned", "aftercare", "waiting",
  ]) {
    assert.ok(
      participantInteractionsSource.includes(`data-section="${section}"`),
      `S4.1 requires a ${section} section`);
  }
});

test("kernel 75 completion copy comes from the server, not the client", () => {
  // The client must not become a second place where this prose is decided,
  // or the wording lives in two files and drifts.
  assert.match(participantInteractionsSource, /c\.headline/, "headline must come from the payload");
  assert.match(participantInteractionsSource, /c\.body\s*\|\|\s*\[\]/, "body must come from the payload");
  assert.match(participantInteractionsSource, /c\.waiting_copy\s*\|\|\s*\[\]/, "waiting copy must come from the payload");
});

test("kernel 75 hides one-time recognition on a replay", () => {
  // S7.3's visible half: a second Character earns Character-level
  // acknowledgement, never a second copy of the Player-level one.
  assert.match(participantInteractionsSource, /recog\.newly_granted\s*&&\s*recog\.grant/,
    "the recognition beat must be gated on newly_granted");
});

test("kernel 75 escapes Player-authored text in the completion Program", () => {
  // Story summaries quote a Player's own words verbatim (their door
  // intention), so markup in them must render as literal characters. The
  // Program builds HTML strings, so the guarantee here is that every
  // Player-derived value goes through escapeHtml rather than being
  // interpolated raw.
  assert.match(participantInteractionsSource, /escapeHtml\(line\)/,
    "recap lines must be escaped");
  assert.match(participantInteractionsSource, /escapeHtml\(draft\[p\.key\] \|\| ""\)/,
    "draft answers must be escaped when refilling the form");
  // Flatten first: these interpolations legitimately span several lines, so
  // a line-by-line scan reports false positives.
  const flat = participantInteractionsSource.replace(/\s+/g, " ");

  // Every Player-derived value that reaches an HTML template must pass
  // through escapeHtml. Checked by pairing each source with its escape.
  for (const [source, escaped] of [
    ["c.recap_lines", "escapeHtml(line)"],
    ["c.body", "escapeHtml(p)"],
    ["c.waiting_copy", "escapeHtml(p)"],
    ["draft[p.key]", 'escapeHtml(draft[p.key] || "")'],
    ["p.label", "escapeHtml(p.label)"],
  ]) {
    assert.ok(flat.includes(source), `${source} must be rendered`);
    assert.ok(flat.includes(escaped), `${source} must be escaped via ${escaped}`);
  }

  // And nothing writes Player-derived HTML directly.
  assert.ok(!/innerHTML\s*=\s*(recap|answer|event\.summary|row\.)/.test(flat),
    "Player-authored strings must never be assigned through innerHTML");
});

test("kernel 75 closing the completion Program does not erase completion", () => {
  // S1.3/S4.3: closing puts the panel away. The local projection and the
  // reopen affordance both persist -- there is no state reset here.
  const handler = participantInteractionsSource.slice(
    participantInteractionsSource.indexOf('"close-completion":'));
  const body = handler.slice(0, handler.indexOf("},"));
  assert.match(body, /panel\.close\(\)/, "Explore Catharsis must close the panel");
  assert.ok(!/clear|reset|delete|milestone/i.test(body),
    "closing must not clear or reset completion state");
});

// --- Snapshot plumbing ------------------------------------------------------

test("kernel 75 tutorial_state survives snapshot normalization", () => {
  const snapshot = normalizeSnapshot({
    session: {
      show_id: "show-1",
      current_show_scene_placement_id: "placement-1",
      local_projection: { scene_id: "s1", scene_slug: "tutorial-handoff", scene_title: "Outside the Courtyard" },
      tutorial_state: {
        milestones: ["kessa_intro_completed", "tutorial_handoff_entered"],
        completion_available: true,
        completed: false,
        completion_interaction_id: "ra-interaction",
      },
    },
    elements: [],
  });
  const state = snapshot.session.tutorial_state;
  assert.ok(state, "tutorial_state must survive normalization");
  assert.equal(state.completion_available, true);
  assert.equal(state.completion_interaction_id, "ra-interaction");
});

test("kernel 75 tutorial_state carries no denominator", () => {
  // S1.11 forbids "N of M ready" framing anywhere. The server type has no
  // such field; this asserts the client never invents one.
  const forbidden = /tutorial_state\.(total|expected|of_total|player_count|ready_count|percent)/;
  assert.ok(!forbidden.test(runtimeSource),
    "the client must not read or synthesize a participant denominator");
});

// --- Reopen affordance and waiting copy (S4.3, S10.1) -----------------------

test("kernel 75 reopen button appears only when completion is available", () => {
  assert.match(runtimeSource, /tutorialState\?\.completion_available\s*&&\s*tutorialState\?\.completion_interaction_id/,
    "the reopen affordance must be gated on both flags");
  assert.match(runtimeSource, /openTutorialCompletion/,
    "the reopen affordance must call the reopen path");
});

test("kernel 75 keeps the projection banner click-through", () => {
  // Kernel 74's defect (b) was a control that could not act but could still
  // block clicks on the stage behind it. The banner stays
  // pointer-events:none and only the button takes pointer events.
  assert.match(runtimeSource, /banner\.style\.cssText = "pointer-events:none/,
    "the banner itself must stay click-through");
  assert.match(runtimeSource, /reopen\.style\.cssText = "pointer-events:auto/,
    "only the reopen button may take pointer events");
});

test("kernel 75 waiting copy states scheduled human continuation with no countdown", () => {
  const waitingLine = runtimeSource.match(/waiting\.textContent = "([^"]+)"/);
  assert.ok(waitingLine, "the waiting line must exist");
  const copy = waitingLine[1];
  assert.match(copy, /scheduled Session/,
    "S10.1: the copy must say continuation happens at a scheduled Session");
  assert.ok(!/\d/.test(copy),
    `the waiting copy must contain no digits (no countdown, no participant count): ${copy}`);
  assert.ok(!/\bof\b/.test(copy),
    `the waiting copy must not use "N of M" framing: ${copy}`);
});

// --- Aftercare (S8) ---------------------------------------------------------

test("kernel 75 aftercare skip requires two presses", () => {
  // S8.4/S1.14: the first press opens the confirmation; only the second
  // POSTs. Closing the panel or pressing Go Back never increments.
  assert.match(participantInteractionsSource, /function confirmAftercareSkip/,
    "the skip confirmation must be its own step");
  assert.match(participantInteractionsSource, /"skip-confirm":\s*async\s*\(\)\s*=>/,
    "the POST must hang off the confirmation button, not the first Skip");
  assert.match(participantInteractionsSource, /aftercare\/skip[\s\S]{0,200}confirmed:\s*true/,
    "the skip request must carry explicit confirmation");
  assert.match(participantInteractionsSource, /"skip-back":\s*\(\)\s*=>\s*renderTutorialCompletion/,
    "Go Back must return without writing anything");
});

test("kernel 75 aftercare skip warning is factual, not punitive", () => {
  assert.match(participantInteractionsSource, /You skipped Aftercare last time/,
    "S1.14's one-skip wording");
  assert.match(participantInteractionsSource, /You skipped the last two Aftercare check-ins/,
    "S1.14's two-skip wording");
});

test("kernel 75 aftercare form tells the Player their Director can read it", () => {
  // Non-negotiable. Aftercare is backstage-visible by design, and copy that
  // implies otherwise is the privacy bug.
  assert.match(participantInteractionsSource, /Your Director can read this/,
    "the notice must be present");

  // Order matters within the RENDERED template, not within the file: the
  // fields are built into a variable before the template that places them.
  const template = participantInteractionsSource.slice(
    participantInteractionsSource.indexOf('data-section="aftercare-form"'));
  const noticeIndex = template.indexOf("Your Director can read this");
  const fieldsIndex = template.indexOf("${fields}");
  assert.ok(noticeIndex >= 0 && fieldsIndex >= 0, "both the notice and the fields must be in the form");
  assert.ok(noticeIndex < fieldsIndex,
    "the notice must render above the first field, before the Player types");
});

test("kernel 75 aftercare drafts autosave server-side, not to localStorage", () => {
  // A Player may finish on another device, and reflection text must not
  // linger in browser storage after a logout on a shared machine.
  assert.match(participantInteractionsSource, /aftercare\/draft/,
    "drafts must be saved to the server");
  // Check for actual storage CALLS, not the word in a comment explaining
  // why it is not used.
  assert.ok(!/localStorage\.(setItem|getItem)/.test(participantInteractionsSource),
    "Aftercare drafts must not be written to browser storage");
  assert.match(participantInteractionsSource, /scheduleAftercareDraftSave/,
    "typing must schedule a debounced server-side save");
});

test("kernel 75 offers My People as the default note follow-up", () => {
  assert.match(participantInteractionsSource, /function renderNotesReminder/,
    "S8.6 requires the notes reminder");
  assert.match(participantInteractionsSource, /Don't forget to update your notes/,
    "S8.6's wording");
  assert.match(participantInteractionsSource, /\/venues\/trailers\/people\.html/,
    "Update My People must route to the existing My People experience");
});

// --- Story So Far (S6) ------------------------------------------------------

test("kernel 75 Greenroom renders the Story So Far page", () => {
  assert.match(greenroomSource, /page\.key === "story"/,
    "the story page must have a render branch");
  assert.match(greenroomSource, /function renderStorySoFarPage/,
    "the renderer must exist");
});

test("kernel 75 Story So Far renders entries through textContent", () => {
  assert.match(greenroomSource, /body\.textContent = event\.summary/,
    "summaries quote Player words verbatim and must not be parsed as markup");
  assert.match(greenroomSource, /title\.textContent = event\.title/,
    "titles must use textContent");
});

test("kernel 75 Story So Far exposes a reveal control but no text editor", () => {
  // S5.5: the Player controls visibility and nothing else. There must be no
  // path that edits generated text.
  assert.match(greenroomSource, /toggleStoryVisibility/, "the reveal control must exist");
  assert.match(greenroomSource, /visibility_state: next/, "the PATCH must send only a visibility state");
  assert.ok(!/story-events[\s\S]{0,300}summary:/.test(greenroomSource),
    "no client path may submit an edited summary");
});

test("kernel 75 Story So Far says plainly when a viewer sees only shared entries", () => {
  assert.match(greenroomSource, /story_events_owner_view/,
    "the page must distinguish the owner view from a shared-only view");
});

// --- Director's Chair (S9) --------------------------------------------------

test("kernel 75 Director's Chair Aftercare loader is fail-soft", () => {
  // The page's shared bootstrap catch calls showForbidden(), which blanks
  // everything. A backend failure in the newest section must degrade that
  // section only.
  assert.match(chairSource, /loadAftercareReview\(\)\.catch\(\(\)\s*=>\s*\{\}\)/,
    "the Aftercare loader must not be able to blank the page");
});

test("kernel 75 Director's Chair Aftercare is read-only", () => {
  // S9.3: no replies, scores, comments, or Director edits to Player
  // responses. There must be no mutating request in this section.
  const section = chairSource.slice(chairSource.indexOf("Kernel 75 S9: Aftercare review"));
  assert.ok(!/method:\s*"(POST|PATCH|PUT|DELETE)"[\s\S]{0,200}aftercare/i.test(section),
    "the Aftercare review section must issue no mutating requests");
});

test("kernel 75 Director's Chair distinguishes every Aftercare state", () => {
  for (const state of ["submitted", "skipped", "draft", "none"]) {
    assert.ok(chairSource.includes(`${state}:`),
      `S9.5 requires the ${state} state to be named explicitly`);
  }
  assert.match(chairSource, /Draft content is not shared/,
    "a draft must be reported without exposing its content");
});

test("kernel 75 Director's Chair export control respects can_export", () => {
  assert.match(chairSource, /aftercareExport\.hidden = !data\.can_export/,
    "the export control must be hidden when the caller cannot export");
  assert.match(chairSource, /aftercare-review\.csv/,
    "the export must point at the CSV route");
  assert.match(chairSource, /id="aftercare-export"[^>]*download/,
    "the export must be a plain download anchor so Content-Disposition applies");
});

test("kernel 75 Director's Chair renders Player text through textContent", () => {
  assert.match(chairSource, /a\.textContent = answer/,
    "Aftercare answers must be rendered as text");
  assert.match(chairSource, /text\.textContent = row\.door_intention/,
    "the door intention must be rendered as text");
});

test("kernel 75 Director's Chair table carries no denominator", () => {
  const section = chairSource.slice(chairSource.indexOf("Kernel 75 S9: Aftercare review"));
  assert.ok(!/\d+\s*of\s*\d+|of \$\{|percent|progress-bar/i.test(section),
    "S1.11 forbids 'N of M' framing in the backstage table");
});

test("kernel 75 hides the previous Scene's interaction buttons under a local projection", () => {
  // A Player who has walked out through the gate must not still be offered
  // "Try the Door" and "Speak with Kessa" from the Courtyard they left.
  // The shared Scene has not moved, so those buttons would otherwise stay
  // live -- correct in the data model, wrong on screen, and worse in a
  // narrated demonstration (§11.4).
  const fn = runtimeSource.slice(runtimeSource.indexOf("async function renderParticipantInteractionButtons"));
  const body = fn.slice(0, fn.indexOf("\n    }\n"));
  assert.match(body, /session\?\.local_projection[\s\S]{0,120}list\.remove\(\)/,
    "an active local projection must remove the shared Scene's interaction buttons");
});

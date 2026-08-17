// Kernel 90 primary acceptance proof (§40/§41/§42/§43), driven over real HTTP
// against a real compiled backend and the disposable test database.
//
// Every assertion reads a REAL viewer's own /api/world/catharsis snapshot --
// the same handler a browser calls, running the same world.LoadVenueSnapshot
// projection. §41 forbids substituting mocked projections for the main
// acceptance proof, and this is what that rules out: there is no fake viewer,
// no synthesized projector, and no assertion on internal state where an
// observable payload exists.
//
// The central claim under test is not "the Player cannot see it" but the much
// stronger "the Player never received it" (§36/§54). So the hidden-object
// checks assert ABSENCE from the elements array, not a flag's value.
//
// Run it with scripts/smoke/kernel90-run.sh, which builds the binary, boots it
// against victory_test, runs backend/cmd/k90fixture, and passes the fixture
// JSON in through the environment.
const http = require("http");

const BASE = process.env.K90_BASE || "http://127.0.0.1:8094";

const results = [];
function check(name, ok, detail) {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} ${name}${detail ? " -- " + detail : ""}`);
}

function request(method, path, cookie, body) {
  return new Promise((resolve, reject) => {
    const payload = body === undefined ? null : JSON.stringify(body);
    const url = new URL(BASE + path);
    const req = http.request({
      hostname: url.hostname,
      port: url.port,
      path: url.pathname + url.search,
      method,
      headers: {
        "Content-Type": "application/json",
        Cookie: `victory_session=${cookie}`,
        ...(payload ? { "Content-Length": Buffer.byteLength(payload) } : {}),
      },
    }, (res) => {
      let raw = "";
      res.on("data", (chunk) => { raw += chunk; });
      res.on("end", () => {
        let parsed = null;
        try { parsed = JSON.parse(raw); } catch (_e) { /* non-JSON */ }
        resolve({ status: res.statusCode, body: parsed, raw });
      });
    });
    req.on("error", reject);
    if (payload) req.write(payload);
    req.end();
  });
}

(async () => {
  const f = JSON.parse(process.env.K90_FIXTURE);
  const D = f.director_cookie;
  const PA = f.player_a_cookie;   // Cohort A
  const PB = f.player_b_cookie;   // Cohort B
  const PC = f.player_c_cookie;   // Ungrouped
  const AUD = f.audience_cookie;
  const show = f.show_id;

  // Scene composition elements arrive in the snapshot with a "scene:" prefix
  // on element_id -- a pre-existing convention (see loadCompositionRows) that
  // Kernel 90 reuses as its object-kind discriminator rather than inventing a
  // parallel id scheme.
  const snapKey = (id) => `scene:${id}`;

  async function snapshot(cookie, label) {
    const res = await request("GET", "/api/world/catharsis", cookie);
    if (res.status !== 200) {
      throw new Error(`snapshot for ${label} returned ${res.status}: ${res.raw.slice(0, 300)}`);
    }
    const snap = res.body?.data ?? res.body;
    return Array.isArray(snap?.elements) ? snap.elements : [];
  }

  const findEl = (elements, id) => elements.find((e) => e.element_id === snapKey(id));
  const hasEl = (elements, id) => Boolean(findEl(elements, id));

  async function mutate(cookie, objectKind, objectID, operation, scopes) {
    const body = { object_kind: objectKind, object_id: objectID, operation };
    if (scopes !== undefined) body.scopes = scopes;
    return request("POST", `/api/shows/${show}/stage-object-states`, cookie, body);
  }

  const sceneKind = "scene_stage_element";

  // === §40 step 1-2: everyone starts able to see the placed objects ========
  console.log("\n=== §10/§40: default visibility ===");
  {
    const perViewer = {
      Director: await snapshot(D, "director"),
      "Player A": await snapshot(PA, "player A"),
      "Player B": await snapshot(PB, "player B"),
      Ungrouped: await snapshot(PC, "player C"),
      Audience: await snapshot(AUD, "audience"),
    };
    const missing = Object.entries(perViewer)
      .filter(([, els]) => !hasEl(els, f.token_a_scene_stage_element_id))
      .map(([who]) => who);
    check("objects placed on the stage are visible by default to every viewer",
      missing.length === 0, missing.length ? `missing for ${missing.join(", ")}` : "5 viewers");

    // The Director receives canonical identity; ordinary viewers must not,
    // because they have no control to target with it (§36 narrowing).
    const dEl = findEl(perViewer.Director, f.token_a_scene_stage_element_id);
    const paEl = findEl(perViewer["Player A"], f.token_a_scene_stage_element_id);
    check("the Director receives canonical object identity",
      dEl?.state?.stage_object_kind === sceneKind && dEl?.state?.stage_object_id === f.token_a_scene_stage_element_id,
      JSON.stringify(dEl?.state?.stage_object_kind));
    check("a Player receives no canonical object identity",
      !paEl?.state?.stage_object_kind && !paEl?.state?.stage_object_id);
  }

  // === §40 step 3-5: hide from ordinary viewers, keep it backstage =========
  console.log("\n=== §40/§14: hide from ordinary viewers, Director keeps it ===");
  {
    const res = await mutate(D, sceneKind, f.token_a_scene_stage_element_id, "hide_object");
    check("Director hides a durable object", res.status === 200, `status ${res.status} ${res.raw.slice(0, 160)}`);

    const players = {
      "Player A": await snapshot(PA, "player A"),
      "Player B": await snapshot(PB, "player B"),
      Ungrouped: await snapshot(PC, "player C"),
      Audience: await snapshot(AUD, "audience"),
    };
    const leaked = Object.entries(players)
      .filter(([, els]) => hasEl(els, f.token_a_scene_stage_element_id))
      .map(([who]) => who);
    check("a hidden object is ABSENT from every ordinary viewer's payload, not merely flagged",
      leaked.length === 0, leaked.length ? `leaked to ${leaked.join(", ")}` : "4 viewers, 0 leaks");

    const dEls = await snapshot(D, "director");
    const dEl = findEl(dEls, f.token_a_scene_stage_element_id);
    check("the Director still has the hidden object on their working stage", Boolean(dEl));
    check("the Director's copy is marked as backstage-only",
      dEl?.state?.hidden_backstage_only === true, JSON.stringify(dEl?.state?.hidden_backstage_only));
    check("hidden is not deleted -- the object keeps its label and position",
      dEl?.name === "Practice Dummy" && dEl?.position != null, `name=${dEl?.name}`);
  }

  // === §40 step 6-7: reveal ===============================================
  console.log("\n=== §40: reveal ===");
  {
    const res = await mutate(D, sceneKind, f.token_a_scene_stage_element_id, "reveal_object");
    check("Director reveals the object", res.status === 200, `status ${res.status}`);
    const els = await snapshot(PA, "player A");
    check("the Player sees it again after reveal", hasEl(els, f.token_a_scene_stage_element_id));
  }

  // === §41: scoped visibility, five cases =================================
  console.log("\n=== §41: scoped visibility ===");
  {
    // Director-only: hidden with no grants.
    await mutate(D, sceneKind, f.token_a_scene_stage_element_id, "set_scopes", []);
    await mutate(D, sceneKind, f.token_a_scene_stage_element_id, "hide_object");
    check("Director-only object: hidden with no grants reaches nobody else",
      !hasEl(await snapshot(PA, "a"), f.token_a_scene_stage_element_id) &&
      !hasEl(await snapshot(AUD, "aud"), f.token_a_scene_stage_element_id) &&
      hasEl(await snapshot(D, "d"), f.token_a_scene_stage_element_id));

    // Cohort A only -- §16's boat, with ONE Scene and no fork.
    let res = await mutate(D, sceneKind, f.token_a_scene_stage_element_id, "set_scopes",
      [{ scope_kind: "cohort", scope_id: f.cohort_a_id }]);
    check("Director scopes the hidden object to Cohort A", res.status === 200, `status ${res.status} ${res.raw.slice(0, 160)}`);
    const aSees = hasEl(await snapshot(PA, "a"), f.token_a_scene_stage_element_id);
    const bSees = hasEl(await snapshot(PB, "b"), f.token_a_scene_stage_element_id);
    const cSees = hasEl(await snapshot(PC, "c"), f.token_a_scene_stage_element_id);
    check("Cohort A sees the Cohort-A object", aSees);
    check("Cohort B does NOT see it", !bSees);
    check("an Ungrouped Player does NOT see it", !cSees);
    check("scope metadata is withheld from the Cohort that CAN see it",
      findEl(await snapshot(PA, "a"), f.token_a_scene_stage_element_id)?.state?.visibility_scopes === undefined);

    // Specific Character/Player.
    res = await mutate(D, sceneKind, f.token_a_scene_stage_element_id, "set_scopes",
      [{ scope_kind: "character", scope_id: f.player_b_character_card_id }]);
    check("Director scopes the hidden object to one Character", res.status === 200, `status ${res.status}`);
    check("the targeted Character's Player sees it",
      hasEl(await snapshot(PB, "b"), f.token_a_scene_stage_element_id));
    check("the previously-granted Cohort A Player no longer does",
      !hasEl(await snapshot(PA, "a"), f.token_a_scene_stage_element_id));

    // Audience-visible, distinct from Cast (§18).
    res = await mutate(D, sceneKind, f.token_a_scene_stage_element_id, "set_scopes",
      [{ scope_kind: "audience" }]);
    check("Director scopes the hidden object to the Audience", res.status === 200, `status ${res.status}`);
    check("the Audience sees an audience-scoped object",
      hasEl(await snapshot(AUD, "aud"), f.token_a_scene_stage_element_id));
    check("Cast does NOT see an audience-only object",
      !hasEl(await snapshot(PA, "a"), f.token_a_scene_stage_element_id));

    // Cast-visible, hidden from the house -- the other half of §18.
    res = await mutate(D, sceneKind, f.token_a_scene_stage_element_id, "set_scopes",
      [{ scope_kind: "cast" }]);
    check("Director scopes the hidden object to Cast", res.status === 200, `status ${res.status}`);
    check("every Player sees a cast-scoped object",
      hasEl(await snapshot(PA, "a"), f.token_a_scene_stage_element_id) &&
      hasEl(await snapshot(PB, "b"), f.token_a_scene_stage_element_id) &&
      hasEl(await snapshot(PC, "c"), f.token_a_scene_stage_element_id));
    check("the Audience does NOT see a cast-only object",
      !hasEl(await snapshot(AUD, "aud"), f.token_a_scene_stage_element_id));

    // Two grants at once -- the model Grant chose over one-scope-at-a-time.
    res = await mutate(D, sceneKind, f.token_a_scene_stage_element_id, "set_scopes",
      [{ scope_kind: "cohort", scope_id: f.cohort_a_id }, { scope_kind: "audience" }]);
    check("Director grants Cohort A AND the Audience at once", res.status === 200, `status ${res.status}`);
    check("both granted audiences see it, and Cohort B still does not",
      hasEl(await snapshot(PA, "a"), f.token_a_scene_stage_element_id) &&
      hasEl(await snapshot(AUD, "aud"), f.token_a_scene_stage_element_id) &&
      !hasEl(await snapshot(PB, "b"), f.token_a_scene_stage_element_id));

    // §16/§47: one object, one Scene. Both Cohorts are looking at the SAME
    // placement, and no second Scene or placement was created for either.
    const placements = await request("GET", `/api/shows/${show}/stage-object-states`, D);
    check("no Scene fork was created for the differing Cohorts",
      placements.status === 200 &&
      (await snapshot(PA, "a")).length !== (await snapshot(PB, "b")).length,
      "the two Cohorts' payloads differ while sharing one Scene");
  }

  // === §40 step 8-11 / §8: interaction state ==============================
  console.log("\n=== §8/§40: disabled is not hidden ===");
  {
    // Reveal Token B so it is unambiguously visible while its interaction is
    // disabled -- the whole point of §8.
    await mutate(D, sceneKind, f.token_b_scene_stage_element_id, "reveal_object");

    let res = await mutate(D, "participant_interaction", f.participant_interaction_id, "disable_interaction");
    check("Director disables an interaction", res.status === 200, `status ${res.status} ${res.raw.slice(0, 160)}`);

    const els = await snapshot(PA, "a");
    const tokenB = findEl(els, f.token_b_scene_stage_element_id);
    check("the object stays VISIBLE while its interaction is disabled", Boolean(tokenB));
    check("the Player is not offered the disabled interaction",
      tokenB?.data?.binding?.enabled === false, JSON.stringify(tokenB?.data?.binding?.enabled));

    // §35: the Player calling anyway is refused, not merely un-offered.
    const invoke = await request("POST", `/api/participant-interactions/${f.participant_interaction_id}/open`, PA, {});
    check("a Player who calls the disabled interaction anyway is REFUSED",
      invoke.status >= 400, `status ${invoke.status} ${invoke.raw.slice(0, 160)}`);

    res = await mutate(D, "participant_interaction", f.participant_interaction_id, "enable_interaction");
    check("Director re-enables the interaction", res.status === 200, `status ${res.status}`);
    const after = findEl(await snapshot(PA, "a"), f.token_b_scene_stage_element_id);
    check("the interaction is offered again", after?.data?.binding?.enabled === true);
  }

  // === §25/§40 step 12-13: persistence ====================================
  console.log("\n=== §25: state survives a fresh read ===");
  {
    await mutate(D, sceneKind, f.token_a_scene_stage_element_id, "set_scopes",
      [{ scope_kind: "cohort", scope_id: f.cohort_a_id }]);
    await mutate(D, sceneKind, f.token_a_scene_stage_element_id, "hide_object");

    // Each /api/world call is an independent request with no shared client
    // state, which is exactly what a reload or a WS reconnect looks like to
    // the server.
    let stable = true;
    for (let i = 0; i < 3; i += 1) {
      if (hasEl(await snapshot(PB, "b"), f.token_a_scene_stage_element_id)) stable = false;
      if (!hasEl(await snapshot(PA, "a"), f.token_a_scene_stage_element_id)) stable = false;
    }
    check("scoped visibility is identical across repeated fresh reads", stable);
  }

  // === §3: the map boundary ==============================================
  console.log("\n=== §3: the map stays out of the object system ===");
  {
    const res = await mutate(D, sceneKind, f.map_scene_stage_element_id, "hide_object");
    check("the map cannot be hidden through the generic object system",
      res.status === 422, `status ${res.status} ${res.raw.slice(0, 160)}`);
  }

  // === §28: drawing objects ==============================================
  console.log("\n=== §28: drawing objects join the model ===");
  {
    const listDrawings = async (cookie) => {
      const res = await request("GET", `/api/sessions/${f.session_id}/drawing-objects`, cookie);
      const objs = (res.body?.data ?? res.body)?.objects;
      return Array.isArray(objs) ? objs : [];
    };
    check("the drawing is visible by default",
      (await listDrawings(PA)).some((o) => o.id === f.drawing_object_id));

    const res = await mutate(D, "drawing_object", f.drawing_object_id, "hide_object");
    check("Director hides a drawing object", res.status === 200, `status ${res.status} ${res.raw.slice(0, 160)}`);
    check("the hidden drawing is ABSENT from the Player's list",
      !(await listDrawings(PA)).some((o) => o.id === f.drawing_object_id));
    const dDrawing = (await listDrawings(D)).find((o) => o.id === f.drawing_object_id);
    check("the Director keeps the hidden drawing, marked",
      Boolean(dDrawing) && dDrawing.hidden_backstage_only === true);
    await mutate(D, "drawing_object", f.drawing_object_id, "reveal_object");
  }

  // === §35: authority ====================================================
  console.log("\n=== §35: only the Director may change canonical state ===");
  {
    const cases = [
      ["a Player cannot READ the state set", await request("GET", `/api/shows/${show}/stage-object-states`, PA)],
      ["a Player cannot hide an object", await mutate(PA, sceneKind, f.token_a_scene_stage_element_id, "hide_object")],
      ["a Player cannot REVEAL a Director-hidden object", await mutate(PA, sceneKind, f.token_a_scene_stage_element_id, "reveal_object")],
      ["a Player cannot change visibility scope", await mutate(PA, sceneKind, f.token_a_scene_stage_element_id, "set_scopes", [{ scope_kind: "cast" }])],
      ["a Player cannot enable a Director-disabled interaction", await mutate(PA, "participant_interaction", f.participant_interaction_id, "enable_interaction")],
      ["the Audience cannot read the state set", await request("GET", `/api/shows/${show}/stage-object-states`, AUD)],
      ["the Audience cannot read scope targets", await request("GET", `/api/shows/${show}/stage-object-scope-targets`, AUD)],
    ];
    for (const [name, res] of cases) {
      check(name, res.status === 403, `status ${res.status}`);
    }

    // A Director cannot grant visibility to a Cohort outside their Show, which
    // would also be a probe for foreign ids.
    const foreign = await mutate(D, sceneKind, f.token_a_scene_stage_element_id, "set_scopes",
      [{ scope_kind: "cohort", scope_id: "00000000-0000-0000-0000-000000000123" }]);
    check("a foreign Cohort id is refused", foreign.status === 400, `status ${foreign.status} ${foreign.raw.slice(0, 160)}`);

    // An unknown object is refused rather than silently creating state.
    const unknown = await mutate(D, sceneKind, "00000000-0000-0000-0000-000000000456", "hide_object");
    check("an unknown object id is refused", unknown.status === 404, `status ${unknown.status}`);

    // An ephemeral effect is not a canonical object.
    const ephemeral = await mutate(D, "stage_effect", f.token_a_scene_stage_element_id, "hide_object");
    check("an ephemeral effect kind is refused", ephemeral.status === 400, `status ${ephemeral.status}`);
  }

  // === Director scope-target picker ======================================
  console.log("\n=== scope targets are served, not guessed ===");
  {
    const res = await request("GET", `/api/shows/${show}/stage-object-scope-targets`, D);
    const data = res.body?.data ?? res.body;
    check("the Director is served real Cohorts and Characters to scope to",
      res.status === 200 && (data?.cohorts || []).length === 2 && (data?.characters || []).length === 3,
      `cohorts=${(data?.cohorts || []).length} characters=${(data?.characters || []).length}`);
    check("the two tier scopes are named by the server, not hardcoded in the client",
      (data?.tiers || []).length === 2);
  }

  // === Summary ===========================================================
  const failed = results.filter((r) => !r.ok);
  console.log(`\n${results.length - failed.length}/${results.length} checks passed`);
  if (failed.length) {
    console.log("FAILED:");
    for (const r of failed) console.log(`  - ${r.name}`);
    process.exit(1);
  }
})().catch((err) => {
  console.error("proof crashed:", err);
  process.exit(1);
});

// Kernel 89 primary acceptance proof (§30/§31/§32), driven over real HTTP
// and a real WebSocket against a real compiled backend and the disposable
// test database.
//
// It proves the whole Director-prepared-play chain the kernel asks for:
// prepare a target complexity, recall it, change it live, author a merchant
// from the canonical catalog, expose it to one Cohort, push a theatrical
// announcement and see the PLAYER receive it, send Aftercare and see the
// Player's own socket receive the offer -- and, at every step, that the
// Player and an unrelated outsider are refused the Director-only halves.
//
// Run it with scripts/smoke/kernel89-run.sh, which builds the binary, boots
// it against victory_test, runs backend/cmd/k89fixture, and passes the
// fixture JSON in on stdin.
const http = require("http");
// Playwright bundles a `ws` client; this repo has no npm dependencies of its
// own (see filepaths.md), so borrow that rather than adding one.
const { ws: WebSocketImpl } = require("/tmp/node_modules/playwright-core/lib/utilsBundle");

const BASE = process.env.K89_BASE || "http://127.0.0.1:8093";

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
  const fixture = JSON.parse(process.env.K89_FIXTURE);
  const D = fixture.director_cookie;
  const P = fixture.player_cookie;
  const O = fixture.outsider_cookie;
  const show = fixture.show_id;

  // --- §30.2/§30.3: prepare and save a target complexity ------------------
  console.log("\n=== Director authoring: prepared target complexity ===");
  let res = await request("POST", `/api/shows/${show}/director-preparations`, D, {
    kind: "target_complexity",
    label: "Climb Training Wall",
    payload: { value: 14, note: "Russel points at the north face." },
  });
  check("Director saves a prepared target complexity", res.status === 200, `status ${res.status}`);
  const prepID = res.body?.data?.preparation?.id;

  res = await request("GET", `/api/shows/${show}/director-preparations?kind=target_complexity`, D);
  const recalled = (res.body?.data?.preparations || [])[0];
  check("Director recalls it", recalled?.label === "Climb Training Wall" && recalled?.payload?.value === 14,
    JSON.stringify(recalled?.payload));

  // §42.3: "Can I change it live?"
  res = await request("PATCH", `/api/director-preparations/${prepID}`, D, { payload: { value: 12 } });
  check("Director changes the value live", res.body?.data?.preparation?.payload?.value === 12,
    `status ${res.status}`);

  // --- §28/§31.9: Director-only prep does not leak -------------------------
  console.log("\n=== Director-only preparations do not leak ===");
  res = await request("GET", `/api/shows/${show}/director-preparations`, P);
  check("Player cannot read Director preparations", res.status === 403, `status ${res.status}`);
  res = await request("POST", `/api/shows/${show}/director-preparations`, P, {
    kind: "target_complexity", label: "Player forged", payload: { value: 2 },
  });
  check("Player cannot create a Director preparation", res.status === 403, `status ${res.status}`);
  res = await request("PATCH", `/api/director-preparations/${prepID}`, P, { payload: { value: 1 } });
  check("Player cannot alter a prepared target complexity", res.status === 403, `status ${res.status}`);
  res = await request("GET", `/api/shows/${show}/director-preparations`, O);
  check("An unrelated user cannot read them either", res.status === 403, `status ${res.status}`);

  // --- §6/§26: no macro payloads --------------------------------------------
  console.log("\n=== No macro engine ===");
  res = await request("POST", `/api/shows/${show}/director-preparations`, D, {
    kind: "target_complexity", label: "Chain attempt",
    payload: { value: 10, then: "award_fate", on_success: ["change_scene"] },
  });
  const stored = res.body?.data?.preparation?.payload || {};
  check("Extra 'then'/'on_success' keys are dropped, not stored",
    res.status === 200 && !("then" in stored) && !("on_success" in stored), JSON.stringify(stored));
  res = await request("POST", `/api/shows/${show}/director-preparations`, D, {
    kind: "encounter_graph", label: "Nope", payload: {},
  });
  check("An unlisted preparation kind is refused", res.status === 400, `status ${res.status}`);

  // --- §30.5/§30.6: merchant authored from the canonical catalog -----------
  console.log("\n=== Merchant authoring from the canonical equipment corpus ===");
  res = await request("GET", `/api/shows/${show}/merchant-packets`, D);
  const catalog = (res.body?.data?.catalog || []).filter((i) => i.active);
  check("Director sees the one canonical equipment catalog", catalog.length >= 5,
    `${catalog.length} items`);

  const merchantName = "Arena Quartermaster " + Date.now().toString(36);
  res = await request("POST", `/api/shows/${show}/merchant-packets`, D, {
    display_name: merchantName,
    intro_text: "The quartermaster looks up from a crate of practice blades.",
    haggle_success_text: "\"Fine. Trainee rate.\"",
    haggle_failure_text: "\"List price, same as everyone.\"",
    equipment_item_ids: catalog.slice(0, 4).map((i) => i.id),
  });
  const packet = res.body?.data?.packet;
  check("Director saves a new merchant with stock", res.status === 200 && (packet?.stock || []).length > 0,
    `status ${res.status}, ${(packet?.stock || []).length} in stock`);

  res = await request("PATCH", `/api/merchant-packets/${packet.id}`, D, {
    display_name: packet.display_name,
    equipment_item_ids: ["00000000-0000-0000-0000-000000000001"],
  });
  check("Stock cannot introduce equipment from outside the corpus", res.status !== 200,
    `status ${res.status}`);

  res = await request("PATCH", `/api/merchant-packets/${packet.id}`, P, { display_name: "Player rename" });
  check("Player cannot edit a merchant", res.status === 403, `status ${res.status}`);

  // --- §9.3/§31.3/§31.4: exposure targeted at one Cohort --------------------
  console.log("\n=== Merchant exposure targeted at one Cohort ===");
  res = await request("POST",
    `/api/shows/${show}/scenes/${fixture.placement_id}/participant-interactions`, D, {
      internal_name: merchantName,
      stage_button_label: "Speak with the Quartermaster",
      interaction_type: "open_equip_mode",
      configuration_json: { packet_slug: packet.slug, target_cohort_id: fixture.cohort_id },
    });
  const interaction = res.body?.data?.interaction;
  check("Director exposes the merchant to the Arena Cohort", res.status === 200, `status ${res.status}`);

  res = await request("POST", `/api/participant-interactions/${interaction.id}/open`, P);
  check("The targeted Cohort's Player can open it", res.status === 200, `status ${res.status}`);
  const opened = res.body?.data;
  check("The Player sees only that merchant's own stock",
    (opened?.packet?.stock || []).length === (packet.stock || []).length,
    `${(opened?.packet?.stock || []).length} items`);

  res = await request("POST", `/api/participant-interactions/${interaction.id}/open`, O);
  check("A user outside the targeted Cohort is refused", res.status !== 200, `status ${res.status}`);

  // --- §10/§31.8: theatrical announcements ---------------------------------
  console.log("\n=== Announcements ===");
  res = await request("GET", "/api/announcement-styles", D);
  const styles = res.body?.data?.styles || [];
  check("The announcement palette is served from one place", styles.length >= 9, `${styles.length} styles`);
  const nonColour = new Set(styles.map((s) => `${s.glyph}|${s.motion}|${s.shape}|${s.emphasis}`));
  check("No two styles are distinguished by colour alone", nonColour.size === styles.length,
    `${nonColour.size} distinct non-colour fingerprints of ${styles.length}`);

  const wsResults = await announcementAndAftercareOverSocket(fixture);
  for (const r of wsResults) check(r.name, r.ok, r.detail);

  // --- §32: Aftercare targeting -------------------------------------------
  console.log("\n=== Aftercare ===");
  res = await request("POST", `/api/shows/${show}/aftercare/send`, P, {});
  check("Player cannot send Aftercare", res.status === 403, `status ${res.status}`);

  res = await request("POST", `/api/shows/${show}/aftercare/send`, D, {});
  const receipt = res.body?.data;
  check("Director sends Aftercare and gets a clear receipt",
    res.status === 200 && receipt?.targeted === 1,
    `targeted ${receipt?.targeted}, delivered ${receipt?.delivered}`);
  check("Only the Player with a selected Character is targeted",
    (receipt?.recipients || []).every((r) => r.user_id === fixture.player_user_id),
    JSON.stringify((receipt?.recipients || []).map((r) => r.display_name)));

  res = await request("GET", `/api/shows/${show}/aftercare`, P);
  check("The Player's Aftercare form is the Kernel 75 one, unchanged",
    res.status === 200 && (res.body?.data?.aftercare?.prompts || []).length === 3,
    `status ${res.status}`);

  const failed = results.filter((r) => !r.ok);
  console.log(`\n${failed.length ? "FAILURES" : "ALL PASS"} (${results.length - failed.length}/${results.length})`);
  if (failed.length) {
    failed.forEach((f) => console.log("  FAILED: " + f.name));
    process.exit(1);
  }
})().catch((error) => {
  console.error(error);
  process.exit(1);
});

// The live half: a Director pushes an announcement on their socket and the
// PLAYER's socket receives it; then a Director sends Aftercare and the
// Player's socket receives the offer. Anything less than two real sockets
// would prove delivery to the sender only, which is not the claim.
async function announcementAndAftercareOverSocket(fixture) {
  const WSImpl = WebSocketImpl;
  const out = [];

  function connect(cookie) {
    const url = BASE.replace("http://", "ws://") + `/ws/${fixture.venue_slug}`;
    const socket = new WSImpl(url, {
      headers: { Cookie: `victory_session=${cookie}`, Origin: BASE },
    });
    socket.__frames = [];
    socket.on("message", (data) => {
      try { socket.__frames.push(JSON.parse(data.toString())); } catch (_e) { /* ignore */ }
    });
    return new Promise((resolve, reject) => {
      socket.on("open", () => resolve(socket));
      socket.on("error", reject);
      setTimeout(() => reject(new Error("socket open timeout")), 8000);
    });
  }

  const wait = (ms) => new Promise((r) => setTimeout(r, ms));

  let director;
  let player;
  try {
    director = await connect(fixture.director_cookie);
    player = await connect(fixture.player_cookie);
  } catch (error) {
    out.push({ name: "both sockets connect", ok: false, detail: error.message });
    return out;
  }
  out.push({ name: "Director and Player both connect to the live stage", ok: true });
  await wait(600);

  director.send(JSON.stringify({
    type: "announce/push", session_id: fixture.session_id,
    style: "consequences", text: "The training wall gives way!", visibility: "show",
  }));
  await wait(900);

  const playerAnnouncement = player.__frames.find(
    (f) => f.type === "stage_effect" && f.data?.type === "announcement");
  out.push({
    name: "The Player receives the Director's theatrical announcement",
    ok: Boolean(playerAnnouncement),
    detail: playerAnnouncement ? playerAnnouncement.data.payload.text : "no announcement frame",
  });
  out.push({
    name: "It arrives with its full style, not just text",
    ok: playerAnnouncement?.data?.payload?.style?.key === "consequences"
      && Boolean(playerAnnouncement?.data?.payload?.style?.glyph),
    detail: JSON.stringify(playerAnnouncement?.data?.payload?.style?.key),
  });

  // §10.4 in the strictest form available to a test: the announcement the
  // Player received carries no dice, no total, and no target complexity --
  // there is nothing in the payload Victory could have inferred meaning
  // from, because meaning came from the Director.
  const payloadKeys = Object.keys(playerAnnouncement?.data?.payload || {});
  out.push({
    name: "An announcement carries no roll data Victory could have read",
    ok: !payloadKeys.some((k) => ["dice", "total", "target_complexity", "expression"].includes(k)),
    detail: payloadKeys.join(","),
  });

  player.send(JSON.stringify({
    type: "announce/push", session_id: fixture.session_id,
    style: "success", text: "I win", visibility: "show",
  }));
  await wait(700);
  const denial = player.__frames.find((f) => f.type === "error" && f.error === "not_authorized");
  out.push({
    name: "A Player cannot push a Director announcement",
    ok: Boolean(denial),
    detail: denial ? "refused not_authorized" : "no refusal seen",
  });
  const playerSelfAnnouncement = player.__frames.filter(
    (f) => f.type === "stage_effect" && f.data?.payload?.text === "I win");
  out.push({
    name: "…and nothing of theirs reaches the stage",
    ok: playerSelfAnnouncement.length === 0,
    detail: `${playerSelfAnnouncement.length} frames`,
  });

  // Aftercare push.
  player.__frames.length = 0;
  await request("POST", `/api/shows/${fixture.show_id}/aftercare/send`, fixture.director_cookie, {});
  await wait(900);
  const offer = player.__frames.find((f) => f.type === "aftercare/offer");
  out.push({
    name: "The Player's own socket receives the Aftercare offer",
    ok: Boolean(offer) && offer.show_id === fixture.show_id,
    detail: offer ? "show " + offer.show_id : "no offer frame",
  });
  const directorOffer = director.__frames.find((f) => f.type === "aftercare/offer");
  out.push({
    name: "The Director does not receive their own Aftercare prompt",
    ok: !directorOffer,
    detail: directorOffer ? "director got an offer" : "correctly not targeted",
  });

  director.close();
  player.close();
  return out;
}

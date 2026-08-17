// Kernel 89 §10, second venue: announcements must work in the First Theater,
// not only in Catharsis.
//
// This is a narrow, deliberate proof. First Theater carries none of Kernel
// 85's Socio tools and none of the merchant/Aftercare capability flags, so
// the only Kernel 89 claim that applies there is the announcement one — and
// that claim is exactly what this script settles, over a real
// /ws/first-theater socket with a Director and a Player.
//
// PRECONDITION, and the reason this script currently reports a blocker
// rather than a pass: `access.ResolveVisibleVenues` has **no visibility
// branch for `first-theater` at all** (grep it -- zero matches), so no
// ordinary Director or Player can open /ws/first-theater; the upgrade 403s
// before any Kernel 89 code runs. That is a pre-existing condition
// documented in current-state.md's Known Gaps ("still investor/demo-only --
// no participant-facing map-visibility work targets it"), not something
// Kernel 89 introduced, and Kernel 89 deliberately did not widen venue
// visibility to paper over it.
//
// This script is kept, working, so that the moment First Theater gains a
// visibility branch its announcement support can be proven in one command
// instead of re-derived. Following kernel74-tutorial-browser.js's
// precedent: preflight, report the blocker plainly, and exit rather than
// pretending or forcing.
//
// Run via scripts/smoke/kernel89-run.sh --first-theater.
const http = require("http");
const { ws: WebSocketImpl } = require("/tmp/node_modules/playwright-core/lib/utilsBundle");

const BASE = process.env.K89_BASE || "http://127.0.0.1:8093";

const results = [];
function check(name, ok, detail) {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} ${name}${detail ? " -- " + detail : ""}`);
}

const wait = (ms) => new Promise((r) => setTimeout(r, ms));

function connect(fixture, cookie) {
  const url = BASE.replace("http://", "ws://") + `/ws/${fixture.venue_slug}`;
  const socket = new WebSocketImpl(url, {
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

(async () => {
  const fixture = JSON.parse(process.env.K89_FIXTURE);
  check("fixture is on the First Theater venue", fixture.venue_slug === "first-theater",
    fixture.venue_slug);

  let director;
  let player;
  try {
    director = await connect(fixture, fixture.director_cookie);
    player = await connect(fixture, fixture.player_cookie);
  } catch (error) {
    if (String(error.message).includes("403")) {
      console.log("\nPRECONDITION NOT MET -- not a Kernel 89 failure.");
      console.log("  /ws/first-theater refused the upgrade with 403.");
      console.log("  Cause: access.ResolveVisibleVenues has no visibility branch for");
      console.log("  'first-theater', so no ordinary user can reach the venue at all.");
      console.log("  This predates Kernel 89 (current-state.md Known Gaps) and Kernel 89");
      console.log("  deliberately did not widen venue visibility to work around it.");
      console.log("\n  Kernel 89 status for this venue: announcement modules are LOADED in");
      console.log("  frontend/venues/first-theater/index.html and the server's announce/push");
      console.log("  handler is venue-agnostic (proven on /ws/catharsis), but no Director");
      console.log("  can reach First Theater to use them. Reported as PARTIAL, not PASS.");
      process.exit(2);
    }
    throw error;
  }
  check("Director and Player connect to /ws/first-theater", true);
  await wait(600);

  director.send(JSON.stringify({
    type: "announce/push", session_id: fixture.session_id,
    style: "revelation", text: "The banner drops.", visibility: "show",
  }));
  await wait(900);

  const received = player.__frames.find(
    (f) => f.type === "stage_effect" && f.data?.type === "announcement");
  check("The Player receives a Director announcement in the First Theater",
    Boolean(received), received ? received.data.payload.text : "no announcement frame");
  check("…with its full style, from the same server-side palette",
    received?.data?.payload?.style?.key === "revelation"
      && Boolean(received?.data?.payload?.style?.glyph),
    received?.data?.payload?.style?.key || "none");

  player.send(JSON.stringify({
    type: "announce/push", session_id: fixture.session_id,
    style: "success", text: "not mine to send", visibility: "show",
  }));
  await wait(700);
  check("A Player cannot push one here either",
    Boolean(player.__frames.find((f) => f.type === "error" && f.error === "not_authorized")));

  director.close();
  player.close();

  const failed = results.filter((r) => !r.ok);
  console.log(`\n${failed.length ? "FAILURES" : "ALL PASS"} (${results.length - failed.length}/${results.length})`);
  process.exit(failed.length ? 1 : 0);
})().catch((error) => {
  console.error(error);
  process.exit(1);
});

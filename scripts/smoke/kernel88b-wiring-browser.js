// Kernel 88B: end-to-end wiring proof for per-request error routing.
//
// The unit tests in kernel88b-action-error-routing-test.js cover each module
// in isolation. This one loads socket.js, action-requests.js and
// session-sync.js into a real browser in the same order the venue pages do,
// and pushes a raw error frame through the whole chain -- the one thing the
// Node tests cannot check, since runtime.js itself is browser-only.
const http = require("http"); const fs = require("fs"); const path = require("path");
const { chromium } = require("/tmp/node_modules/playwright");
const LIB = "/opt/victory/frontend/lib/stage-runtime";
const server = http.createServer((req, res) => {
  const m = req.url.match(/\/lib\/stage-runtime\/([\w.-]+\.js)/);
  if (m) { res.writeHead(200, {"Content-Type":"application/javascript"}); res.end(fs.readFileSync(path.join(LIB, m[1]))); return; }
  res.writeHead(200, {"Content-Type":"text/html"}); res.end(fs.readFileSync(path.join(__dirname, "kernel88b-wiring-harness.html")));
});
(async () => {
  await new Promise(r => server.listen(4603, r));
  const browser = await chromium.launch({ args: ["--disable-gpu","--disable-webgl","--disable-software-rasterizer"] });
  const page = await browser.newPage();
  const errors = [];
  page.on("pageerror", e => errors.push(e.message));
  await page.goto("http://127.0.0.1:4603/", { waitUntil: "domcontentloaded" });
  const claimed = await page.evaluate(() => window.__run({ type:"error", error:"not_your_turn", request_id:"r1" }, "r1"));
  const unclaimed = await page.evaluate(() => window.__run({ type:"error", error:"showing_closed", request_id:"r2" }, null));
  const noId = await page.evaluate(() => window.__run({ type:"error", error:"action_denied" }, null));
  console.log("pageerrors:", errors.length ? errors : "none");
  console.log("claimed  :", JSON.stringify(claimed));
  console.log("unclaimed:", JSON.stringify(unclaimed));
  console.log("no id    :", JSON.stringify(noId));
  const ok = errors.length === 0
    && claimed.claimedText === "not_your_turn" && claimed.generic.length === 0
    && unclaimed.generic.length === 4 && noId.generic.length === 4 && noId.requestId === "";
  console.log(ok ? "\nWIRING OK" : "\nWIRING FAILED");
  await browser.close(); server.close(); process.exit(ok ? 0 : 1);
})();

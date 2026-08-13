// Minimal local dev static+proxy server for Kernel 87 browser proof.
// Serves frontend/ statically and proxies /api, /ws, /auth to the local
// backend, so both are same-origin (required by the WS upgrader's Origin
// check and by cookie scoping).
const http = require("node:http");
const httpProxy = (() => {
  // Tiny hand-rolled proxy (no new dependency): forwards method/headers/
  // body and pipes the response back, including upgrade (WS) support.
  return {
    web(req, res, target) {
      const url = new URL(req.url, target);
      const proxyReq = http.request(url, { method: req.method, headers: req.headers }, (proxyRes) => {
        res.writeHead(proxyRes.statusCode, proxyRes.headers);
        proxyRes.pipe(res);
      });
      req.pipe(proxyReq);
      proxyReq.on("error", (e) => { res.writeHead(502); res.end("proxy error: " + e.message); });
    },
  };
})();
const fs = require("node:fs");
const path = require("node:path");

const FRONTEND_ROOT = process.argv[2] || path.join(__dirname, "..", "..", "frontend");
const BACKEND_TARGET = process.argv[3] || "http://127.0.0.1:8092";
const PORT = Number(process.argv[4] || 8090);

const MIME = { ".html": "text/html", ".js": "application/javascript", ".css": "text/css", ".png": "image/png", ".jpg": "image/jpeg", ".svg": "image/svg+xml", ".json": "application/json" };

const server = http.createServer((req, res) => {
  if (req.url.startsWith("/api/") || req.url.startsWith("/auth/") || req.url.startsWith("/health")) {
    return httpProxy.web(req, res, BACKEND_TARGET);
  }
  let filePath = path.join(FRONTEND_ROOT, decodeURIComponent(req.url.split("?")[0]));
  if (filePath.endsWith("/")) filePath = path.join(filePath, "index.html");
  fs.stat(filePath, (err, stat) => {
    if (err || !stat.isFile()) {
      // try appending index.html for directory-style routes
      const alt = path.join(filePath, "index.html");
      fs.stat(alt, (err2, stat2) => {
        if (err2 || !stat2.isFile()) { res.writeHead(404); res.end("not found: " + filePath); return; }
        res.writeHead(200, { "Content-Type": "text/html" });
        fs.createReadStream(alt).pipe(res);
      });
      return;
    }
    res.writeHead(200, { "Content-Type": MIME[path.extname(filePath)] || "application/octet-stream" });
    fs.createReadStream(filePath).pipe(res);
  });
});

server.on("upgrade", (req, socket, head) => {
  // Browser-proof runs open/close many short-lived WS connections (one per
  // signed-up account/context); a client tab closing mid-upgrade produces
  // a plain ECONNRESET on these sockets, which Node treats as an unhandled
  // 'error' event (process-fatal) unless something listens for it. This is
  // a dev-proxy robustness fix only -- it never touches production, which
  // doesn't run this script at all.
  socket.on("error", () => {});
  const url = new URL(req.url, BACKEND_TARGET);
  const target = http.request(url, { method: "GET", headers: req.headers });
  target.on("error", () => { try { socket.destroy(); } catch (_e) { /* already closed */ } });
  target.on("upgrade", (proxyRes, proxySocket, proxyHead) => {
    proxySocket.on("error", () => {});
    socket.write(`HTTP/1.1 ${proxyRes.statusCode} ${proxyRes.statusMessage}\r\n` +
      Object.entries(proxyRes.headers).map(([k, v]) => `${k}: ${v}`).join("\r\n") + "\r\n\r\n");
    proxySocket.pipe(socket);
    socket.pipe(proxySocket);
  });
  target.end();
});

server.listen(PORT, () => console.log(`dev-proxy listening on http://127.0.0.1:${PORT} -> ${BACKEND_TARGET}`));

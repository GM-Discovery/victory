const test = require("node:test");
const assert = require("node:assert/strict");

const socketControllerModule = require("../../frontend/venues/first-theater/runtime/socket-controller.js");

class FakeSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSED = 3;

  constructor(url) {
    this.url = url;
    this.readyState = FakeSocket.CONNECTING;
    this.sent = [];
    this.listeners = new Map();
    FakeSocket.instances.push(this);
  }

  addEventListener(type, handler) {
    if (!this.listeners.has(type)) {
      this.listeners.set(type, []);
    }
    this.listeners.get(type).push(handler);
  }

  removeEventListener(type, handler) {
    const handlers = this.listeners.get(type) || [];
    this.listeners.set(type, handlers.filter((fn) => fn !== handler));
  }

  send(payload) {
    this.sent.push(JSON.parse(payload));
  }

  emit(type, event = {}) {
    for (const handler of this.listeners.get(type) || []) {
      handler({ ...event, target: this });
    }
  }
}

FakeSocket.instances = [];

test("connectSocket queues while connecting and flushes on open", () => {
  FakeSocket.instances.length = 0;
  const calls = [];
  const controller = socketControllerModule.createSocketController({
    getSessionId: () => "session-1",
    getStageCamera: () => ({ getView: () => ({ panX: 1, panY: 2, zoomRelativeToFit: 3 }), screenToWorld: () => ({ x: 4, y: 5 }) }),
    getLastStagePoint: () => ({ x: 6, y: 7 }),
    getStageSize: () => ({ width: 800, height: 600 }),
    getLocation: () => ({ protocol: "https:", host: "victory.test" }),
    getWebSocketCtor: () => FakeSocket,
    getPerformanceNow: () => 1000,
    setStageStatus: (text) => calls.push(["status", text]),
    setMovementLine: (text) => calls.push(["move", text]),
    updateShellMetaPresentation: () => calls.push(["meta"]),
    refreshWorld: async () => {},
    onMessage: () => null,
  });

  controller.connectSocket();
  const socket = controller.getWebSocket();
  assert.equal(socket.url, "wss://victory.test/ws/the-cave");
  assert.equal(controller.sendAction("create/token", { payload: "queued" }), true);
  assert.deepEqual(socket.sent, []);

  socket.readyState = FakeSocket.OPEN;
  socket.emit("open");
  assert.deepEqual(socket.sent, [
    {
      type: "create/token",
      session_id: "session-1",
      payload: "queued",
    },
  ]);
  assert.ok(calls.some((entry) => entry[0] === "status" && entry[1] === "Connected to the Cave WebSocket."));
  assert.ok(calls.some((entry) => entry[0] === "meta"));
});

test("focus ping payload uses current stage camera and last stage point", () => {
  const calls = [];
  const controller = socketControllerModule.createSocketController({
    getSessionId: () => "session-1",
    getStageCamera: () => ({
      getView: () => ({ panX: 9, panY: 8, zoomRelativeToFit: 1.5 }),
      screenToWorld: () => ({ x: 40, y: 50 }),
    }),
    getLastStagePoint: () => ({ x: 60, y: 70 }),
    getStageSize: () => ({ width: 100, height: 80 }),
    getLocation: () => ({ protocol: "http:", host: "victory.test" }),
    getWebSocketCtor: () => FakeSocket,
    getPerformanceNow: () => 1000,
    setStageStatus: (text) => calls.push(["status", text]),
    setMovementLine: (text) => calls.push(["move", text]),
    updateShellMetaPresentation: () => {},
    refreshWorld: async () => {},
    onMessage: () => null,
  });

  const payload = controller.buildFocusPingPayload();
  assert.equal(payload.venue_slug, "the-cave");
  assert.equal(payload.focus_x, 60);
  assert.equal(payload.focus_y, 70);
  assert.equal(payload.camera_center_x, 40);
  assert.equal(payload.camera_center_y, 50);
  assert.equal(payload.camera_pan_x, 9);
  assert.equal(payload.camera_pan_y, 8);
  assert.equal(payload.camera_zoom_relative_to_fit, 1.5);
  assert.match(payload.event_id, /^\d+-[0-9a-f]+$/);

  controller.connectSocket();
  const socket = controller.getWebSocket();
  socket.readyState = FakeSocket.OPEN;
  socket.emit("open");
  assert.equal(controller.sendFocusPing(), true);
  assert.equal(socket.sent[socket.sent.length - 1].type, "venue/focus_ping");
  assert.ok(calls.some((entry) => entry[0] === "status" && entry[1] === "Focused venue."));
  assert.ok(calls.some((entry) => entry[0] === "move" && entry[1] === "Focused venue."));
});

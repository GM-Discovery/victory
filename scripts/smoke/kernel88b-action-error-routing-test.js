// Kernel 88B: per-request error routing.
//
// Proves the whole chain a refused action travels: the socket frame carries
// request_id through normalization, the registry matches it to whoever sent
// it, and session-sync stands the generic surfaces down when a module claims
// its own failure (and does NOT stand them down otherwise).
//
// Plain Node -- these stage-runtime modules are UMD factories, so none of
// this needs a browser.
const assert = require("assert");
const path = require("path");

const LIB = process.env.STAGE_RUNTIME_DIR || "/opt/victory/frontend/lib/stage-runtime";
const { createActionRequestRegistry } = require(path.join(LIB, "action-requests.js"));
const { createSessionSync } = require(path.join(LIB, "session-sync.js"));
const socketModule = require(path.join(LIB, "socket.js"));

const results = [];
function test(name, fn) {
  try {
    fn();
    results.push({ name, ok: true });
    console.log("PASS " + name);
  } catch (err) {
    results.push({ name, ok: false });
    console.log("FAIL " + name + " -- " + err.message);
  }
}

// --- Registry -----------------------------------------------------------

function fakeClock() {
  let seq = 0;
  const timers = new Map();
  return {
    setTimeoutFn: (cb, ms) => { timers.set(++seq, { cb, ms }); return seq; },
    clearTimeoutFn: (id) => { timers.delete(id); },
    fire: () => { for (const [id, t] of Array.from(timers)) { timers.delete(id); t.cb(); } },
    count: () => timers.size,
  };
}

test("an error reaches the handler that registered its request id", () => {
  const clock = fakeClock();
  const reg = createActionRequestRegistry(clock);
  let seen = null;
  reg.register("req-1", (errorText) => { seen = errorText; return true; });
  const claimed = reg.dispatch("req-1", "not_your_turn", {});
  assert.strictEqual(seen, "not_your_turn");
  assert.strictEqual(claimed, true, "handler returning true must claim the error");
});

test("an unknown request id is left unclaimed for the generic surfaces", () => {
  const reg = createActionRequestRegistry(fakeClock());
  assert.strictEqual(reg.dispatch("never-registered", "boom", {}), false);
});

test("a missing request id is left unclaimed", () => {
  const reg = createActionRequestRegistry(fakeClock());
  assert.strictEqual(reg.dispatch("", "boom", {}), false);
  assert.strictEqual(reg.dispatch(undefined, "boom", {}), false);
});

test("a handler returning false declines the claim", () => {
  const reg = createActionRequestRegistry(fakeClock());
  reg.register("req-2", () => false);
  assert.strictEqual(reg.dispatch("req-2", "boom", {}), false,
    "an explicit false must fall through to the generic surfaces");
});

test("a throwing handler fails open rather than swallowing the error", () => {
  let reported = null;
  const reg = createActionRequestRegistry({
    ...fakeClock(),
    onHandlerError: (err) => { reported = err; },
  });
  reg.register("req-3", () => { throw new Error("handler bug"); });
  assert.strictEqual(reg.dispatch("req-3", "boom", {}), false,
    "a broken handler must not silently absorb a failure");
  assert.ok(reported instanceof Error);
});

test("each error is delivered once", () => {
  const reg = createActionRequestRegistry(fakeClock());
  let calls = 0;
  reg.register("req-4", () => { calls += 1; return true; });
  reg.dispatch("req-4", "boom", {});
  reg.dispatch("req-4", "boom", {});
  assert.strictEqual(calls, 1);
});

test("registrations expire, so successful actions cannot leak handlers", () => {
  const clock = fakeClock();
  const reg = createActionRequestRegistry(clock);
  reg.register("req-5", () => true);
  assert.strictEqual(reg.pendingCount(), 1);
  clock.fire(); // the expiry timer for a request that simply succeeded
  assert.strictEqual(reg.pendingCount(), 0);
  assert.strictEqual(reg.dispatch("req-5", "late", {}), false);
});

test("unregister withdraws a request that was never sent", () => {
  const reg = createActionRequestRegistry(fakeClock());
  const unregister = reg.register("req-6", () => true);
  unregister();
  assert.strictEqual(reg.pendingCount(), 0);
  assert.strictEqual(reg.dispatch("req-6", "boom", {}), false);
});

test("re-registering an id replaces rather than stacks", () => {
  const clock = fakeClock();
  const reg = createActionRequestRegistry(clock);
  let which = "";
  reg.register("req-7", () => { which = "first"; return true; });
  reg.register("req-7", () => { which = "second"; return true; });
  assert.strictEqual(reg.pendingCount(), 1);
  assert.strictEqual(clock.count(), 1, "the replaced registration's timer must be cleared");
  reg.dispatch("req-7", "boom", {});
  assert.strictEqual(which, "second");
});

// --- Socket frame normalization -----------------------------------------

test("an error frame carries request_id through normalization", () => {
  const dispatcher = socketModule.createDispatcher({ onError: () => {} });
  const result = dispatcher.dispatch(
    JSON.stringify({ type: "error", error: "not_your_turn", request_id: "req-8" }));
  assert.strictEqual(result.kind, "error");
  assert.strictEqual(result.error, "not_your_turn");
  assert.strictEqual(result.requestId, "req-8",
    "without this the registry has no id to match a refusal against");
});

test("an error frame with no request_id normalizes to an empty id", () => {
  const dispatcher = socketModule.createDispatcher({ onError: () => {} });
  const result = dispatcher.dispatch(JSON.stringify({ type: "error", error: "action_denied" }));
  assert.strictEqual(result.kind, "error");
  assert.strictEqual(result.requestId, "");
});

// --- session-sync claim behaviour ---------------------------------------

function sessionSyncHarness(handleActionError) {
  const calls = { diceTrayError: 0, stageStatus: [], movement: [], chat: [] };
  const sync = createSessionSync({
    handleActionError,
    handleDiceTrayError: () => { calls.diceTrayError += 1; },
    setStageStatus: (t) => calls.stageStatus.push(t),
    setMovementLine: (t) => calls.movement.push(t),
    appendSystemChatNotice: (t) => calls.chat.push(t),
  });
  return { sync, calls };
}

test("a claimed error stays out of the stage status, movement line and chat", () => {
  const { sync, calls } = sessionSyncHarness(() => true);
  sync.handleSocketMessage({ kind: "error", error: "not_your_turn", requestId: "req-9" });
  assert.strictEqual(calls.diceTrayError, 0);
  assert.deepStrictEqual(calls.stageStatus, []);
  assert.deepStrictEqual(calls.movement, []);
  assert.deepStrictEqual(calls.chat, [],
    "a privately-handled failure must not be announced to the room");
});

test("an unclaimed error still reaches the generic surfaces", () => {
  const { sync, calls } = sessionSyncHarness(() => false);
  sync.handleSocketMessage({ kind: "error", error: "showing_closed", requestId: "req-10" });
  assert.strictEqual(calls.diceTrayError, 1);
  assert.strictEqual(calls.stageStatus.length, 1);
  assert.strictEqual(calls.movement.length, 1);
  assert.strictEqual(calls.chat.length, 1);
});

test("an error with no request id behaves exactly as it did before 88B", () => {
  let consulted = false;
  const { sync, calls } = sessionSyncHarness(() => { consulted = true; return true; });
  sync.handleSocketMessage({ kind: "error", error: "action_denied" });
  assert.strictEqual(consulted, false, "no id means there is nobody to ask");
  assert.strictEqual(calls.diceTrayError, 1);
  assert.strictEqual(calls.stageStatus.length, 1);
});

const failed = results.filter((r) => !r.ok);
console.log(`\n${results.length - failed.length}/${results.length} passed`);
process.exit(failed.length ? 1 : 0);

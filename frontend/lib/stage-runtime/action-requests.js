// Kernel 88B: per-request error routing for stage actions.
//
// Actions travel one way over the socket. The only reply a failed one
// produces is a {type:"error", request_id} frame, and before this that frame
// was handled exclusively by generic surfaces -- the stage status line, a
// system chat notice, the dice tray. A module that sent its own action could
// therefore report success and never learn the server had refused it. The
// Socio Player HUD's mechanic roll was the motivating case: "Roll sent.",
// then silence.
//
// A module registers the request_id it is about to send. If an error for that
// id arrives, its handler runs and claims the error, and the generic surfaces
// stand down -- a refusal belongs next to the control the person pressed, and
// announcing a private failure to the whole room is worse than useless.
//
// Registrations expire on a timer, because the overwhelmingly common outcome
// is success, which produces no reply at all. Without expiry every successful
// action would leak a handler for the life of the page.
(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryStageActionRequests = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  const DEFAULT_WINDOW_MS = 15000;
  const MIN_WINDOW_MS = 1000;

  function createActionRequestRegistry(options = {}) {
    const setTimeoutFn = options.setTimeoutFn || ((cb, ms) => setTimeout(cb, ms));
    const clearTimeoutFn = options.clearTimeoutFn || ((id) => clearTimeout(id));
    const onHandlerError = options.onHandlerError || ((err) => {
      if (typeof console !== "undefined") console.error("action error handler failed", err);
    });
    const defaultWindowMs = Number(options.windowMs) > 0 ? Number(options.windowMs) : DEFAULT_WINDOW_MS;

    const pending = new Map();

    function forget(id) {
      const entry = pending.get(id);
      if (!entry) return;
      clearTimeoutFn(entry.timer);
      pending.delete(id);
    }

    // Returns an unregister function so a caller that fails to send (socket
    // closed, say) can withdraw immediately rather than leaving a handler
    // waiting for a reply that can never come.
    function register(requestId, handler, timeoutMs) {
      const id = String(requestId || "").trim();
      if (!id || typeof handler !== "function") return () => {};
      forget(id); // re-registering an id replaces it rather than stacking
      const window = Math.max(MIN_WINDOW_MS, Number(timeoutMs) || defaultWindowMs);
      const timer = setTimeoutFn(() => { pending.delete(id); }, window);
      pending.set(id, { handler, timer });
      return () => forget(id);
    }

    // Returns true when a module claimed the error, meaning it has shown the
    // failure itself and the generic surfaces should stay quiet. An unknown
    // id, a missing id, or a handler that throws or explicitly returns false
    // all leave the error unclaimed -- failing open is the safe direction,
    // since the cost is a duplicate message rather than a silent failure.
    function dispatch(requestId, errorText, message) {
      const id = String(requestId || "").trim();
      if (!id) return false;
      const entry = pending.get(id);
      if (!entry) return false;
      clearTimeoutFn(entry.timer);
      pending.delete(id);
      try {
        return entry.handler(errorText, message) !== false;
      } catch (err) {
        onHandlerError(err);
        return false;
      }
    }

    return {
      register,
      dispatch,
      // Test/diagnostic only -- how many replies are still being waited on.
      pendingCount: () => pending.size,
    };
  }

  return { createActionRequestRegistry };
});

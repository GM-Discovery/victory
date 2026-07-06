(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryCatharsisSocket = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  function normalizeAction(action) {
    const source = action && typeof action === "object" ? action : {};
    const payload = source.payload && typeof source.payload === "object" ? source.payload : {};
    const target = source.target && typeof source.target === "object" ? source.target : {};
    const visibility = source.visibility && typeof source.visibility === "object" ? source.visibility : {};
    const normalizedPayload = { ...payload };
    if (normalizedPayload.nameplateVisible !== undefined && normalizedPayload.nameplate_visible === undefined) {
      normalizedPayload.nameplate_visible = normalizedPayload.nameplateVisible;
    }
    if (normalizedPayload.nameplate_visible !== undefined && normalizedPayload.nameplateVisible === undefined) {
      normalizedPayload.nameplateVisible = normalizedPayload.nameplate_visible;
    }
    return {
      ...source,
      type: String(source.type || "").trim(),
      target,
      payload: normalizedPayload,
      visibility,
    };
  }

  function createDispatcher(options = {}) {
    const parseMessage = typeof options.parseMessage === "function"
      ? options.parseMessage
      : (raw) => JSON.parse(raw);
    const state = options.state || null;
    const handlers = {
      onSnapshot: typeof options.onSnapshot === "function" ? options.onSnapshot : () => {},
      onAction: typeof options.onAction === "function" ? options.onAction : () => {},
      onFocusPing: typeof options.onFocusPing === "function" ? options.onFocusPing : () => {},
      onPong: typeof options.onPong === "function" ? options.onPong : () => {},
      onError: typeof options.onError === "function" ? options.onError : () => {},
      onUnknown: typeof options.onUnknown === "function" ? options.onUnknown : () => {},
    };
    let boundSocket = null;
    let boundHandler = null;

    function dispatch(raw) {
      let msg;
      try {
        msg = typeof raw === "string" ? parseMessage(raw) : raw;
      } catch (error) {
        return { kind: "malformed", error };
      }

      if (!msg || typeof msg !== "object") {
        return { kind: "unknown" };
      }

      if (msg.type === "snapshot") {
        const snapshot = msg.data || msg.snapshot || msg.payload || {};
        handlers.onSnapshot(snapshot, msg);
        return { kind: "snapshot", snapshot, message: msg };
      }

      if (msg.type === "pong") {
        handlers.onPong(msg);
        return { kind: "pong", message: msg };
      }

      if (msg.type === "venue/focus_ping") {
        const data = msg.data || {};
        handlers.onFocusPing(data, msg);
        return { kind: "focus_ping", data, message: msg };
      }

      if (msg.type === "showing/update") {
        return { kind: "showing_update", message: msg };
      }

      if (msg.type === "venue/update") {
        return { kind: "venue_update", message: msg };
      }

      if (msg.type === "character/projection_updated") {
        return {
          kind: "character_projection_updated",
          characterId: String(msg.character_id || ""),
          projectionVersion: String(msg.projection_version || ""),
          changedDimensions: Array.isArray(msg.changed_dimensions) ? msg.changed_dimensions : [],
          message: msg,
        };
      }

      if (msg.type === "error") {
        handlers.onError(String(msg.error || "action_denied"), msg);
        return { kind: "error", error: String(msg.error || "action_denied"), message: msg };
      }

      if (msg.type === "action") {
        const action = normalizeAction(msg.data || msg.payload || {});
        state?.applyEvent?.(action, { event: msg });
        handlers.onAction(action, msg);
        return { kind: "action", action, message: msg };
      }

      handlers.onUnknown(msg);
      return { kind: "unknown", message: msg };
    }

    function bind(socket) {
      if (!socket || typeof socket.addEventListener !== "function") {
        return () => {};
      }
      if (boundSocket === socket && boundHandler) {
        return () => {
          if (boundSocket === socket && boundHandler) {
            socket.removeEventListener("message", boundHandler);
            boundSocket = null;
            boundHandler = null;
          }
        };
      }
      if (boundSocket && boundHandler) {
        boundSocket.removeEventListener("message", boundHandler);
      }
      boundSocket = socket;
      boundHandler = (event) => {
        dispatch(event?.data);
      };
      socket.addEventListener("message", boundHandler);
      return () => {
        if (boundSocket === socket && boundHandler) {
          socket.removeEventListener("message", boundHandler);
          boundSocket = null;
          boundHandler = null;
        }
      };
    }

    return {
      dispatch,
      bind,
    };
  }

  return {
    createDispatcher,
    normalizeAction,
  };
});

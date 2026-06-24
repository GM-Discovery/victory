(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryFirstTheaterSocketController = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  function createSocketController(deps = {}) {
    const getSessionId = typeof deps.getSessionId === "function" ? deps.getSessionId : () => "";
    const getStageCamera = typeof deps.getStageCamera === "function" ? deps.getStageCamera : () => null;
    const getLastStagePoint = typeof deps.getLastStagePoint === "function" ? deps.getLastStagePoint : () => null;
    const getStageSize = typeof deps.getStageSize === "function" ? deps.getStageSize : () => ({ width: 0, height: 0 });
    const getLocation = typeof deps.getLocation === "function" ? deps.getLocation : () => ({ protocol: "http:", host: "localhost" });
    const getWebSocketCtor = typeof deps.getWebSocketCtor === "function" ? deps.getWebSocketCtor : () => (typeof WebSocket !== "undefined" ? WebSocket : null);
    const getPerformanceNow = typeof deps.getPerformanceNow === "function" ? deps.getPerformanceNow : () => Date.now();
    const setStageStatus = typeof deps.setStageStatus === "function" ? deps.setStageStatus : () => {};
    const setMovementLine = typeof deps.setMovementLine === "function" ? deps.setMovementLine : () => {};
    const updateShellMetaPresentation = typeof deps.updateShellMetaPresentation === "function" ? deps.updateShellMetaPresentation : () => {};
    const refreshWorld = typeof deps.refreshWorld === "function" ? deps.refreshWorld : async () => {};
    const onMessage = typeof deps.onMessage === "function" ? deps.onMessage : () => {};
    const onConnect = typeof deps.onConnect === "function" ? deps.onConnect : () => {};
    const onClose = typeof deps.onClose === "function" ? deps.onClose : () => {};
    const onError = typeof deps.onError === "function" ? deps.onError : () => {};
    const onReadyToRetry = typeof deps.onReadyToRetry === "function" ? deps.onReadyToRetry : () => {};
    const onPing = typeof deps.onPing === "function" ? deps.onPing : () => {};
    const onFocusPing = typeof deps.onFocusPing === "function" ? deps.onFocusPing : () => {};
    const onPong = typeof deps.onPong === "function" ? deps.onPong : () => {};

    let ws = null;
    let pendingSocketActions = [];
    let socketReconnectTimer = null;
    let socketReconnectDelay = 1000;
    let pendingPingStartedAt = null;
    let lastPingMs = null;

    function canSendStageAction() {
      return !!ws && ws.readyState === getWebSocketCtor()?.OPEN;
    }

    function scheduleSocketReconnect() {
      if (socketReconnectTimer) return;
      const setTimeoutFn = deps.setTimeoutFn || ((cb, delay) => window.setTimeout(cb, delay));
      socketReconnectTimer = setTimeoutFn(() => {
        socketReconnectTimer = null;
        socketReconnectDelay = Math.min(socketReconnectDelay * 2, 10000);
        onReadyToRetry();
        connectSocket();
      }, socketReconnectDelay);
    }

    function queuePendingSocketAction(type, extra = {}) {
      pendingSocketActions.push({
        type,
        extra: { ...extra },
      });
    }

    function flushPendingSocketActions() {
      if (!canSendStageAction() || pendingSocketActions.length === 0) return;

      const queued = pendingSocketActions;
      pendingSocketActions = [];
      for (const item of queued) {
        ws.send(JSON.stringify({
          type: item.type,
          session_id: getSessionId(),
          ...item.extra,
        }));
      }
    }

    function sendAction(type, extra = {}) {
      if (!canSendStageAction()) {
        if (ws && ws.readyState === getWebSocketCtor()?.CONNECTING && getSessionId()) {
          queuePendingSocketAction(type, extra);
          return true;
        }
        return false;
      }

      ws.send(JSON.stringify({
        type,
        session_id: getSessionId(),
        ...extra,
      }));
      return true;
    }

    function buildFocusPingPayload() {
      const view = getStageCamera()?.getView?.() || {};
      const size = getStageSize();
      const centerWorld = getStageCamera()?.screenToWorld?.(size.width / 2, size.height / 2) || { x: 0, y: 0 };
      const focusWorld = getLastStagePoint() || centerWorld;
      return {
        venue_slug: "the-cave",
        focus_x: Math.round(Number(focusWorld.x || 0)),
        focus_y: Math.round(Number(focusWorld.y || 0)),
        camera_center_x: Math.round(Number(centerWorld.x || 0)),
        camera_center_y: Math.round(Number(centerWorld.y || 0)),
        camera_pan_x: Number(view.panX || 0),
        camera_pan_y: Number(view.panY || 0),
        camera_zoom_relative_to_fit: Number(view.zoomRelativeToFit || 1),
        event_id: `${Date.now()}-${Math.random().toString(16).slice(2, 8)}`,
      };
    }

    function sendPing() {
      if (!canSendStageAction() && !(ws && ws.readyState === getWebSocketCtor()?.CONNECTING && getSessionId())) {
        return false;
      }
      pendingPingStartedAt = getPerformanceNow();
      onPing(pendingPingStartedAt);
      return sendAction("ping");
    }

    function sendFocusPing() {
      if (!canSendStageAction() && !(ws && ws.readyState === getWebSocketCtor()?.CONNECTING && getSessionId())) {
        return false;
      }
      if (!getStageCamera()) {
        setStageStatus("Camera is not ready.");
        return false;
      }
      const sent = sendAction("venue/focus_ping", buildFocusPingPayload());
      if (sent) {
        setStageStatus("Focused venue.");
        setMovementLine("Focused venue.");
      }
      return sent;
    }

    function connectSocket() {
      if (socketReconnectTimer) {
        const clearTimeoutFn = deps.clearTimeoutFn || ((value) => window.clearTimeout(value));
        clearTimeoutFn(socketReconnectTimer);
        socketReconnectTimer = null;
      }
      const protocol = getLocation().protocol === "https:" ? "wss:" : "ws:";
      const WebSocketCtor = getWebSocketCtor();
      if (!WebSocketCtor) {
        setStageStatus("WebSocket unavailable.");
        return null;
      }

      ws = new WebSocketCtor(`${protocol}//${getLocation().host}/ws/the-cave`);

      ws.addEventListener("open", () => {
        socketReconnectDelay = 1000;
        setStageStatus("Connected to the Cave WebSocket.");
        updateShellMetaPresentation();
        flushPendingSocketActions();
        onConnect();
      });

      ws.addEventListener("close", () => {
        setStageStatus("WebSocket closed.");
        setMovementLine("Socket closed");
        updateShellMetaPresentation();
        pendingSocketActions = [];
        onClose();
        scheduleSocketReconnect();
      });

      ws.addEventListener("error", () => {
        setStageStatus("WebSocket error.");
        updateShellMetaPresentation();
        onError();
        scheduleSocketReconnect();
      });

      ws.addEventListener("message", async (event) => {
        const raw = event?.data;
        const result = onMessage(raw, { ws, pendingPingStartedAt, setPendingPingStartedAt(value) {
          pendingPingStartedAt = value;
        }, setLastPingMs(value) {
          lastPingMs = value;
        }, getLastPingMs() {
          return lastPingMs;
        }, refreshWorld });
        if (result?.kind === "pong" && pendingPingStartedAt !== null) {
          lastPingMs = Math.max(0, Math.round(getPerformanceNow() - Number(pendingPingStartedAt)));
          pendingPingStartedAt = null;
          updateShellMetaPresentation();
          onPong(lastPingMs);
        }
      });

      return ws;
    }

    return {
      canSendStageAction,
      scheduleSocketReconnect,
      queuePendingSocketAction,
      flushPendingSocketActions,
      sendAction,
      buildFocusPingPayload,
      sendPing,
      sendFocusPing,
      connectSocket,
      getWebSocket: () => ws,
      getLastPingMs: () => lastPingMs,
    };
  }

  return { createSocketController };
});

// Lightweight client for the Storyboards live-event websocket (Kernel 80
// spec 8). Watches exactly one board_id at a time; on open (and on every
// reconnect) sends watch_board and expects a fresh {"type":"snapshot"} in
// reply -- never a replay of missed events, matching the server contract
// documented in backend/internal/storyboards/ws.go.
function watchStoryboard(boardId, handlers) {
  if (!boardId) {
    return { close() {} };
  }

  let closed = false;
  let socket = null;
  let reconnectTimer = null;

  function connect() {
    if (closed) return;
    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    socket = new WebSocket(`${proto}//${window.location.host}/ws/storyboards`);

    socket.addEventListener("open", () => {
      socket.send(JSON.stringify({ type: "watch_board", board_id: boardId }));
    });

    socket.addEventListener("message", (event) => {
      let msg;
      try {
        msg = JSON.parse(event.data);
      } catch (error) {
        return;
      }
      if (!msg || !msg.type) return;
      if (msg.type === "snapshot") {
        handlers.onSnapshot && handlers.onSnapshot(msg.data);
        return;
      }
      if (msg.type === "error") {
        handlers.onError && handlers.onError(msg.error);
        return;
      }
      if (msg.type.indexOf("storyboard/") === 0) {
        handlers.onEvent && handlers.onEvent(msg.type, msg.data);
      }
    });

    socket.addEventListener("close", () => {
      if (closed) return;
      reconnectTimer = setTimeout(connect, 2000);
    });

    socket.addEventListener("error", () => {
      try { socket.close(); } catch (error) { /* already closing */ }
    });
  }

  connect();

  return {
    close() {
      closed = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      if (socket) socket.close();
    },
  };
}

// Lightweight client for the player-profile invalidation websocket
// (Kernel 61A §9). Owner tabs (My Face, Workbook) and cross-user viewer
// tabs (view.html) all use this the same way: watch one profile_id, get
// told to refetch when it changes. There is no separate "owner channel" --
// the server (backend/internal/network/player_profile_invalidation.go)
// only ever sends an opaque {profile_id, projection_version}, never facts.
function watchPlayerProfile(profileId, onUpdate) {
  if (!profileId) {
    return { close() {} };
  }

  let closed = false;
  let socket = null;
  let lastVersion = null;
  let reconnectTimer = null;

  function connect() {
    if (closed) return;
    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    socket = new WebSocket(`${proto}//${window.location.host}/ws/player-profile`);

    socket.addEventListener("open", () => {
      socket.send(JSON.stringify({ type: "watch_profile", profile_id: profileId }));
    });

    socket.addEventListener("message", (event) => {
      let msg;
      try {
        msg = JSON.parse(event.data);
      } catch (error) {
        return;
      }
      if (!msg || msg.type !== "player_profile/projection_updated") return;
      if (msg.profile_id !== profileId) return;
      if (lastVersion !== null && msg.projection_version === lastVersion) return;
      lastVersion = msg.projection_version;
      onUpdate(msg);
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

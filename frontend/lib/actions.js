export function createActionClient({ getWS, getSessionId, getActorId }) {
  function send(type, payload = {}) {
    const ws = getWS();
    if (!ws || ws.readyState !== WebSocket.OPEN) return false;

    ws.send(JSON.stringify({
      type,
      session_id: getSessionId(),
      ...payload
    }));

    return true;
  }

  return {
    speak(text) {
      return send("perform/speak", { text });
    },

    react(kind) {
      return send("react/emote", { kind });
    },

    reveal(element_slug, layer = "audience") {
      return send("act/reveal_element", { element_slug, layer });
    },

    hide(element_slug, layer = "audience") {
      return send("act/hide_element", { element_slug, layer });
    }
  };
}

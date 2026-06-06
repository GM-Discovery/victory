(function () {
  function normalizeMicCommand(rawText) {
    const raw = String(rawText || "").trim();
    if (!/^\/mic(\s+|$)/i.test(raw)) {
      return null;
    }

    const tail = raw.replace(/^\/mic\s*/i, "").trim().toLowerCase();
    if (!tail) {
      return "";
    }

    switch (tail) {
      case "hot":
      case "on":
      case "start":
      case "off":
      case "status":
        return tail;
      default:
        return "";
    }
  }

  function normalizeSessionCommand(rawText) {
    const raw = String(rawText || "").trim();
    if (!/^\/session(\s+|$)/i.test(raw)) {
      return null;
    }

    const tail = raw.replace(/^\/session\s*/i, "").trim().toLowerCase();
    if (!tail) {
      return "status";
    }

    switch (tail) {
      case "status":
      case "start":
      case "end":
        return tail;
      default:
        return "";
    }
  }

  function emitMicRefresh(venueSlug) {
    const slug = String(venueSlug || "").trim();
    if (!slug) return;
    window.dispatchEvent(new CustomEvent("victory:discord-mic-refresh", {
      detail: { venueSlug: slug },
    }));
  }

  async function sendMicCommand(venueSlug, rawText) {
    const command = normalizeMicCommand(rawText);
    if (command === null) {
      return { handled: false };
    }

    if (!command) {
      return {
        handled: true,
        ok: false,
        message: "Use /mic hot, /mic off, or /mic status.",
      };
    }

    const response = await fetch("/api/discord/mic/control", {
      method: "POST",
      credentials: "include",
      cache: "no-store",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        venue_slug: String(venueSlug || "").trim(),
        command,
      }),
    });

    const payload = await response.json().catch(() => null);
    const error = payload?.data?.error || payload?.error || `HTTP ${response.status}`;

    if (!response.ok || !payload?.ok) {
      return {
        handled: true,
        ok: false,
        message: payload?.data?.message || payload?.data?.detail || error,
      };
    }

    emitMicRefresh(venueSlug);
    return {
      handled: true,
      ok: true,
      command: String(payload?.data?.command || command).trim(),
      message: String(payload?.data?.message || "").trim(),
      data: payload.data || {},
    };
  }

  async function sendSessionCommand(venueSlug, rawText) {
    const command = normalizeSessionCommand(rawText);
    if (command === null) {
      return { handled: false };
    }

    if (!command) {
      return {
        handled: true,
        ok: false,
        message: "Use /session start, /session end, or /session status.",
      };
    }

    const response = await fetch("/api/session/control", {
      method: "POST",
      credentials: "include",
      cache: "no-store",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        venue_slug: String(venueSlug || "").trim(),
        command,
      }),
    });

    const payload = await response.json().catch(() => null);
    const error = payload?.data?.error || payload?.error || `HTTP ${response.status}`;

    if (!response.ok || !payload?.ok) {
      return {
        handled: true,
        ok: false,
        message: payload?.data?.message || payload?.data?.detail || error,
      };
    }

    return {
      handled: true,
      ok: true,
      command: String(payload?.data?.command || command).trim(),
      state: String(payload?.data?.state || "").trim(),
      message: String(payload?.data?.message || "").trim(),
      data: payload.data || {},
    };
  }

  window.VictoryMicChat = {
    normalizeMicCommand,
    normalizeSessionCommand,
    sendMicCommand,
    sendSessionCommand,
    emitMicRefresh,
  };
})();

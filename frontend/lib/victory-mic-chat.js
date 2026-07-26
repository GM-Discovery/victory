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

  // "/showtime <code>", "/showtime <code> status", "/showtime <code> end" --
  // unlike /mic and /session (single-word verbs), showtime's first token is
  // always the Show's short code, with an optional trailing action verb.
  function normalizeShowtimeCommand(rawText) {
    const raw = String(rawText || "").trim();
    if (!/^\/showtime(\s+|$)/i.test(raw)) {
      return null;
    }

    const tail = raw.replace(/^\/showtime\s*/i, "").trim();
    if (!tail) {
      return { shortCode: "", action: "start" };
    }

    const parts = tail.split(/\s+/);
    const shortCode = parts[0];
    const verb = (parts[1] || "start").toLowerCase();
    if (!["start", "status", "end"].includes(verb)) {
      return { shortCode: "", action: "start" };
    }

    return { shortCode, action: verb };
  }

  async function sendShowtimeCommand(venueSlug, rawText) {
    const parsed = normalizeShowtimeCommand(rawText);
    if (parsed === null) {
      return { handled: false };
    }

    if (!parsed.shortCode) {
      return {
        handled: true,
        ok: false,
        message: "Use /showtime <code>, /showtime <code> status, or /showtime <code> end.",
      };
    }

    const response = await fetch("/api/showtime/control", {
      method: "POST",
      credentials: "include",
      cache: "no-store",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        short_code: parsed.shortCode,
        action: parsed.action,
        venue_slug: String(venueSlug || "").trim(),
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
      message: String(payload?.data?.message || "").trim(),
      data: payload.data || {},
    };
  }

  function normalizeJournalCommand(rawText) {
    const raw = String(rawText || "").trim();
    if (!/^\/journal(\s+|$)/i.test(raw)) {
      return null;
    }

    const tail = raw.replace(/^\/journal\s*/i, "").trim();
    return tail;
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

  async function sendJournalCommand(rawText, options = {}) {
    const body = normalizeJournalCommand(rawText);
    if (body === null) {
      return { handled: false };
    }

    if (!body) {
      return {
        handled: true,
        ok: false,
        message: "Use /journal <text>.",
      };
    }

    const response = await fetch("/api/character-journals", {
      method: "POST",
      credentials: "include",
      cache: "no-store",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        body,
        visibility: String(options.visibility || "private").trim() || "private",
      }),
    });

    const payload = await response.json().catch(() => null);
    const error = payload?.data?.error || payload?.error || `HTTP ${response.status}`;

    if (!response.ok || !payload?.ok) {
      return {
        handled: true,
        ok: false,
        message: payload?.data?.error === "no_active_character"
          ? "No active character. Load a character before writing a journal entry."
          : payload?.data?.message || payload?.data?.detail || error,
      };
    }

    return {
      handled: true,
      ok: true,
      message: "Journal entry saved privately.",
      data: payload.data || {},
    };
  }

  window.VictoryMicChat = {
    normalizeMicCommand,
    normalizeSessionCommand,
    normalizeShowtimeCommand,
    normalizeJournalCommand,
    sendMicCommand,
    sendSessionCommand,
    sendShowtimeCommand,
    sendJournalCommand,
    emitMicRefresh,
  };
})();

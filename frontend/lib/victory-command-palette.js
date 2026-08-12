(function () {
  const CATALOGUE_TTL_MS = 30000;
  const catalogueCache = new Map(); // key: `${venueSlug}::${sessionId}` -> { at, data }

  function isLiteralEscape(text) {
    return /^\/\//.test(String(text || ""));
  }

  function isOpenTriggerText(text) {
    const raw = String(text || "");
    const trimmedStart = raw.replace(/^\s+/, "");
    return trimmedStart.startsWith("/") && !isLiteralEscape(trimmedStart);
  }

  function splitFirstWord(str) {
    const s = String(str || "");
    const match = s.match(/^\s*(\S+)\s*([\s\S]*)$/);
    if (!match) return ["", ""];
    return [match[1], match[2]];
  }

  // parseSlashCommand extracts {path, args} from a raw composer string
  // beginning with "/". Free-text tails (biography/quote/journal/ooc bodies)
  // are kept as a single opaque array element so interior whitespace is
  // preserved verbatim rather than lost to re-joining split tokens.
  function parseSlashCommand(text) {
    const raw = String(text || "").trim();
    if (!raw.startsWith("/") || isLiteralEscape(raw)) return null;

    const withoutSlash = raw.slice(1);
    const [pathWord, rest] = splitFirstWord(withoutSlash);
    const path = pathWord.toLowerCase();
    if (!path) return null;

    switch (path) {
      case "mic":
      case "session": {
        const [sub] = splitFirstWord(rest);
        return { path, args: sub ? [sub.toLowerCase()] : [] };
      }
      case "char": {
        const [sub, tail] = splitFirstWord(rest);
        if (!sub) return { path, args: [] };
        const subLower = sub.toLowerCase();
        if (subLower === "set") {
          const [field, value] = splitFirstWord(tail);
          return { path, args: ["set", field.toLowerCase(), value] };
        }
        if (subLower === "add") {
          // "/char add skill <name>" or
          // "/char add skill --custom --name <n> --description <d> --attribute <a>" --
          // each word stays a separate arg so the server can split --flags apart.
          const [addSub, addTail] = splitFirstWord(tail);
          if (addSub.toLowerCase() !== "skill") return { path, args: ["add"] };
          const skillArgs = addTail ? addTail.split(/\s+/).filter(Boolean) : [];
          return { path, args: ["add", "skill", ...skillArgs] };
        }
        if (subLower === "advance") {
          return { path, args: ["advance", tail] };
        }
        return { path, args: [subLower] };
      }
      case "bio":
      case "quote": {
        const [sub, tail] = splitFirstWord(rest);
        if (sub.toLowerCase() === "set") return { path, args: ["set", tail] };
        const body = rest.trim();
        return body ? { path, args: ["set", body] } : { path, args: [] };
      }
      case "journal": {
        const [sub, tail] = splitFirstWord(rest);
        if (sub.toLowerCase() === "recent") return { path, args: ["recent"] };
        if (sub.toLowerCase() === "add") return { path, args: ["add", tail] };
        return { path, args: [] };
      }
      case "ooc":
      case "ic":
        return { path, args: [rest] };
      case "help":
        return { path, args: rest ? [rest.trim()] : [] };
      default:
        return { path, args: rest ? [rest] : [] };
    }
  }

  async function fetchCatalogue(venueSlug, sessionId, { force = false } = {}) {
    const key = `${venueSlug || ""}::${sessionId || ""}`;
    const cached = catalogueCache.get(key);
    if (!force && cached && Date.now() - cached.at < CATALOGUE_TTL_MS) {
      return cached.data;
    }
    try {
      const params = new URLSearchParams();
      if (venueSlug) params.set("venue", venueSlug);
      if (sessionId) params.set("session_id", sessionId);
      const response = await fetch(`/api/commands/available?${params.toString()}`, {
        method: "GET",
        credentials: "include",
        cache: "no-store",
      });
      const payload = await response.json().catch(() => null);
      if (!response.ok || !payload?.ok) return cached ? cached.data : null;
      const data = payload.data || null;
      catalogueCache.set(key, { at: Date.now(), data });
      return data;
    } catch (error) {
      return cached ? cached.data : null;
    }
  }

  async function execute({ path, args, venueSlug, sessionId, idempotencyKey }) {
    const response = await fetch("/api/commands/execute", {
      method: "POST",
      credentials: "include",
      cache: "no-store",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        path,
        args: args || [],
        venue_slug: venueSlug || "",
        session_id: sessionId || "",
        idempotency_key: idempotencyKey || newIdempotencyKey(),
      }),
    });
    const payload = await response.json().catch(() => null);
    return { ok: Boolean(response.ok && payload?.ok), status: response.status, data: payload?.data, error: payload?.error };
  }

  function newIdempotencyKey() {
    if (window.crypto?.randomUUID) return window.crypto.randomUUID();
    return `cmd-${Date.now()}-${Math.random().toString(16).slice(2)}`;
  }

  function matchesFilter(cmd, needle) {
    if (!needle) return true;
    const haystacks = [cmd.path, cmd.title, cmd.description, ...(cmd.aliases || [])];
    return haystacks.some((h) => String(h || "").toLowerCase().includes(needle));
  }

  function ensureStyles() {
    if (document.getElementById("victory-command-palette-style")) return;
    const style = document.createElement("style");
    style.id = "victory-command-palette-style";
    style.textContent = `
      .victory-command-palette {
        position: absolute;
        z-index: 1000;
        min-width: 260px;
        max-width: 420px;
        max-height: 260px;
        overflow-y: auto;
        background: var(--victory-palette-bg, #1b1b22);
        color: var(--victory-palette-fg, #f2f0e8);
        border: 1px solid rgba(255,255,255,0.15);
        border-radius: 8px;
        box-shadow: 0 8px 24px rgba(0,0,0,0.4);
        font-size: 13px;
        padding: 4px;
      }
      .victory-command-palette__item {
        padding: 6px 8px;
        border-radius: 6px;
        cursor: pointer;
        display: flex;
        flex-direction: column;
        gap: 2px;
      }
      .victory-command-palette__item[aria-selected="true"] {
        background: rgba(255,255,255,0.12);
      }
      .victory-command-palette__usage {
        font-family: monospace;
        opacity: 0.85;
      }
      .victory-command-palette__desc {
        opacity: 0.7;
        font-size: 12px;
      }
      .victory-command-palette__target {
        padding: 4px 8px;
        font-size: 11px;
        opacity: 0.6;
        border-bottom: 1px solid rgba(255,255,255,0.1);
        margin-bottom: 4px;
      }
      .victory-command-palette__empty {
        padding: 8px;
        opacity: 0.6;
      }
    `;
    document.head.appendChild(style);
  }

  // attach wires an <input>/<textarea> composer element to the palette.
  // options: { venueSlug, getSessionId(), onDispatch({path, args, raw}) }
  // onDispatch is called when the user selects a command that should
  // execute immediately (navigation) or be handed off to the caller's own
  // send flow. attach() itself never calls sendAction/fetch for chat --
  // the caller's composer keeps owning that.
  function attach(inputEl, options = {}) {
    if (!inputEl) return { destroy() {} };
    ensureStyles();

    const venueSlug = options.venueSlug || "";
    const getSessionId = typeof options.getSessionId === "function" ? options.getSessionId : () => "";

    let container = null;
    let items = [];
    let selectedIndex = -1;
    let open = false;

    function close() {
      open = false;
      if (container) {
        container.remove();
        container = null;
      }
      selectedIndex = -1;
      items = [];
    }

    function renderTargetLine(catalogue) {
      const active = catalogue?.active_character;
      if (!active) return null;
      const name = active.name || active.display_name || "Unnamed character";
      const line = document.createElement("div");
      line.className = "victory-command-palette__target";
      line.textContent = `Target: ${name}`;
      return line;
    }

    function renderItems(filtered) {
      container.querySelectorAll(".victory-command-palette__item").forEach((el) => el.remove());
      const targetLine = container.querySelector(".victory-command-palette__target");
      if (targetLine) targetLine.remove();

      if (!filtered.length) {
        const empty = document.createElement("div");
        empty.className = "victory-command-palette__empty";
        empty.textContent = "No matching commands.";
        container.appendChild(empty);
        return;
      }

      filtered.forEach((cmd, index) => {
        const item = document.createElement("div");
        item.className = "victory-command-palette__item";
        item.setAttribute("role", "option");
        item.setAttribute("aria-selected", index === selectedIndex ? "true" : "false");
        const usage = document.createElement("div");
        usage.className = "victory-command-palette__usage";
        usage.textContent = cmd.usage || `/${cmd.path}`;
        const desc = document.createElement("div");
        desc.className = "victory-command-palette__desc";
        desc.textContent = cmd.description || "";
        item.appendChild(usage);
        item.appendChild(desc);
        item.addEventListener("mousedown", (event) => {
          event.preventDefault();
          selectItem(cmd);
        });
        container.appendChild(item);
      });
    }

    function selectItem(cmd) {
      const template = cmd.usage || `/${cmd.path}`;
      inputEl.value = template.endsWith(" ") ? template : `${template} `;
      inputEl.focus();
      close();
    }

    async function open_() {
      const value = inputEl.value;
      if (!isOpenTriggerText(value)) {
        close();
        return;
      }
      const catalogue = await fetchCatalogue(venueSlug, getSessionId());
      if (!isOpenTriggerText(inputEl.value)) {
        // input changed while awaiting fetch
        return;
      }
      if (!catalogue || !Array.isArray(catalogue.commands)) {
        // Graceful fallback: raw commands still work, palette just stays closed.
        close();
        return;
      }

      const needle = inputEl.value.replace(/^\s*\//, "").trim().toLowerCase();
      const filtered = catalogue.commands.filter((cmd) => matchesFilter(cmd, needle));

      if (!container) {
        container = document.createElement("div");
        container.className = "victory-command-palette";
        container.setAttribute("role", "listbox");
        positionContainer();
        document.body.appendChild(container);
      }
      const targetLine = renderTargetLine(catalogue);
      if (targetLine) container.appendChild(targetLine);
      items = filtered;
      selectedIndex = filtered.length ? 0 : -1;
      renderItems(filtered);
      updateSelectionHighlight();
      open = true;
    }

    function positionContainer() {
      const rect = inputEl.getBoundingClientRect();
      container.style.left = `${rect.left + window.scrollX}px`;
      container.style.top = `${rect.bottom + window.scrollY + 4}px`;
      container.style.width = `${Math.max(rect.width, 260)}px`;
    }

    function updateSelectionHighlight() {
      if (!container) return;
      const nodes = container.querySelectorAll(".victory-command-palette__item");
      nodes.forEach((node, index) => {
        node.setAttribute("aria-selected", index === selectedIndex ? "true" : "false");
      });
    }

    function onInput() {
      open_();
    }

    function onKeydown(event) {
      if (!open || !container) return;
      // stopImmediatePropagation on every key we consume here so the venue's
      // own "Enter sends chat" listener on this same input doesn't also fire
      // (addEventListener order, not preventDefault, is what stops that).
      if (event.key === "ArrowDown") {
        event.preventDefault();
        event.stopImmediatePropagation();
        if (items.length) selectedIndex = (selectedIndex + 1) % items.length;
        updateSelectionHighlight();
      } else if (event.key === "ArrowUp") {
        event.preventDefault();
        event.stopImmediatePropagation();
        if (items.length) selectedIndex = (selectedIndex - 1 + items.length) % items.length;
        updateSelectionHighlight();
      } else if (event.key === "Escape") {
        event.stopImmediatePropagation();
        close();
      } else if (event.key === "Enter" || event.key === "Tab") {
        if (event.key === "Enter" && inputEl.value.trim() === "/") {
          close();
          return;
        }
        if (selectedIndex >= 0 && items[selectedIndex]) {
          event.preventDefault();
          event.stopImmediatePropagation();
          selectItem(items[selectedIndex]);
        }
      }
    }

    function onBlur() {
      // Delay so a mousedown selection on the list can still register.
      setTimeout(() => close(), 120);
    }

    inputEl.addEventListener("input", onInput);
    inputEl.addEventListener("keydown", onKeydown);
    inputEl.addEventListener("blur", onBlur);

    return {
      destroy() {
        inputEl.removeEventListener("input", onInput);
        inputEl.removeEventListener("keydown", onKeydown);
        inputEl.removeEventListener("blur", onBlur);
        close();
      },
      isOpen: () => open,
    };
  }

  window.VictoryCommandPalette = {
    isOpenTriggerText,
    isLiteralEscape,
    parseSlashCommand,
    fetchCatalogue,
    execute,
    attach,
  };
})();

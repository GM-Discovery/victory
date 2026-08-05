(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryStageProgramPanel = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  // Kernel 73: a generic, venue-agnostic participant-local overlay for
  // large-format interactive content (an item, dialogue, chart,
  // instruction, or interaction) -- distinct from "Audience Program" (the
  // curated Show-Run listing). Built dynamically via document.createElement,
  // like runtime.js's existing Cue button tray, rather than depending on
  // static per-venue HTML markup -- so no venue has to hand-edit its
  // index.html to gain one. Not Catharsis-specific; Kessa's Equip Mode is
  // configuration on top of this, not a copy of it.

  function ensureStyles(doc) {
    if (!doc || !doc.head || doc.getElementById("victory-program-panel-style")) return;
    const style = doc.createElement("style");
    style.id = "victory-program-panel-style";
    style.textContent = `
      .victory-program-panel-backdrop {
        position: fixed;
        inset: 0;
        z-index: 2000;
        background: rgba(10, 8, 6, 0.72);
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 24px;
      }
      .victory-program-panel-backdrop[hidden] { display: none; }
      .victory-program-speaker {
        position: fixed;
        inset: 0;
        z-index: 2010;
        pointer-events: none;
      }
      .victory-program-speaker[hidden] { display: none; }
      .victory-program-speaker__image {
        position: absolute;
        bottom: 0;
        width: 33.333vw;
        height: 66.666vh;
        object-fit: contain;
        object-position: bottom center;
        filter: drop-shadow(0 18px 18px rgba(0, 0, 0, 0.38));
      }
      .victory-program-speaker--left .victory-program-speaker__image { left: 0; }
      .victory-program-speaker--right .victory-program-speaker__image { right: 0; }
      .victory-program-speaker--mirrored .victory-program-speaker__image { transform: scaleX(-1); }
      .victory-program-panel {
        width: min(560px, 100%);
        max-height: min(80vh, 720px);
        overflow-y: auto;
        background: #1b1610;
        color: #f0e3c8;
        border: 1px solid rgba(255, 233, 197, 0.25);
        border-radius: 14px;
        box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
        padding: 20px 22px 22px;
        font-size: 14px;
      }
      .victory-program-panel__header {
        display: flex;
        align-items: flex-start;
        gap: 12px;
        margin-bottom: 12px;
      }
      .victory-program-panel__image {
        width: 64px;
        height: 64px;
        border-radius: 10px;
        object-fit: cover;
        background: rgba(255, 255, 255, 0.06);
        flex-shrink: 0;
      }
      .victory-program-panel__title-block { flex: 1; min-width: 0; }
      .victory-program-panel__title {
        font-size: 1.15rem;
        font-weight: 700;
        margin: 0 0 4px;
      }
      .victory-program-panel__subtitle {
        opacity: 0.75;
        font-size: 0.85rem;
      }
      .victory-program-panel__close {
        background: none;
        border: 1px solid rgba(255, 233, 197, 0.3);
        color: inherit;
        border-radius: 999px;
        width: 30px;
        height: 30px;
        cursor: pointer;
        flex-shrink: 0;
      }
      .victory-program-panel__close:hover { background: rgba(255, 255, 255, 0.08); }
      .victory-program-panel__body { line-height: 1.5; }
      .victory-aftercare-fields {
        display: flex;
        flex-direction: column;
        gap: 16px;
        margin-top: 16px;
      }
      .victory-aftercare-field {
        display: flex;
        flex-direction: column;
        align-items: stretch;
        gap: 7px;
        width: 100%;
      }
      .victory-aftercare-field__prompt {
        display: block;
        font-weight: 600;
        line-height: 1.35;
      }
      .victory-aftercare-field textarea {
        display: block;
        box-sizing: border-box;
        width: 100%;
        min-height: 88px;
        resize: vertical;
        margin: 0;
        padding: 10px 12px;
        border: 1px solid rgba(255, 233, 197, 0.3);
        border-radius: 8px;
        background: rgba(10, 8, 6, 0.45);
        color: inherit;
        font: inherit;
        line-height: 1.4;
      }
      .victory-program-panel__error {
        background: rgba(200, 60, 40, 0.18);
        border: 1px solid rgba(255, 120, 100, 0.4);
        border-radius: 8px;
        padding: 8px 10px;
        margin-bottom: 10px;
        font-size: 0.85rem;
      }
      .victory-program-panel__loading {
        opacity: 0.7;
        font-style: italic;
        padding: 8px 0;
      }
      .victory-program-panel__buttons {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        margin-top: 14px;
      }
      .victory-program-panel__buttons button {
        background: rgba(255, 233, 197, 0.1);
        border: 1px solid rgba(255, 233, 197, 0.3);
        color: inherit;
        border-radius: 8px;
        padding: 8px 14px;
        cursor: pointer;
        font-size: 0.9rem;
      }
      .victory-program-panel__buttons button:hover:not(:disabled) {
        background: rgba(255, 233, 197, 0.2);
      }
      .victory-program-panel__buttons button:disabled {
        opacity: 0.5;
        cursor: default;
      }
      .victory-program-panel__buttons button.is-primary {
        background: rgba(120, 200, 140, 0.25);
        border-color: rgba(120, 220, 150, 0.5);
      }
    `;
    doc.head.appendChild(style);
  }

  function escapeHtml(value) {
    return String(value ?? "").replace(/[&<>"']/g, (ch) => ({
      "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
    }[ch]));
  }

  // createProgramPanel(deps): deps.document/deps.window default to the
  // global document/window (real browser use); tests inject fakes, matching
  // dice.js's createDiceTrayController(deps) convention.
  function createProgramPanel(deps) {
    const cfg = deps || {};
    const doc = cfg.document || (typeof document !== "undefined" ? document : null);
    if (!doc) throw new Error("createProgramPanel requires a document (global or injected)");
    ensureStyles(doc);

    let backdrop = null;
    let panel = null;
    let titleEl = null;
    let subtitleEl = null;
    let imageEl = null;
    let bodyEl = null;
    let closeButton = null;
    let speaker = null;
    let speakerImage = null;
    let onCloseCallback = null;
    let lastFocusedElement = null;

    function build() {
      if (backdrop) return;
      backdrop = doc.createElement("div");
      backdrop.className = "victory-program-panel-backdrop";
      backdrop.hidden = true;
      backdrop.setAttribute("role", "presentation");

      panel = doc.createElement("div");
      panel.className = "victory-program-panel";
      panel.setAttribute("role", "dialog");
      panel.setAttribute("aria-modal", "true");
      panel.tabIndex = -1;

      const header = doc.createElement("div");
      header.className = "victory-program-panel__header";

      imageEl = doc.createElement("img");
      imageEl.className = "victory-program-panel__image";
      imageEl.alt = "";
      imageEl.hidden = true;

      const titleBlock = doc.createElement("div");
      titleBlock.className = "victory-program-panel__title-block";
      titleEl = doc.createElement("h2");
      titleEl.className = "victory-program-panel__title";
      subtitleEl = doc.createElement("div");
      subtitleEl.className = "victory-program-panel__subtitle";
      titleBlock.appendChild(titleEl);
      titleBlock.appendChild(subtitleEl);

      closeButton = doc.createElement("button");
      closeButton.className = "victory-program-panel__close";
      closeButton.setAttribute("aria-label", "Close");
      closeButton.textContent = "×";
      closeButton.addEventListener("click", () => close());

      header.appendChild(imageEl);
      header.appendChild(titleBlock);
      header.appendChild(closeButton);

      bodyEl = doc.createElement("div");
      bodyEl.className = "victory-program-panel__body";

      panel.appendChild(header);
      panel.appendChild(bodyEl);
      backdrop.appendChild(panel);

      backdrop.addEventListener("click", (event) => {
        if (event.target === backdrop) close();
      });
      backdrop.addEventListener("keydown", (event) => {
        if (event.key === "Escape") {
          event.preventDefault();
          close();
          return;
        }
        if (event.key === "Tab" && typeof panel.querySelectorAll === "function") {
          const focusable = panel.querySelectorAll("button, [href], input, select, textarea, [tabindex]:not([tabindex='-1'])");
          if (!focusable || focusable.length === 0) return;
          const first = focusable[0];
          const last = focusable[focusable.length - 1];
          if (event.shiftKey && doc.activeElement === first) {
            event.preventDefault();
            last.focus();
          } else if (!event.shiftKey && doc.activeElement === last) {
            event.preventDefault();
            first.focus();
          }
        }
      });

      speaker = doc.createElement("div");
      speaker.className = "victory-program-speaker";
      speaker.hidden = true;
      speakerImage = doc.createElement("img");
      speakerImage.className = "victory-program-speaker__image";
      speakerImage.alt = "";
      speaker.appendChild(speakerImage);
      backdrop.appendChild(speaker);
      doc.body.appendChild(backdrop);
    }

    function open(config) {
      build();
      const cfg = config || {};
      onCloseCallback = typeof cfg.onClose === "function" ? cfg.onClose : null;
      titleEl.textContent = cfg.title || "";
      subtitleEl.textContent = cfg.subtitle || "";
      if (cfg.imageUrl) {
        imageEl.src = cfg.imageUrl;
        imageEl.hidden = false;
      } else {
        imageEl.hidden = true;
        imageEl.removeAttribute("src");
      }
      bodyEl.innerHTML = "";
      lastFocusedElement = doc.activeElement;
      backdrop.hidden = false;
      panel.focus({ preventScroll: true });
    }

    function close() {
      if (!backdrop || backdrop.hidden) return;
      backdrop.hidden = true;
      hideSpeaker();
      bodyEl.innerHTML = "";
      if (lastFocusedElement && typeof lastFocusedElement.focus === "function") {
        lastFocusedElement.focus({ preventScroll: true });
      }
      if (onCloseCallback) onCloseCallback();
    }

    function showSpeaker(config) {
      build();
      const speakerConfig = config || {};
      const imageUrl = String(speakerConfig.imageUrl || "").trim();
      if (!imageUrl) {
        hideSpeaker();
        return;
      }
      speaker.className = `victory-program-speaker victory-program-speaker--${speakerConfig.side === "right" ? "right" : "left"}${speakerConfig.mirrored ? " victory-program-speaker--mirrored" : ""}`;
      speakerImage.src = imageUrl;
      speakerImage.alt = String(speakerConfig.alt || speakerConfig.label || "");
      speaker.hidden = false;
    }

    function hideSpeaker() {
      if (!speaker) return;
      speaker.hidden = true;
      speakerImage.removeAttribute("src");
    }

    function setBody(html) {
      if (!bodyEl) return;
      bodyEl.innerHTML = html;
    }

    // updateHeader lets a caller refresh title/subtitle/image after open()
    // without resetting scroll/focus/body -- used once real data replaces
    // an initial loading placeholder.
    function updateHeader(config) {
      if (!titleEl) return;
      const cfg = config || {};
      if (cfg.title !== undefined) titleEl.textContent = cfg.title || "";
      if (cfg.subtitle !== undefined) subtitleEl.textContent = cfg.subtitle || "";
      if (cfg.imageUrl) {
        imageEl.src = cfg.imageUrl;
        imageEl.hidden = false;
      } else if (cfg.imageUrl === "") {
        imageEl.hidden = true;
        imageEl.removeAttribute("src");
      }
    }

    function setLoading(isLoading) {
      if (!bodyEl) return;
      if (isLoading) {
        bodyEl.innerHTML = `<div class="victory-program-panel__loading">Loading…</div>`;
      }
    }

    function setError(message) {
      if (!bodyEl) return;
      bodyEl.innerHTML = `<div class="victory-program-panel__error">${escapeHtml(message)}</div>`;
    }

    function isOpen() {
      return Boolean(backdrop && !backdrop.hidden);
    }

    return { open, close, setBody, setLoading, setError, isOpen, updateHeader, showSpeaker, hideSpeaker };
  }

  return { createProgramPanel, escapeHtml };
});

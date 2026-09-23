(function () {
  const LINK_ID = "back-to-map-link";
  // document.currentScript is only valid synchronously while this script
  // tag is executing -- it reads as null by the time a later
  // DOMContentLoaded callback runs, so its data-attribute must be captured
  // here, not inside mount().
  const forceFloating = document.currentScript?.dataset.backToMap === "floating";

  function installStyle() {
    if (document.getElementById("back-to-map-style")) return;
    const style = document.createElement("style");
    style.id = "back-to-map-style";
    style.textContent = `
      .back-to-map-link {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 6px;
        border-radius: 999px;
        border: 1px solid rgba(255, 255, 255, 0.16);
        padding: 10px 16px;
        min-height: 40px;
        box-sizing: border-box;
        color: inherit;
        background: rgba(255, 255, 255, 0.06);
        text-decoration: none;
        font-weight: 700;
        white-space: nowrap;
      }

      .back-to-map-link:hover,
      .back-to-map-link:focus-visible {
        background: rgba(255, 255, 255, 0.12);
      }

      /* Kernel 95 Pass 4: this used to be a permanently bold, fully
         opaque banner (heavy blur, near-black background, full-white
         text) regardless of anything else on screen -- the one piece of
         chrome that never receded. Quiet at rest, full presence on
         hover/focus, same "present on demand" language as the tray
         settings buttons and the header itself. */
      .back-to-map-link--floating {
        position: fixed;
        /* Falls back to plain 12px (4px + 8px) on venues that don't set
           --header-rendered-height at all -- only stage-runtime venues
           (Catharsis, First Theater) track it, where this button used to
           sit right on top of the header's title/status chips whenever
           the header was pinned open or simply taller than the old
           static 12px guess assumed. */
        top: calc(var(--header-rendered-height, 4px) + 8px);
        left: 12px;
        z-index: 95;
        padding: 8px 14px;
        min-height: 34px;
        font-weight: 600;
        color: rgba(255, 255, 255, 0.82);
        background: rgba(12, 14, 18, 0.32);
        backdrop-filter: blur(3px);
        opacity: 0.55;
        transition: opacity 180ms ease, background 180ms ease, color 180ms ease;
      }

      .back-to-map-link--floating:hover,
      .back-to-map-link--floating:focus-visible {
        opacity: 1;
        color: #fff;
        background: rgba(12, 14, 18, 0.72);
        backdrop-filter: blur(10px);
      }

      @media (prefers-reduced-motion: reduce) {
        .back-to-map-link--floating {
          transition-duration: 1ms;
        }
      }
    `;
    document.head.appendChild(style);
  }

  function createLink(floating) {
    const link = document.createElement("a");
    link.id = LINK_ID;
    link.className = floating ? "back-to-map-link back-to-map-link--floating" : "back-to-map-link";
    link.href = "/";
    link.setAttribute("aria-label", "Back to map");
    link.textContent = "← Back to Map";
    return link;
  }

  function mount() {
    if (document.getElementById(LINK_ID)) return;
    installStyle();

    if (forceFloating) {
      document.body.appendChild(createLink(true));
      return;
    }

    const header = document.querySelector("header.header") || document.querySelector("header") || null;
    const host = header?.querySelector(".launchbar, .toolbar, .header-right, .top-actions") || null;

    if (host) {
      host.insertBefore(createLink(false), host.firstChild);
    } else {
      document.body.appendChild(createLink(true));
    }
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", mount, { once: true });
  } else {
    mount();
  }
})();

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

      .back-to-map-link--floating {
        position: fixed;
        top: 14px;
        left: 14px;
        z-index: 95;
        color: #fff;
        background: rgba(12, 14, 18, 0.72);
        backdrop-filter: blur(10px);
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

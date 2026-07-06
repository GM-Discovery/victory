(function () {
  const BADGE_ID = "venue-account-badge";

  function displayName(data) {
    const display = String(data?.display_name || "").trim();
    const handle = String(data?.handle || "").trim();
    const generic = new Set(["web user", "webuser", "browser user", "browser", "account", "user"]);
    if (display && !generic.has(display.toLowerCase())) return display;
    if (handle) return handle;
    return "Account";
  }

  function installStyle() {
    if (document.getElementById("venue-account-badge-style")) return;
    const style = document.createElement("style");
    style.id = "venue-account-badge-style";
    style.textContent = `
      .venue-account-badge {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        border-radius: 999px;
        border: 1px solid rgba(255, 255, 255, 0.16);
        padding: 10px 14px;
        color: inherit;
        background: rgba(255, 255, 255, 0.06);
        text-decoration: none;
        font-weight: 700;
      }

      .venue-account-badge:hover,
      .venue-account-badge:focus-visible {
        background: rgba(255, 255, 255, 0.12);
      }

      .venue-account-badge__avatar {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 1.65rem;
        height: 1.65rem;
        border-radius: 50%;
        background: rgba(255, 255, 255, 0.12);
        font-size: 0.74rem;
        line-height: 1;
      }

      .venue-account-badge--floating {
        position: fixed;
        top: 14px;
        right: 14px;
        z-index: 50;
        color: #fff;
        background: rgba(12, 14, 18, 0.72);
        backdrop-filter: blur(10px);
      }
    `;
    document.head.appendChild(style);
  }

  function initials(label) {
    const words = String(label || "Account").trim().split(/\s+/).filter(Boolean);
    if (!words.length) return "A";
    if (words.length === 1) return words[0].slice(0, 2).toUpperCase();
    return `${words[0][0] || ""}${words[1][0] || ""}`.toUpperCase();
  }

  function createBadge(host) {
    const badge = document.createElement("a");
    badge.id = BADGE_ID;
    badge.className = host ? "venue-account-badge" : "venue-account-badge venue-account-badge--floating";
    badge.href = "/account/";
    badge.setAttribute("aria-label", "Open account page");

    const avatar = document.createElement("span");
    avatar.className = "venue-account-badge__avatar";
    avatar.setAttribute("aria-hidden", "true");
    avatar.textContent = "A";

    const label = document.createElement("span");
    label.className = "venue-account-badge__label";
    label.textContent = "Account";

    badge.append(avatar, label);
    return { badge, avatar, label };
  }

  async function refreshLabel(label, avatar) {
    try {
      const response = await fetch("/api/session/me", { credentials: "include" });
      const payload = await response.json().catch(() => null);
      if (!response.ok || !payload?.signed_in) return;
      const name = displayName(payload.data || {});
      label.textContent = name;
      avatar.textContent = initials(name);
    } catch (error) {
      console.warn("account badge lookup failed", error);
    }
  }

  function mount() {
    if (document.getElementById(BADGE_ID) || document.getElementById("account-label")) return;
    const header = document.querySelector("header.header") || document.querySelector("header") || null;
    if (header?.querySelector('a[href="/account/"]')) return;
    installStyle();

    const host = header?.querySelector(".launchbar, .toolbar, .header-right") || null;
    const { badge, avatar, label } = createBadge(host);

    if (host) {
      host.appendChild(badge);
    } else {
      document.body.appendChild(badge);
    }

    refreshLabel(label, avatar);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", mount, { once: true });
  } else {
    mount();
  }
})();

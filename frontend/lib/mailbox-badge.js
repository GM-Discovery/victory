// Kernel 95 Pass 6: shared "do I have unread Mailbox messages" check and
// pip-toggling helper, used by three separate surfaces (the campus Info
// Booth, the Trailers venue map pin, and the account-menu dropdown inside
// Catharsis/First Theater). Deliberately reads GET /api/messages -- the
// same endpoint the Mailbox page itself uses, with the same `read` field
// per message -- rather than inventing a second source of unread truth.
(function () {
  const CACHE_TTL_MS = 20000;
  let cached = null; // { at, hasUnread }

  function installStyle() {
    if (document.getElementById("mailbox-badge-style")) return;
    const style = document.createElement("style");
    style.id = "mailbox-badge-style";
    style.textContent = `
      [data-mailbox-pip-target] {
        position: relative;
      }

      [data-mailbox-pip-target].has-unread-pip::after {
        content: "";
        position: absolute;
        top: -2px;
        right: -2px;
        width: 9px;
        height: 9px;
        border-radius: 50%;
        background: #ff5a3c;
        border: 1.5px solid rgba(8, 8, 8, 0.85);
        pointer-events: none;
      }
    `;
    document.head.appendChild(style);
  }

  async function hasUnreadMessages({ force = false } = {}) {
    if (!force && cached && Date.now() - cached.at < CACHE_TTL_MS) {
      return cached.hasUnread;
    }
    try {
      const res = await fetch("/api/messages", { credentials: "include", cache: "no-store" });
      if (!res.ok) {
        cached = { at: Date.now(), hasUnread: false };
        return false;
      }
      const payload = await res.json().catch(() => null);
      const messages = payload?.data?.messages;
      const hasUnread = Array.isArray(messages) && messages.some((m) => !m.read);
      cached = { at: Date.now(), hasUnread };
      return hasUnread;
    } catch (error) {
      return cached ? cached.hasUnread : false;
    }
  }

  // Finds every element in `root` carrying data-mailbox-pip-target and
  // toggles the pip dot on it based on current unread state. Safe to call
  // repeatedly (e.g. on page load and again when a menu containing one of
  // these targets opens) -- the TTL cache above keeps repeat calls cheap.
  async function refreshPips(root = document, { force = false } = {}) {
    const targets = root.querySelectorAll("[data-mailbox-pip-target]");
    if (targets.length === 0) return;
    installStyle();
    const hasUnread = await hasUnreadMessages({ force });
    targets.forEach((el) => {
      el.classList.toggle("has-unread-pip", hasUnread);
    });
  }

  // Reported: the pip kept showing unread even after reading everything
  // in the Mailbox. The backend's own is_read tracking (messages.go's
  // getMessage) was already correct -- this was the page, not the data.
  // Pressing Back after reading mail restores the previous page from the
  // browser's bfcache rather than re-running its script from scratch, so
  // whatever pip state was on screen on the way out (unread) just sits
  // there -- nothing ever calls refreshPips again to notice it's stale.
  // `pageshow` with `event.persisted` is the standard signal for exactly
  // this restoration; forcing past the TTL cache here matters since a
  // bfcache restore can happen well within its 20s window.
  window.addEventListener("pageshow", (event) => {
    if (event.persisted) refreshPips(document, { force: true });
  });

  window.VictoryMailboxBadge = { hasUnreadMessages, refreshPips };
})();

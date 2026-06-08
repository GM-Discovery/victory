(function () {
  const selector = "#discord-mic-status[data-venue-slug]";

  function escapeText(value) {
    return String(value || "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }

  function renderStatus(el, status) {
    if (!el) return;
    const mic = status?.mic || null;
    const canControl = Boolean(status?.can_control);
    const active = Boolean(mic?.active);

    if (!active && !canControl) {
      el.hidden = true;
      return;
    }

    el.hidden = false;
    const parts = [
      active ? "House Mic: Hot" : "House Mic: Off",
      status?.linked ? "Discord Bridge: On" : "Discord Bridge: Off",
      status?.command_registered ? "Discord Intake: Ready" : "Discord Intake: Check",
    ];
    const threadName = String(mic?.thread_name || "").trim();
    const threadURL = String(mic?.thread_url || "").trim();
    const label = parts.join(" · ");

    if (active && threadName && threadURL) {
      el.innerHTML = `${escapeText(label)} <a href="${escapeText(threadURL)}" target="_blank" rel="noreferrer">${escapeText(threadName)}</a>`;
      return;
    }

    if (active && threadName) {
      el.textContent = `${label} ${threadName}`;
      return;
    }

    el.textContent = label;
  }

  async function loadStatus(el) {
    const venueSlug = String(el.dataset.venueSlug || "").trim();
    if (!venueSlug) return;

    try {
      const response = await fetch(`/api/discord/mic/status?venue_slug=${encodeURIComponent(venueSlug)}`, {
        credentials: "include",
        cache: "no-store",
      });
      const payload = await response.json().catch(() => null);
      if (!response.ok || !payload?.ok) {
        throw new Error(payload?.error || `Failed to load mic status (HTTP ${response.status})`);
      }
      renderStatus(el, payload.data || {});
    } catch (error) {
      if (el.dataset.venueSlug) {
        el.hidden = false;
        el.textContent = "House Mic unavailable";
      }
      console.warn("discord mic status failed", error);
    }
  }

  async function refreshStatus(venueSlug) {
    const slug = String(venueSlug || "").trim();
    if (!slug) return;
    const el = Array.from(document.querySelectorAll(selector)).find((node) => {
      return String(node.dataset?.venueSlug || "").trim() === slug;
    });
    if (!el) return;
    await loadStatus(el);
  }

  function bootstrap() {
    const el = document.querySelector(selector);
    if (!el) return;
    loadStatus(el);
    window.setInterval(() => loadStatus(el), 30000);
  }

  window.addEventListener("victory:discord-mic-refresh", (event) => {
    const venueSlug = event?.detail?.venueSlug;
    if (venueSlug) {
      refreshStatus(venueSlug);
    }
  });

  window.VictoryDiscordMic = {
    refreshStatus,
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", bootstrap, { once: true });
  } else {
    bootstrap();
  }
})();

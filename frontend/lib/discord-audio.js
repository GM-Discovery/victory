(function () {
  const STYLE_ID = "victory-discord-presence-style";
  const instances = new Map();
  const REFRESH_EVENT = "victory:discord-presence-refresh";

  function ensureStyles() {
    if (document.getElementById(STYLE_ID)) return;
    const style = document.createElement("style");
    style.id = STYLE_ID;
    style.textContent = `
      .victory-discord-presence {
        display: grid;
        gap: 10px;
        padding: 14px 15px 15px;
        border-radius: 16px;
        border: 1px solid rgba(239, 219, 176, 0.14);
        background: linear-gradient(180deg, rgba(18, 24, 31, 0.9), rgba(10, 14, 20, 0.86));
        box-shadow: 0 16px 34px rgba(0, 0, 0, 0.24);
        color: #f5ead7;
        min-width: 0;
      }

      .victory-discord-presence__eyebrow {
        text-transform: uppercase;
        letter-spacing: 0.16em;
        font-size: 0.72rem;
        color: #d9b873;
      }

      .victory-discord-presence__title {
        font-size: 1rem;
        font-weight: 800;
        color: #fff1ce;
      }

      .victory-discord-presence__line,
      .victory-discord-presence__hint,
      .victory-discord-presence__meta {
        font-size: 0.93rem;
        line-height: 1.45;
        color: #d3c4af;
      }

      .victory-discord-presence__line strong {
        color: #f5e3bf;
      }

      .victory-discord-presence__status {
        font-size: 0.84rem;
        text-transform: uppercase;
        letter-spacing: 0.1em;
        color: #9fc4de;
      }

      .victory-discord-presence__status[data-state="ready"] {
        color: #97d39b;
      }

      .victory-discord-presence__status[data-state="needs_repair"] {
        color: #e0b26b;
      }

      .victory-discord-presence__status[data-state="server_not_linked"],
      .victory-discord-presence__status[data-state="not_enabled"] {
        color: #c9a5a5;
      }

      .victory-discord-presence__actions {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
        align-items: center;
      }

      .victory-discord-presence__button {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        min-height: 38px;
        padding: 0 14px;
        border-radius: 999px;
        border: 1px solid rgba(169, 194, 223, 0.24);
        background: linear-gradient(180deg, rgba(72, 96, 123, 0.96), rgba(38, 54, 73, 0.96));
        color: #f8f0e2;
        text-decoration: none;
        font-weight: 700;
        box-shadow: 0 10px 24px rgba(0, 0, 0, 0.18);
      }

      .victory-discord-presence__button[aria-disabled="true"] {
        opacity: 0.48;
        pointer-events: none;
      }

      .victory-discord-presence__meta-row {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
        color: #9aa9b8;
        font-size: 0.8rem;
      }

      .victory-discord-presence__meta-chip {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 5px 10px;
        border-radius: 999px;
        border: 1px solid rgba(255, 255, 255, 0.08);
        background: rgba(255, 255, 255, 0.03);
      }

      .victory-discord-presence__participants {
        display: grid;
        gap: 8px;
      }

      .victory-discord-presence__participant {
        display: grid;
        grid-template-columns: 36px minmax(0, 1fr);
        gap: 10px;
        align-items: center;
        padding: 10px 11px;
        border-radius: 14px;
        border: 1px solid rgba(255, 255, 255, 0.08);
        background: rgba(255, 255, 255, 0.03);
      }

      .victory-discord-presence__participant[data-speaking="true"] {
        box-shadow: 0 0 0 1px rgba(151, 211, 155, 0.32), 0 0 0 4px rgba(151, 211, 155, 0.08);
      }

      .victory-discord-presence__avatar {
        width: 36px;
        height: 36px;
        border-radius: 999px;
        overflow: hidden;
        background: radial-gradient(circle at 30% 20%, rgba(255, 243, 218, 0.28), rgba(255, 255, 255, 0.06));
        border: 1px solid rgba(255, 255, 255, 0.12);
        display: grid;
        place-items: center;
        color: #f6eddc;
        font-size: 0.82rem;
        font-weight: 800;
      }

      .victory-discord-presence__avatar img {
        width: 100%;
        height: 100%;
        object-fit: cover;
        display: block;
      }

      .victory-discord-presence__participant-main {
        display: grid;
        gap: 3px;
        min-width: 0;
      }

      .victory-discord-presence__participant-name {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        align-items: center;
        min-width: 0;
      }

      .victory-discord-presence__participant-label {
        color: #f6e6c4;
        font-weight: 800;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .victory-discord-presence__participant-badge {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 3px 8px;
        border-radius: 999px;
        background: rgba(151, 211, 155, 0.12);
        border: 1px solid rgba(151, 211, 155, 0.18);
        color: #a9dfb0;
        font-size: 0.74rem;
        letter-spacing: 0.04em;
        text-transform: uppercase;
      }

      .victory-discord-presence__participant-subline {
        color: #b7c1cc;
        font-size: 0.83rem;
        line-height: 1.35;
      }

      .victory-discord-presence__empty {
        padding: 10px 11px;
        border-radius: 14px;
        border: 1px dashed rgba(255, 255, 255, 0.11);
        background: rgba(255, 255, 255, 0.02);
        color: #9aa9b8;
        font-size: 0.9rem;
      }
    `;
    document.head.appendChild(style);
  }

  function normalizeVenueSlug(value) {
    return String(value || "").trim().toLowerCase();
  }

  function escapeHtml(value) {
    return String(value || "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll("\"", "&quot;")
      .replaceAll("'", "&#39;");
  }

  function escapeAttribute(value) {
    return escapeHtml(value);
  }

  function initialsForName(value) {
    const parts = String(value || "")
      .trim()
      .split(/\s+/)
      .filter(Boolean);
    if (parts.length === 0) return "?";
    if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
    return (parts[0][0] + parts[1][0]).toUpperCase();
  }

  function prettyState(value) {
    return String(value || "")
      .split("_")
      .filter(Boolean)
      .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
      .join(" ");
  }

  function mountRoot(host, venueSlug) {
    host.innerHTML = `
      <section class="victory-discord-presence" data-venue-slug="${venueSlug}">
        <div class="victory-discord-presence__eyebrow">Presence</div>
        <div class="victory-discord-presence__title">Discord Presence</div>
        <div class="victory-discord-presence__line" data-audio-channel>Loading presence status...</div>
        <div class="victory-discord-presence__status" data-audio-status data-state="loading">Loading</div>
        <div class="victory-discord-presence__actions">
          <a class="victory-discord-presence__button" data-audio-open href="#" target="_blank" rel="noopener noreferrer" aria-disabled="true">Open Discord Presence</a>
        </div>
        <div class="victory-discord-presence__hint" data-audio-hint></div>
        <div class="victory-discord-presence__meta-row" data-audio-meta hidden>
          <span class="victory-discord-presence__meta-chip" data-audio-meta-channel></span>
          <span class="victory-discord-presence__meta-chip" data-audio-meta-state></span>
        </div>
        <div class="victory-discord-presence__participants" data-audio-participants></div>
      </section>
    `;
    return host.firstElementChild;
  }

  function renderStatus(root, payload) {
    const line = root.querySelector("[data-audio-channel]");
    const status = root.querySelector("[data-audio-status]");
    const open = root.querySelector("[data-audio-open]");
    const hint = root.querySelector("[data-audio-hint]");
    const meta = root.querySelector("[data-audio-meta]");
    const metaChannel = root.querySelector("[data-audio-meta-channel]");
    const metaState = root.querySelector("[data-audio-meta-state]");
    const participants = root.querySelector("[data-audio-participants]");

    const data = payload && typeof payload === "object" ? payload : {};
    const audioChannel = data.audio_channel && typeof data.audio_channel === "object" ? data.audio_channel : null;
    const venueName = String(data.venue_name || "").trim();
    const channelName = String(audioChannel?.name || "").trim();
    const channelType = String(audioChannel?.type || "voice").trim();
    const statusValue = String(data.status || "unknown").trim();
    const message = String(data.message || "").trim();
    const repairHint = String(data.repair_hint || "").trim();
    const canOpen = Boolean(data.can_open && String(audioChannel?.open_url || "").trim());
    const participantsList = Array.isArray(data.participants) ? data.participants : [];
    const voiceTrackingReason = String(data.voice_state_tracking?.reason || "").trim();
    const speakerReason = String(data.speaker_indicator?.reason || "").trim();
    const volumeReason = String(data.volume_controls?.reason || "").trim();

    let channelLabel = "Presence: Not configured";
    if (statusValue === "not_enabled") {
      channelLabel = "Presence: Not enabled for this venue";
    } else if (statusValue === "server_not_linked") {
      channelLabel = "Presence: Server not linked";
    } else if (channelName) {
      channelLabel = `Presence: ${channelName}`;
    }
    if (venueName && statusValue === "ready" && !channelName) {
      channelLabel = `Presence: ${venueName}`;
    }

    line.textContent = channelLabel;
    status.textContent = `Status: ${prettyState(statusValue)}`;
    status.dataset.state = statusValue;
    open.href = canOpen ? String(audioChannel.open_url).trim() : "#";
    open.setAttribute("aria-disabled", canOpen ? "false" : "true");
    open.hidden = false;
    hint.textContent = message || repairHint || (audioChannel?.status === "missing" ? "Needs repair in Producer's Office." : "Presence is backed by the mapped Discord voice channel.");
    if (!hint.textContent) {
      hint.hidden = true;
    } else {
      hint.hidden = false;
    }
    if (meta) {
      const chips = [];
      if (channelName) {
        chips.push(`Channel: ${channelName}`);
      }
      if (audioChannel?.type) {
        chips.push(`Type: ${String(audioChannel.type).trim()}`);
      }
      if (audioChannel?.expected_name) {
        chips.push(`Expected: ${String(audioChannel.expected_name).trim()}`);
      }
      meta.hidden = chips.length === 0;
      metaChannel.textContent = chips[0] || "";
      metaState.textContent = chips[1] || `Mode: ${prettyState(statusValue)}`;
      if (voiceTrackingReason || speakerReason || volumeReason) {
        const extra = [];
        if (voiceTrackingReason) extra.push(voiceTrackingReason);
        if (speakerReason) extra.push(speakerReason);
        if (volumeReason) extra.push(volumeReason);
        hint.textContent = [hint.textContent, ...extra].filter(Boolean).join(" ");
        hint.hidden = false;
      }
    }

    if (participants) {
      if (participantsList.length === 0) {
        participants.innerHTML = `<div class="victory-discord-presence__empty">No Discord voice participants are currently visible.</div>`;
      } else {
        participants.innerHTML = participantsList.map((participant) => {
          const label = escapeHtml(String(participant.display_name || participant.discord_display_name || participant.discord_user_id || "Unknown"));
          const sourceLabel = participant.source_label ? `<span class="victory-discord-presence__participant-badge">${escapeHtml(String(participant.source_label))}</span>` : "";
          const statusLabel = escapeHtml(String(participant.status || "listening"));
          const linkedLabel = participant.victory_display_name ? escapeHtml(String(participant.victory_display_name)) : "";
          const discordLabel = participant.discord_display_name ? escapeHtml(String(participant.discord_display_name)) : "";
          const avatar = String(participant.avatar_url || "").trim()
            ? `<img src="${escapeAttribute(String(participant.avatar_url))}" alt="">`
            : `<span>${escapeHtml(initialsForName(label))}</span>`;
          const sublineParts = [];
          if (linkedLabel && linkedLabel !== label) {
            sublineParts.push(`Victory: ${linkedLabel}`);
          }
          if (discordLabel && discordLabel !== label) {
            sublineParts.push(`Discord: ${discordLabel}`);
          }
          if (participant.source_label) {
            sublineParts.push(String(participant.source_label));
          }
          const subline = sublineParts.length ? sublineParts.join(" · ") : (participant.linked_user_id ? "Linked Victory user" : "Discord-only participant");
          return `
            <div class="victory-discord-presence__participant" data-speaking="${participant.speaking ? "true" : "false"}">
              <div class="victory-discord-presence__avatar" aria-hidden="true">${avatar}</div>
              <div class="victory-discord-presence__participant-main">
                <div class="victory-discord-presence__participant-name">
                  <span class="victory-discord-presence__participant-label">${label}</span>
                  <span class="victory-discord-presence__participant-badge">${escapeHtml(statusLabel)}</span>
                  ${sourceLabel}
                </div>
                <div class="victory-discord-presence__participant-subline">${escapeHtml(subline)}</div>
              </div>
            </div>
          `;
        }).join("");
      }
    }

    if (channelType && channelName && statusValue === "ready") {
      line.dataset.channelType = channelType;
    }
  }

  async function refreshInstance(instance) {
    if (!instance || instance.destroyed || instance.pending) return;
    const { venueSlug, root } = instance;
    if (!root) return;

    instance.pending = true;
    try {
      const response = await fetch(`/api/discord/audio/status?venue_slug=${encodeURIComponent(venueSlug)}`, {
        credentials: "include",
        cache: "no-store",
      });
      const payload = await response.json().catch(() => null);
      if (!response.ok || !payload?.ok) {
        renderStatus(root, {
          status: "unavailable",
          message: payload?.data?.message || payload?.data?.detail || payload?.error || `HTTP ${response.status}`,
          audio_channel: null,
          can_open: false,
        });
        return;
      }
      renderStatus(root, payload.data || {});
    } catch (error) {
      renderStatus(root, {
        status: "unavailable",
        message: error instanceof Error ? error.message : "Failed to load audio status.",
        audio_channel: null,
        can_open: false,
      });
    } finally {
      instance.pending = false;
    }
  }

  function mount(options) {
    ensureStyles();
    const venueSlug = normalizeVenueSlug(options?.venueSlug);
    if (!venueSlug) return null;

    const host = typeof options?.mount === "string" ? document.querySelector(options.mount) : options?.mount;
    if (!host) return null;

    if (instances.has(venueSlug)) {
      instances.get(venueSlug).destroy();
    }

    const root = mountRoot(host, venueSlug);
    const openLink = root.querySelector("[data-audio-open]");
    openLink?.addEventListener("click", (event) => {
      if (openLink.getAttribute("aria-disabled") === "true") {
        event.preventDefault();
      }
    });
    const instance = {
      venueSlug,
      host,
      root,
      pending: false,
      destroyed: false,
      timer: window.setInterval(() => refreshInstance(instance), 5000),
      destroy() {
        if (this.destroyed) return;
        this.destroyed = true;
        if (this.timer) {
          window.clearInterval(this.timer);
          this.timer = null;
        }
        instances.delete(this.venueSlug);
      },
      refresh() {
        return refreshInstance(this);
      },
    };

    instances.set(venueSlug, instance);
    refreshInstance(instance);
    return instance;
  }

  window.addEventListener("visibilitychange", () => {
    if (document.hidden) return;
    for (const instance of instances.values()) {
      refreshInstance(instance);
    }
  });

  window.addEventListener(REFRESH_EVENT, (event) => {
    const slug = normalizeVenueSlug(event?.detail?.venueSlug);
    if (!slug) return;
    const instance = instances.get(slug);
    if (instance) {
      refreshInstance(instance);
    }
  });

  window.VictoryDiscordAudio = {
    mount,
    refresh(venueSlug) {
      const instance = instances.get(normalizeVenueSlug(venueSlug));
      if (instance) return refreshInstance(instance);
      return null;
    },
    destroy(venueSlug) {
      const instance = instances.get(normalizeVenueSlug(venueSlug));
      if (instance) instance.destroy();
    },
  };
  window.VictoryDiscordPresence = window.VictoryDiscordAudio;
})();

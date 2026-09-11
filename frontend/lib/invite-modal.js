// Kernel 100: one shared "Invite People" modal, reachable from the
// account menu on every venue page, replacing the bare-token version
// that used to be duplicated separately in Producer's Office and
// Director's Chair. Deliberately not office-specific -- it asks the
// backend what roles *this* logged-in user is actually allowed to
// invite (GET /api/invites/authority) and only offers those, so a
// Director never sees "Director" as an option and a Producer sees
// everything, regardless of which page they opened this from.
(function () {
  let modalEl = null;
  let statusEl = null;
  let formEl = null;
  let roleSelect = null;
  let emailInput = null;
  let venueSlugField = null;
  let venueSlugInput = null;
  let linkResultEl = null;
  let linkInput = null;

  // Confirmed on real hardware as a real bug: window.location.origin is
  // always "localhost" for an Operator browsing their own machine, which
  // is useless to anyone else the link gets sent to -- localhost on the
  // recipient's machine means their machine, not the Operator's. The
  // Windows launcher (when remote access is on) keeps the backend told
  // about its actual reachable address; this asks for that first and only
  // falls back to the browser's own origin when nothing better is known
  // (no remote access configured, or any non-Windows deployment where
  // this endpoint always reports null).
  async function resolvePublicOrigin() {
    try {
      const res = await fetch("/api/system/public-url", { credentials: "include" });
      const payload = await res.json().catch(() => null);
      if (payload?.ok && payload.public_url) return payload.public_url;
    } catch {
      // fall through to the origin fallback below
    }
    return window.location.origin;
  }

  function installStyle() {
    if (document.getElementById("invite-modal-style")) return;
    const style = document.createElement("style");
    style.id = "invite-modal-style";
    style.textContent = `
      .invite-modal-backdrop {
        position: fixed; inset: 0; z-index: 9000;
        background: rgba(0,0,0,0.6);
        display: flex; align-items: center; justify-content: center;
      }
      .invite-modal {
        width: min(420px, calc(100vw - 32px));
        background: #0d0b0c;
        border: 1px solid rgba(211,25,67,0.28);
        border-top: 3px solid #a30d2d;
        border-radius: 6px;
        padding: 24px;
        color: #f6edf0;
        font-family: Inter, ui-sans-serif, system-ui, sans-serif;
        box-shadow: 0 26px 80px rgba(0,0,0,0.72);
      }
      .invite-modal h2 { margin: 0 0 12px; font-size: 1.3rem; }
      .invite-modal label {
        display: block; margin: 12px 0 4px;
        font-size: 0.78rem; font-weight: 700; letter-spacing: 0.08em;
        text-transform: uppercase; color: #e7d9dd;
      }
      .invite-modal select, .invite-modal input {
        width: 100%; padding: 10px 11px; border-radius: 4px;
        border: 1px solid rgba(255,255,255,0.13);
        background: #151113; color: #f6edf0; font-size: 0.95rem;
        box-sizing: border-box;
      }
      .invite-modal button:not(.invite-modal-close) {
        margin-top: 16px; width: 100%; padding: 11px 14px;
        border: 1px solid #d92b50; border-radius: 4px;
        background: linear-gradient(180deg, #d31943, #8b0926);
        color: #fff7f9; font-weight: 700; cursor: pointer;
      }
      .invite-modal .invite-modal-secondary {
        background: transparent; border-color: rgba(255,255,255,0.18); margin-top: 8px;
      }
      .invite-modal-status { margin-top: 10px; min-height: 1.2em; color: #ff6d8e; font-size: 0.9rem; }
      .invite-modal-email-hint { margin: 6px 0 0; color: #bda9ae; font-size: 0.78rem; line-height: 1.4; }
      .invite-modal-header {
        display: flex; align-items: center; justify-content: space-between;
        margin: 0 0 12px;
      }
      .invite-modal-header h2 { margin: 0; }
      .invite-modal-close {
        flex: none; background: transparent; border: none; color: #bda9ae;
        font-size: 1.4rem; line-height: 1; cursor: pointer;
        width: 28px; height: 28px; margin: 0; padding: 0;
        border-radius: 50%; display: flex; align-items: center; justify-content: center;
        transition: background 120ms ease, color 120ms ease;
      }
      .invite-modal-close:hover { background: rgba(255,255,255,0.08); color: #f6edf0; }
      .invite-modal [hidden] { display: none !important; }
    `;
    document.head.appendChild(style);
  }

  function build() {
    if (modalEl) return;
    installStyle();

    const backdrop = document.createElement("div");
    backdrop.className = "invite-modal-backdrop";
    backdrop.hidden = true;
    backdrop.addEventListener("click", (event) => {
      if (event.target === backdrop) close();
    });

    backdrop.innerHTML = `
      <div class="invite-modal" role="dialog" aria-modal="true">
        <div class="invite-modal-header">
          <h2>Invite People</h2>
          <button type="button" class="invite-modal-close" aria-label="Close">×</button>
        </div>
        <form id="invite-modal-form">
          <label for="invite-modal-role">Role</label>
          <select id="invite-modal-role" required></select>

          <div id="invite-modal-venue-field" hidden>
            <label for="invite-modal-venue">Venue slug</label>
            <input id="invite-modal-venue" type="text" placeholder="e.g. first-theater" />
          </div>

          <label for="invite-modal-email">Email (optional)</label>
          <input id="invite-modal-email" type="email" placeholder="Locks the invite to this address" />

          <button type="submit">Create Invite</button>
        </form>
        <div id="invite-modal-status" class="invite-modal-status"></div>
        <div id="invite-modal-link-result" hidden>
          <label for="invite-modal-link">Share this link</label>
          <input id="invite-modal-link" type="text" readonly />
          <button type="button" id="invite-modal-copy" class="invite-modal-secondary">Copy Link</button>
          <button type="button" id="invite-modal-email-share" class="invite-modal-secondary">Open in Mail App</button>
          <p class="invite-modal-email-hint">
            Uses whatever email app this computer has set as default -- often not
            configured, or set to a work account. If nothing opens, Copy Link works
            with any email, text, or chat app instead.
          </p>
        </div>
      </div>
    `;

    document.body.appendChild(backdrop);
    modalEl = backdrop;
    formEl = document.getElementById("invite-modal-form");
    statusEl = document.getElementById("invite-modal-status");
    roleSelect = document.getElementById("invite-modal-role");
    emailInput = document.getElementById("invite-modal-email");
    venueSlugField = document.getElementById("invite-modal-venue-field");
    venueSlugInput = document.getElementById("invite-modal-venue");
    linkResultEl = document.getElementById("invite-modal-link-result");
    linkInput = document.getElementById("invite-modal-link");

    modalEl.querySelector(".invite-modal-close").addEventListener("click", close);
    roleSelect.addEventListener("change", () => {
      venueSlugField.hidden = roleSelect.value !== "audience";
    });
    formEl.addEventListener("submit", onSubmit);
    document.getElementById("invite-modal-copy").addEventListener("click", () => {
      linkInput.select();
      navigator.clipboard?.writeText(linkInput.value).catch(() => {
        document.execCommand("copy");
      });
    });
    document.getElementById("invite-modal-email-share").addEventListener("click", () => {
      const subject = encodeURIComponent("You're invited to Victory");
      const body = encodeURIComponent(`Come join us on Victory:\n\n${linkInput.value}`);
      window.location.href = `mailto:?subject=${subject}&body=${body}`;
    });
  }

  async function onSubmit(event) {
    event.preventDefault();
    statusEl.textContent = "Creating invite...";
    linkResultEl.hidden = true;

    const role = roleSelect.value;
    if (role === "audience" && !venueSlugInput.value.trim()) {
      statusEl.textContent = "A venue slug is required for Audience invites.";
      return;
    }

    try {
      const response = await fetch("/api/invites", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          target_role: role,
          target_email: emailInput.value.trim(),
          venue_slug: role === "audience" ? venueSlugInput.value.trim() : "",
          expires_in_hours: 72,
          max_uses: 1,
        }),
      });
      const payload = await response.json().catch(() => null);
      if (!response.ok || !payload?.ok) {
        throw new Error(payload?.error || `Invite failed (HTTP ${response.status})`);
      }

      const link = `${await resolvePublicOrigin()}/invite/?token=${encodeURIComponent(payload.data.token)}`;
      linkInput.value = link;
      linkResultEl.hidden = false;
      statusEl.textContent = "Invite created -- expires in 72 hours, single use.";
    } catch (error) {
      statusEl.textContent = error.message || String(error);
    }
  }

  async function open() {
    build();
    modalEl.hidden = false;
    statusEl.textContent = "";
    linkResultEl.hidden = true;
    formEl.hidden = false;
    roleSelect.innerHTML = "<option>Loading...</option>";

    try {
      const response = await fetch("/api/invites/authority", { credentials: "include" });
      const payload = await response.json().catch(() => null);
      if (!response.ok || !payload?.ok) {
        formEl.hidden = true;
        statusEl.textContent = "You don't have permission to invite people at this location.";
        return;
      }
      const roleLabels = { director: "Director", cast: "Cast", crew: "Crew", audience: "Audience" };
      roleSelect.innerHTML = payload.data.available_roles
        .map((role) => `<option value="${role}">${roleLabels[role] || role}</option>`)
        .join("");
      venueSlugField.hidden = roleSelect.value !== "audience";
    } catch {
      formEl.hidden = true;
      statusEl.textContent = "Could not check invite permissions -- try again.";
    }
  }

  function close() {
    if (modalEl) modalEl.hidden = true;
  }

  window.VictoryInviteModal = { open, close };
})();

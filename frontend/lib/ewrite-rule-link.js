// Reusable "Link to rule" control (Kernel 79 Goal C) for any object type
// wired through /api/ewrite/object-links: cue, index_card, scene_element,
// dialogue_topic, equipment_item. Attaches to a container element; shows
// the current link (with a Remove button) or a paste-ID form to create one
// -- the same paste-an-ID convention already used elsewhere in this
// console (e.g. the Cue editor's "Target Show Scene Placement ID" field),
// not a new UX pattern. Deliberately unstyled beyond inline layout rules
// so it drops cleanly into any page's existing design system rather than
// assuming a class (like "chip small") that may not exist there.
//
// Usage: attachEwriteRuleLink(containerEl, { objectType: "cue", objectId: cue.id })

(function () {
  function idFieldFor(objectType) {
    switch (objectType) {
      case "cue": return "cue_id";
      case "index_card": return "index_card_element_id";
      case "scene_element": return "scene_element_id";
      case "dialogue_topic": return "dialogue_topic_id";
      case "equipment_item": return "equipment_item_id";
      default: return "object_id";
    }
  }

  function readerHref(pubId, anchor) {
    return "/venues/library/read.html?pub=" + encodeURIComponent(pubId) + (anchor ? "#" + encodeURIComponent(anchor) : "");
  }

  async function fetchLink(objectType, objectId) {
    const res = await fetch(
      "/api/ewrite/object-links?object_type=" + encodeURIComponent(objectType) + "&object_id=" + encodeURIComponent(objectId),
      { credentials: "include" }
    );
    const payload = await res.json().catch(() => null);
    if (!res.ok || !payload?.ok) return null;
    return (payload.data.links || {})[objectId] || null;
  }

  window.attachEwriteRuleLink = function attachEwriteRuleLink(container, opts) {
    const objectType = opts.objectType;
    const objectId = opts.objectId;

    async function render() {
      container.innerHTML = "";
      const status = document.createElement("span");
      status.textContent = "Loading rule link...";
      status.style.cssText = "font-size:12px;opacity:0.7;";
      container.appendChild(status);
      const link = await fetchLink(objectType, objectId);
      container.innerHTML = "";
      if (link) {
        renderLinked(link);
      } else {
        renderUnlinked();
      }
    }

    function renderLinked(link) {
      const a = document.createElement("a");
      a.href = readerHref(link.publication_id, link.section_anchor);
      a.target = "_blank";
      a.rel = "noopener";
      a.style.cssText = "font-size:12px;";
      a.textContent = "Rule: " + (link.section_title || link.publication_title) + " →";
      container.appendChild(a);

      const removeBtn = document.createElement("button");
      removeBtn.type = "button";
      removeBtn.textContent = "Remove";
      removeBtn.style.cssText = "margin-left:6px;font-size:12px;";
      removeBtn.addEventListener("click", async () => {
        if (!link.id) {
          alert("This link cannot be removed from here.");
          return;
        }
        removeBtn.disabled = true;
        try {
          const res = await fetch("/api/ewrite/object-links/" + encodeURIComponent(link.id), {
            method: "DELETE", credentials: "include",
          });
          const payload = await res.json().catch(() => null);
          if (!res.ok || !payload?.ok) {
            alert("Could not remove link: " + (payload?.data?.error || ("HTTP " + res.status)));
            return;
          }
          await render();
        } finally {
          removeBtn.disabled = false;
        }
      });
      container.appendChild(removeBtn);
    }

    function renderUnlinked() {
      const toggle = document.createElement("button");
      toggle.type = "button";
      toggle.textContent = "+ Link to rule";
      toggle.style.cssText = "font-size:12px;";
      toggle.addEventListener("click", () => {
        container.innerHTML = "";
        renderForm();
      });
      container.appendChild(toggle);
    }

    function renderForm() {
      const browseLink = document.createElement("a");
      browseLink.href = "/venues/library/index.html";
      browseLink.target = "_blank";
      browseLink.rel = "noopener";
      browseLink.textContent = "Browse Library ↗";
      browseLink.style.cssText = "display:block;font-size:12px;margin-bottom:4px;";
      container.appendChild(browseLink);

      const pubInput = document.createElement("input");
      pubInput.type = "text";
      pubInput.placeholder = "Publication ID";
      pubInput.style.cssText = "width:200px;margin-right:4px;font-size:12px;";
      container.appendChild(pubInput);

      const sectionInput = document.createElement("input");
      sectionInput.type = "text";
      sectionInput.placeholder = "Section ID (optional)";
      sectionInput.style.cssText = "width:200px;margin-right:4px;font-size:12px;";
      container.appendChild(sectionInput);

      const row = document.createElement("div");
      row.style.cssText = "margin-top:4px;";
      const saveBtn = document.createElement("button");
      saveBtn.type = "button";
      saveBtn.textContent = "Save";
      saveBtn.style.cssText = "font-size:12px;";
      saveBtn.addEventListener("click", async () => {
        const pubId = pubInput.value.trim();
        if (!pubId) {
          alert("Publication ID is required.");
          return;
        }
        saveBtn.disabled = true;
        try {
          const body = { object_type: objectType, publication_id: pubId, section_id: sectionInput.value.trim() };
          body[idFieldFor(objectType)] = objectId;
          const res = await fetch("/api/ewrite/object-links", {
            method: "POST", credentials: "include",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body),
          });
          const payload = await res.json().catch(() => null);
          if (!res.ok || !payload?.ok) {
            alert("Could not save link: " + (payload?.data?.error || ("HTTP " + res.status)));
            return;
          }
          await render();
        } finally {
          saveBtn.disabled = false;
        }
      });
      row.appendChild(saveBtn);

      const cancelBtn = document.createElement("button");
      cancelBtn.type = "button";
      cancelBtn.textContent = "Cancel";
      cancelBtn.style.cssText = "margin-left:4px;font-size:12px;";
      cancelBtn.addEventListener("click", render);
      row.appendChild(cancelBtn);
      container.appendChild(row);
    }

    render();
  };
})();

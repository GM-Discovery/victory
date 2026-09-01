// Kernel 94: central reactive store for the Vue Storyboards board. Ports
// board.html's (pre-K94) flat function set behind one object so components
// share state/actions instead of poking global DOM ids. The server remains
// authoritative -- every mutation here just calls the same REST endpoints
// the old board.html used and then reconciles from the response/reload,
// same as before (spec 18).
import { reactive, computed } from "/lib/vue.esm-browser.prod.js";

const G = window.VictoryStoryboardGrid;

export function createBoardStore(boardId) {
  const apiBase = "/api/storyboards/" + encodeURIComponent(boardId || "");
  const assetMetaCache = new Map(); // assetId -> payload.data | null (404)

  const state = reactive({
    boardId,
    snapshot: null,
    forbidden: false,
    loadError: "",
  });

  const affordances = computed(() =>
    G.roleAffordances(state.snapshot ? state.snapshot.viewer_tier : "")
  );

  async function apiFetch(url, opts) {
    const res = await fetch(url, Object.assign({ credentials: "include" }, opts || {}));
    const payload = await res.json().catch(() => null);
    return { res, payload };
  }
  async function apiPost(url, body, method) {
    return apiFetch(url, {
      method: method || "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body || {}),
    });
  }
  function errorMessage(payload, res) {
    return (payload && payload.data && payload.data.error) || (res && res.status) || "unknown error";
  }

  async function loadSnapshot() {
    const { res, payload } = await apiFetch(apiBase);
    if (isForbiddenAccessResponse(res, payload) || !res.ok || !payload || !payload.ok) {
      state.forbidden = true;
      window.showForbiddenScreen();
      return false;
    }
    state.snapshot = payload.data;
    return true;
  }

  function applySnapshot(data) {
    state.snapshot = data;
  }

  async function loadAssetMeta(assetId) {
    if (assetMetaCache.has(assetId)) return assetMetaCache.get(assetId);
    try {
      const { res, payload } = await apiFetch("/api/assets/" + encodeURIComponent(assetId));
      const meta = res.ok && payload && payload.ok ? payload.data : null;
      assetMetaCache.set(assetId, meta);
      return meta;
    } catch (err) {
      assetMetaCache.set(assetId, null);
      return null;
    }
  }
  function invalidateAssetMeta(assetId) {
    assetMetaCache.delete(assetId);
  }

  // ---------- Board-level ----------

  async function toggleArchive() {
    const archived = !state.snapshot.board.archived_at;
    const { res, payload } = await apiPost(apiBase + "/archive", { archived });
    if (!res.ok || !payload || !payload.ok) { alert("Failed: " + errorMessage(payload, res)); return; }
    await loadSnapshot();
  }

  // ---------- Columns ----------

  // Kernel 94 spec 14 fix: AddColumn always appends after every existing
  // column server-side (columns.go has no position param), so a Timeline
  // board's protected Ending column would end up displaced to the middle
  // whenever the generic "+ Column" control was used. insertColumnFlow's
  // append-then-reorder trick already avoided this for the column menu's
  // Insert left/right items; addColumn now always routes through the same
  // path when an Ending column exists, so there is no longer a plain
  // "append past the end" control left to misuse. ReorderColumns' own
  // boundary-protection check (columns.go) is the actual backstop if this
  // ever races a concurrent edit.
  async function addColumn(title) {
    const ending = (state.snapshot.columns || []).find((c) => c.column_role === "ending");
    if (ending) {
      return insertColumn(ending, "left", title);
    }
    const { res, payload } = await apiPost(apiBase + "/columns", { title });
    if (!res.ok || !payload || !payload.ok) { alert("Add column failed: " + errorMessage(payload, res)); return; }
    await loadSnapshot();
  }

  async function insertColumn(referenceCol, side, titleOverride) {
    const title = titleOverride !== undefined ? titleOverride : prompt("New column title", "New Column");
    if (title === null || title === undefined) return;
    const created = await apiPost(apiBase + "/columns", { title });
    if (!created.res.ok || !created.payload || !created.payload.ok) {
      alert("Could not create column: " + errorMessage(created.payload, created.res));
      return;
    }
    const newId = created.payload.data.column.id;
    const ids = state.snapshot.columns.map((c) => c.id);
    const refIndex = ids.indexOf(referenceCol.id);
    const insertAt = side === "left" ? refIndex : refIndex + 1;
    ids.splice(insertAt, 0, newId);
    const reordered = await apiPost(apiBase + "/columns/reorder", { ordered_column_ids: ids });
    if (!reordered.res.ok || !reordered.payload || !reordered.payload.ok) {
      alert("Column was created but could not be positioned (board changed concurrently) -- it was added at the end instead: " + errorMessage(reordered.payload, reordered.res));
    }
    await loadSnapshot();
  }

  async function renameColumn(col) {
    const title = prompt("Rename column", col.title);
    if (title === null) return;
    const { res, payload } = await apiPost(apiBase + "/columns/" + col.id, { title }, "PATCH");
    if (!res.ok || !payload || !payload.ok) { alert("Rename failed: " + errorMessage(payload, res)); return; }
    await loadSnapshot();
  }

  async function removeColumn(col) {
    let resp = await apiFetch(apiBase + "/columns/" + col.id, { method: "DELETE" });
    if (resp.res.ok && resp.payload && resp.payload.ok) { await loadSnapshot(); return; }
    if (!resp.payload || resp.payload.data.error !== "column_occupied") {
      alert("Remove failed: " + errorMessage(resp.payload, resp.res));
      return;
    }
    const choice = prompt(
      "This column has cards. Type 'delete' to remove them with the column, or the exact title of another column to move the cards there. Leave blank to cancel."
    );
    if (!choice) return;
    let url = apiBase + "/columns/" + col.id + "?resolution=";
    if (choice.trim().toLowerCase() === "delete") {
      url += "delete_cards";
    } else {
      const target = state.snapshot.columns.find((c) => c.title === choice.trim() && c.id !== col.id);
      if (!target) { alert("No other column with that exact title."); return; }
      url += "move_cards&target_column_id=" + encodeURIComponent(target.id);
    }
    const resp2 = await apiFetch(url, { method: "DELETE" });
    if (!resp2.res.ok || !resp2.payload || !resp2.payload.ok) { alert("Remove failed: " + errorMessage(resp2.payload, resp2.res)); return; }
    await loadSnapshot();
  }

  // ---------- Bands ----------

  async function addBand(label) {
    const value = label !== undefined ? label : prompt("New band label", "New Band");
    if (value === null || value === undefined) return;
    const { res, payload } = await apiPost(apiBase + "/bands", { label: value });
    if (!res.ok || !payload || !payload.ok) { alert("Add band failed: " + errorMessage(payload, res)); return; }
    await loadSnapshot();
  }

  async function editBand(band) {
    const label = prompt("Band label", band.label);
    if (label === null) return;
    const { res, payload } = await apiPost(apiBase + "/bands/" + band.id, { label, description: band.description || "" }, "PATCH");
    if (!res.ok || !payload || !payload.ok) { alert("Edit failed: " + errorMessage(payload, res)); return; }
    await loadSnapshot();
  }

  async function toggleBandCollapse(band) {
    const { res, payload } = await apiPost(apiBase + "/bands/" + band.id + "/collapse", { collapsed: !band.is_collapsed });
    if (!res.ok || !payload || !payload.ok) { alert("Collapse toggle failed: " + errorMessage(payload, res)); return; }
    await loadSnapshot();
  }

  async function toggleBandLock(band) {
    const { res, payload } = await apiPost(apiBase + "/bands/" + band.id + "/lock", { locked: !band.is_locked });
    if (!res.ok || !payload || !payload.ok) { alert("Lock toggle failed: " + errorMessage(payload, res)); return; }
    await loadSnapshot();
  }

  async function removeBand(band) {
    let resp = await apiFetch(apiBase + "/bands/" + band.id, { method: "DELETE" });
    if (resp.res.ok && resp.payload && resp.payload.ok) { await loadSnapshot(); return; }
    if (!resp.payload || resp.payload.data.error !== "band_occupied") {
      alert("Remove failed: " + errorMessage(resp.payload, resp.res));
      return;
    }
    const choice = prompt(
      "This band has rows. Type 'delete' to remove them (and their cards) with the band, or the exact label of another band to move the rows there. Leave blank to cancel."
    );
    if (!choice) return;
    let url = apiBase + "/bands/" + band.id + "?resolution=";
    if (choice.trim().toLowerCase() === "delete") {
      url += "delete_rows";
    } else {
      const target = state.snapshot.bands.find((b) => b.label === choice.trim() && b.id !== band.id);
      if (!target) { alert("No other band with that exact label."); return; }
      url += "move_rows&target_band_id=" + encodeURIComponent(target.id);
    }
    const resp2 = await apiFetch(url, { method: "DELETE" });
    if (!resp2.res.ok || !resp2.payload || !resp2.payload.ok) { alert("Remove failed: " + errorMessage(resp2.payload, resp2.res)); return; }
    await loadSnapshot();
  }

  // ---------- Rows ----------

  async function addRow(band, label) {
    const value = label !== undefined ? label : prompt("New row label in " + band.label, "New Row");
    if (value === null || value === undefined) return;
    const { res, payload } = await apiPost(apiBase + "/rows", { band_id: band.id, label: value });
    if (!res.ok || !payload || !payload.ok) { alert("Add row failed: " + errorMessage(payload, res)); return; }
    await loadSnapshot();
  }

  async function renameRow(row) {
    const label = prompt("Row label", row.label);
    if (label === null) return;
    const { res, payload } = await apiPost(apiBase + "/rows/" + row.id, { label, description: row.description || "" }, "PATCH");
    if (!res.ok || !payload || !payload.ok) { alert("Rename failed: " + errorMessage(payload, res)); return; }
    await loadSnapshot();
  }

  async function removeRow(row) {
    let resp = await apiFetch(apiBase + "/rows/" + row.id, { method: "DELETE" });
    if (resp.res.ok && resp.payload && resp.payload.ok) { await loadSnapshot(); return; }
    if (!resp.payload || resp.payload.data.error !== "row_occupied") {
      alert("Remove failed: " + errorMessage(resp.payload, resp.res));
      return;
    }
    const choice = prompt(
      "This row has cards. Type 'delete' to remove them with the row, or the exact label of another row to move the cards there. Leave blank to cancel."
    );
    if (!choice) return;
    let url = apiBase + "/rows/" + row.id + "?resolution=";
    if (choice.trim().toLowerCase() === "delete") {
      url += "delete_cards";
    } else {
      const target = state.snapshot.rows.find((r) => r.label === choice.trim() && r.id !== row.id);
      if (!target) { alert("No other row with that exact label."); return; }
      url += "move_cards&target_row_id=" + encodeURIComponent(target.id);
    }
    const resp2 = await apiFetch(url, { method: "DELETE" });
    if (!resp2.res.ok || !resp2.payload || !resp2.payload.ok) { alert("Remove failed: " + errorMessage(resp2.payload, resp2.res)); return; }
    await loadSnapshot();
  }

  // ---------- Cards ----------

  async function createCard(row, col, title) {
    const value = title !== undefined ? title : prompt("Card title?");
    if (value === null || value === undefined) return;
    const { res, payload } = await apiPost(apiBase + "/cards", { row_id: row.id, column_id: col.id, title: value });
    if (!res.ok || !payload || !payload.ok) { alert("Could not create card: " + errorMessage(payload, res)); return; }
    await loadSnapshot();
  }

  async function saveCard(card, fields) {
    const body = {
      base_version: card.version,
      title: fields.title,
      front_text: fields.front_text,
      back_text: fields.back_text,
      category: fields.category,
      color_token: fields.color_token,
    };
    if (affordances.value.canEditStructure) {
      body.hidden_from_audience = fields.hidden_from_audience;
    }
    const { res, payload } = await apiPost(apiBase + "/cards/" + card.id, body, "PATCH");
    if (!res.ok || !payload || !payload.ok) {
      return { ok: false, error: errorMessage(payload, res) };
    }
    if (affordances.value.canEditStructure && fields.is_locked !== !!card.is_locked) {
      await apiPost(apiBase + "/cards/" + card.id + "/lock", { locked: fields.is_locked });
    }
    await loadSnapshot();
    return { ok: true };
  }

  async function deleteCard(card) {
    const { res, payload } = await apiFetch(
      apiBase + "/cards/" + card.id + "?base_version=" + card.version,
      { method: "DELETE" }
    );
    if (!res.ok || !payload || !payload.ok) return { ok: false, error: errorMessage(payload, res) };
    await loadSnapshot();
    return { ok: true };
  }

  async function moveCard(card, targetRowId, targetColumnId) {
    const { res, payload } = await apiPost(apiBase + "/cards/" + card.id + "/move", {
      base_version: card.version, target_row_id: targetRowId, target_column_id: targetColumnId,
    });
    if (!res.ok || !payload || !payload.ok) return { ok: false, error: errorMessage(payload, res) };
    await loadSnapshot();
    return { ok: true };
  }

  async function swapCards(cardA, cardB) {
    const { res, payload } = await apiPost(apiBase + "/cards/swap", {
      card_a_id: cardA.id, base_version_a: cardA.version,
      card_b_id: cardB.id, base_version_b: cardB.version,
    });
    if (!res.ok || !payload || !payload.ok) { alert("Could not swap cards: " + errorMessage(payload, res)); }
    await loadSnapshot();
  }

  async function uploadCardImage(card, file) {
    const formData = new FormData();
    formData.append("file", file);
    const res = await fetch(apiBase + "/cards/" + card.id + "/image", {
      method: "POST", credentials: "include", body: formData,
    });
    const payload = await res.json().catch(() => null);
    if (!res.ok || !payload || !payload.ok) return { ok: false, error: errorMessage(payload, res) };
    invalidateAssetMeta(payload.data.card.image_asset_id);
    await loadSnapshot();
    return { ok: true, card: payload.data.card };
  }

  async function removeCardImage(card) {
    const { res, payload } = await apiFetch(apiBase + "/cards/" + card.id + "/image", { method: "DELETE" });
    if (!res.ok || !payload || !payload.ok) return { ok: false, error: errorMessage(payload, res) };
    await loadSnapshot();
    return { ok: true, card: payload.data.card };
  }

  // ---------- Sharing / grants ----------

  async function fetchGrants() {
    const { res, payload } = await apiFetch(apiBase + "/grants");
    if (!res.ok || !payload || !payload.ok) return [];
    return payload.data.grants || [];
  }

  async function removeGrant(grantId) {
    const { res, payload } = await apiFetch(apiBase + "/grants/" + grantId, { method: "DELETE" });
    if (!res.ok || !payload || !payload.ok) return { ok: false, error: errorMessage(payload, res) };
    return { ok: true };
  }

  async function addGrant(body) {
    const { res, payload } = await apiPost(apiBase + "/grants", body);
    if (!res.ok || !payload || !payload.ok) return { ok: false, error: errorMessage(payload, res) };
    return { ok: true };
  }

  // ---------- Coordination ----------

  async function postCoordinationAction(kind, targetUserId) {
    const path = apiBase + "/coordination/" + (kind === "leader" ? "group-leader" : "current-turn");
    const { res, payload } = await apiPost(path, { target_user_id: targetUserId });
    if (!res.ok || !payload || !payload.ok) {
      alert("Could not update: " + errorMessage(payload, res));
      return;
    }
    state.snapshot.coordination = payload.data.coordination;
  }

  // ---------- Reference Panel ----------

  async function saveReferenceFieldContent(field, text) {
    const { res, payload } = await apiPost(apiBase + "/reference-fields/" + field.id + "/content", { text });
    if (!res.ok || !payload || !payload.ok) return { ok: false, error: errorMessage(payload, res) };
    field.text_content = text;
    return { ok: true };
  }

  async function saveReferenceItemContent(field, item, content) {
    const { res, payload } = await apiFetch(apiBase + "/reference-fields/" + field.id + "/items/" + item.id, {
      method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ content }),
    });
    if (!res.ok || !payload || !payload.ok) return { ok: false, error: errorMessage(payload, res) };
    item.content = content;
    return { ok: true };
  }

  async function addReferenceItem(field, side, content) {
    const { res, payload } = await apiPost(apiBase + "/reference-fields/" + field.id + "/items", { side, content });
    if (!res.ok || !payload || !payload.ok) { alert("Could not add item: " + errorMessage(payload, res)); return; }
    await loadSnapshot();
  }

  async function moveReferenceItem(field, side, sortedItems, idx, delta) {
    const newOrder = sortedItems.map((it) => it.id);
    const targetIdx = idx + delta;
    const tmp = newOrder[idx];
    newOrder[idx] = newOrder[targetIdx];
    newOrder[targetIdx] = tmp;
    const { res, payload } = await apiPost(apiBase + "/reference-fields/" + field.id + "/items/reorder", { side, ordered_item_ids: newOrder });
    if (!res.ok || !payload || !payload.ok) { alert("Could not reorder: " + errorMessage(payload, res)); return; }
    await loadSnapshot();
  }

  async function removeReferenceItem(field, item) {
    if (!confirm("Remove this item?")) return;
    const { res, payload } = await apiFetch(apiBase + "/reference-fields/" + field.id + "/items/" + item.id, { method: "DELETE" });
    if (!res.ok || !payload || !payload.ok) { alert("Could not remove: " + errorMessage(payload, res)); return; }
    await loadSnapshot();
  }

  async function saveReferenceField(mode, target, body) {
    let res, payload;
    if (mode === "add") {
      ({ res, payload } = await apiPost(apiBase + "/reference-fields", body));
    } else {
      ({ res, payload } = await apiFetch(apiBase + "/reference-fields/" + target.id, {
        method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body),
      }));
    }
    if (!res.ok || !payload || !payload.ok) return { ok: false, error: errorMessage(payload, res) };
    await loadSnapshot();
    return { ok: true };
  }

  async function changeReferenceFieldType(field, newType) {
    const { res, payload } = await apiPost(apiBase + "/reference-fields/" + field.id + "/type", { field_type: newType });
    if (!res.ok || !payload || !payload.ok) { alert("Could not change type: " + errorMessage(payload, res)); return; }
    await loadSnapshot();
  }

  async function removeReferenceField(field) {
    if (!confirm('Remove field "' + field.label + '"?')) return;
    let resp = await apiFetch(apiBase + "/reference-fields/" + field.id, { method: "DELETE" });
    if (resp.res.ok && resp.payload && resp.payload.ok) { await loadSnapshot(); return; }
    if (!resp.payload || resp.payload.data.error !== "reference_field_delete_requires_confirm") {
      alert("Remove failed: " + errorMessage(resp.payload, resp.res));
      return;
    }
    if (!confirm('"' + field.label + '" has content. Delete it anyway?')) return;
    const resp2 = await apiFetch(apiBase + "/reference-fields/" + field.id + "?confirm=true", { method: "DELETE" });
    if (!resp2.res.ok || !resp2.payload || !resp2.payload.ok) { alert("Remove failed: " + errorMessage(resp2.payload, resp2.res)); return; }
    await loadSnapshot();
  }

  return {
    state,
    apiBase,
    affordances,
    apiFetch,
    apiPost,
    errorMessage,
    loadSnapshot,
    applySnapshot,
    loadAssetMeta,
    toggleArchive,
    addColumn,
    insertColumn,
    renameColumn,
    removeColumn,
    addBand,
    editBand,
    toggleBandCollapse,
    toggleBandLock,
    removeBand,
    addRow,
    renameRow,
    removeRow,
    createCard,
    saveCard,
    deleteCard,
    moveCard,
    swapCards,
    uploadCardImage,
    removeCardImage,
    fetchGrants,
    removeGrant,
    addGrant,
    postCoordinationAction,
    saveReferenceFieldContent,
    saveReferenceItemContent,
    addReferenceItem,
    moveReferenceItem,
    removeReferenceItem,
    saveReferenceField,
    changeReferenceFieldType,
    removeReferenceField,
  };
}

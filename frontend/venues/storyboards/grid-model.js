(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryStoryboardGrid = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  // Kernel 81: pure presentation logic for the Storyboards CSS Grid
  // rework, kept separate from board.html's DOM/event code so it can run
  // under node:test (see tests/storyboards/grid-model.test.js). Nothing in
  // here touches the network or the DOM.

  // Ascending capability order -- mirrors backend/internal/storyboards's
  // TierAudience..TierOperator constants exactly (authority.go's
  // tierAtLeastCrew/tierAtLeastDirector/tierIsOwnerOrOperator). This is a
  // display-affordance mirror only: the server re-derives and enforces the
  // real tier on every request (viewer_tier in the snapshot is already
  // server-resolved), so a client that disagrees with this list can at
  // worst show/hide the wrong button, never bypass authority.
  const TIER_ORDER = ["", "audience", "cast", "crew", "director", "producer", "owner", "operator"];

  function tierAtLeast(tier, floor) {
    const t = TIER_ORDER.indexOf(tier);
    const f = TIER_ORDER.indexOf(floor);
    if (t < 0 || f < 0) return false;
    return t >= f;
  }

  // Single source of truth for which controls a viewer's tier affords --
  // board.html reads this instead of scattering tierAtLeast() calls across
  // render code (spec 11).
  function roleAffordances(tier) {
    const canEditCards = tierAtLeast(tier, "crew");
    const canEditStructure = tierAtLeast(tier, "director");
    const canManageSharing = tier === "owner" || tier === "operator";
    return {
      tier: tier,
      canViewHidden: canEditCards,
      canEditCards: canEditCards,
      canDragCards: canEditCards,
      canAttachImage: canEditCards,
      canEditStructure: canEditStructure,
      canInsertColumns: canEditStructure,
      canLockCards: canEditStructure,
      canManageSharing: canManageSharing,
      canArchive: canManageSharing,
      canExport: canEditStructure,
    };
  }

  function cardsByCell(cards) {
    const map = {};
    for (const c of cards || []) {
      const key = c.row_id + "|" + c.column_id;
      (map[key] = map[key] || []).push(c);
    }
    for (const key in map) {
      map[key].sort((a, b) => a.sort_order_in_cell - b.sort_order_in_cell);
    }
    return map;
  }

  // Kernel 81 spec 2.2: the ordinary UI shows exactly one card per cell
  // even though the backend still allows more (Kernel 80's capability,
  // deliberately left intact). This is the tie-break: lowest
  // sort_order_in_cell, matching the array order cardsByCell already
  // produces.
  function visibleCardForCell(cellCards) {
    if (!cellCards || cellCards.length === 0) return null;
    return cellCards[0];
  }

  function rowsByBand(rows) {
    const map = {};
    for (const r of rows || []) {
      (map[r.band_id] = map[r.band_id] || []).push(r);
    }
    for (const key in map) {
      map[key].sort((a, b) => a.sort_order_in_band - b.sort_order_in_band);
    }
    return map;
  }

  // Computes 1-based CSS grid line numbers for every structural element,
  // so board.html can place plain <div>s with inline grid-row/grid-column
  // on one flat grid rather than needing a nested grid per band. Column 1
  // is always the corner/row-label/band-label column; columns 2..N+1 are
  // the board's columns in order. Row 1 is always the header row; then for
  // each band (in sort_order): one band-bar row, one row per member row,
  // then one reserved "add row" spacer line -- a collapsed band
  // contributes zero member-row lines and no spacer line (spec 5.3), so
  // its rows and cards vanish from layout without losing any snapshot data
  // (the caller still has the full row list; it just isn't placed). The
  // spacer line is always reserved, whether or not the viewer's role
  // affords an actual "+ row" button there -- board.html must place
  // *something* (even nothing) at addRowLine rather than recomputing its
  // own line number, or it collides with the next band's bar (found via
  // real browser drag/click testing, not by inspection).
  function computeGridLayout(snapshot) {
    const columns = snapshot.columns || [];
    const bands = (snapshot.bands || []).slice().sort((a, b) => a.sort_order - b.sort_order);
    const byBand = rowsByBand(snapshot.rows || []);

    const columnLine = {};
    columns.forEach((col, i) => {
      columnLine[col.id] = i + 2;
    });

    const rowLine = {};
    const bandHeaderLine = {};
    const bandRowIds = {};
    const addRowLine = {};
    let line = 2; // line 1 is the header row
    for (const band of bands) {
      bandHeaderLine[band.id] = line;
      line += 1;
      const rows = band.is_collapsed ? [] : byBand[band.id] || [];
      bandRowIds[band.id] = rows.map((r) => r.id);
      for (const row of rows) {
        rowLine[row.id] = line;
        line += 1;
      }
      if (!band.is_collapsed) {
        addRowLine[band.id] = line;
        line += 1;
      }
    }

    return {
      lastColumnLine: columns.length + 1,
      lastRowLine: line - 1,
      columnLine: columnLine,
      rowLine: rowLine,
      bandHeaderLine: bandHeaderLine,
      addRowLine: addRowLine,
      bandOrder: bands.map((b) => b.id),
      bandRowIds: bandRowIds,
    };
  }

  // Kernel 81 spec 2.5: color_token is only ever resolved through this
  // fixed whitelist -- never interpolated into a style attribute -- so a
  // crafted token value can never inject arbitrary CSS. Unrecognized or
  // empty tokens fall back to the deliberate default class.
  const COLOR_TOKENS = {
    default: "sb-color-default",
    amber: "sb-color-amber",
    rose: "sb-color-rose",
    crimson: "sb-color-crimson",
    violet: "sb-color-violet",
    teal: "sb-color-teal",
    moss: "sb-color-moss",
    slate: "sb-color-slate",
    gold: "sb-color-gold",
  };

  function colorTokenClass(token) {
    const key = String(token || "").trim().toLowerCase();
    return COLOR_TOKENS[key] || COLOR_TOKENS.default;
  }

  function colorTokenNames() {
    return Object.keys(COLOR_TOKENS);
  }

  // Kernel 81 spec 9.5: a card's pinned image is a gravestone whenever the
  // GET /api/assets/{id} lookup 404'd (hard-deleted -- assetMeta is then
  // null/undefined) or came back with missing_asset true (soft-tombstoned,
  // per assets.tombstoneWarehouseAsset). hasImageRef is
  // card.image_asset_id truthiness -- a card with no reference at all is
  // never a gravestone, just imageless.
  function isImageGravestone(hasImageRef, assetMeta) {
    if (!hasImageRef) return false;
    if (!assetMeta) return true;
    return !!assetMeta.missing_asset;
  }

  // Kernel 81 spec 7.4: what an incoming drag onto a target cell should do,
  // given the cell's current visible card (or null if empty) and the id of
  // the card being dragged.
  function dropResolutionKind(existingVisibleCard, draggedCardId) {
    if (!existingVisibleCard) return "move";
    if (existingVisibleCard.id === draggedCardId) return "noop";
    return "occupied";
  }

  return {
    TIER_ORDER: TIER_ORDER,
    tierAtLeast: tierAtLeast,
    roleAffordances: roleAffordances,
    cardsByCell: cardsByCell,
    visibleCardForCell: visibleCardForCell,
    rowsByBand: rowsByBand,
    computeGridLayout: computeGridLayout,
    COLOR_TOKENS: COLOR_TOKENS,
    colorTokenClass: colorTokenClass,
    colorTokenNames: colorTokenNames,
    isImageGravestone: isImageGravestone,
    dropResolutionKind: dropResolutionKind,
  };
});

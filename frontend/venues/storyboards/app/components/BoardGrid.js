// Kernel 94: the CSS Grid board itself -- column headers, band bars, row
// labels, and cells. Grid line placement still comes from grid-model.js's
// computeGridLayout (unchanged, spec 26 audit: keep proven layout math),
// just applied via Vue :style bindings instead of manual style.gridRow/
// gridColumn assignment.
import { inject, computed, ref, watch, onMounted, onUnmounted } from "/lib/vue.esm-browser.prod.js";
import CardTile from "./CardTile.js";
import { prefersReducedMotion } from "../motion.js";

const G = window.VictoryStoryboardGrid;
const REPOSITION_MS = 220;
const APPEAR_MS = 220;

export default {
  name: "BoardGrid",
  components: { CardTile },
  props: {
    pickCtx: { type: Object, default: null },
  },
  emits: ["open-card", "occupied", "open-image", "pick-cell"],
  setup(props, { emit }) {
    const store = inject("store");
    const snapshot = computed(() => store.state.snapshot);
    const affordances = store.affordances;
    const layout = computed(() => G.computeGridLayout(snapshot.value));
    const byCell = computed(() => G.cardsByCell(snapshot.value.cards));
    const bandsById = computed(() => Object.fromEntries(snapshot.value.bands.map((b) => [b.id, b])));
    const rowsById = computed(() => Object.fromEntries(snapshot.value.rows.map((r) => [r.id, r])));
    const sortedBands = computed(() => layout.value.bandOrder.map((id) => bandsById.value[id]));
    function rowsForBand(bandId) {
      return layout.value.bandRowIds[bandId].map((id) => rowsById.value[id]);
    }
    // Kernel 94 Pass 2 spec 1/7/10: an untouched board (default structure,
    // zero cards) should read as an invitation, not a bordered spreadsheet.
    const isSparse = computed(() => snapshot.value.cards.length === 0);

    // Kernel 94 Pass 4 spec 9/12: "card relocation that visibly preserves
    // identity." Cards live nested inside per-cell containers (spec 26:
    // keeping this matches the proven drop-target hit-testing in
    // CardTile.js), so a card that changes cell is, from Vue's point of
    // view, a different vnode unmounting in the old cell and a new one
    // mounting in the new one -- there is no shared-element identity for
    // Vue's own <TransitionGroup> to animate between. This is the classic
    // FLIP technique instead: measure every card's on-screen rect before
    // the DOM patches (flush: 'pre', the default), then after it patches
    // invert the newly-placed element back to its old screen position with
    // no transition and immediately release it into a transitioning one --
    // it reads as one card sliding, even though it's two different DOM
    // nodes. Also covers band collapse/expand, which reshuffles every
    // card below it without changing any card's row/column id.
    const cardPositionSignature = computed(() => {
      const cards = snapshot.value.cards.map((c) => c.id + ":" + c.row_id + ":" + c.column_id).join("|");
      const bands = snapshot.value.bands.map((b) => b.id + ":" + b.is_collapsed).join("|");
      return cards + "||" + bands;
    });
    let beforeRects = null;
    watch(cardPositionSignature, () => {
      if (prefersReducedMotion()) return;
      beforeRects = new Map();
      document.querySelectorAll("[data-card-id]").forEach((el) => {
        beforeRects.set(el.dataset.cardId, el.getBoundingClientRect());
      });
    });
    watch(cardPositionSignature, () => {
      const before = beforeRects;
      beforeRects = null;
      if (!before) return; // reduced motion, or this is the very first render
      document.querySelectorAll("[data-card-id]").forEach((el) => {
        const id = el.dataset.cardId;
        const prev = before.get(id);
        if (!prev) {
          // A card with no prior on-screen rect is newly created, not
          // moved -- pop it in instead of sliding it from nowhere.
          el.classList.remove("is-appearing");
          void el.offsetWidth; // restart the animation if already applied once this tick
          el.classList.add("is-appearing");
          setTimeout(() => el.classList.remove("is-appearing"), APPEAR_MS);
          return;
        }
        const now = el.getBoundingClientRect();
        const dx = prev.left - now.left;
        const dy = prev.top - now.top;
        if (Math.abs(dx) < 1 && Math.abs(dy) < 1) return;
        el.style.transition = "none";
        el.style.transform = `translate(${dx}px, ${dy}px)`;
        void el.offsetWidth; // force reflow so the instant jump above actually paints first
        requestAnimationFrame(() => {
          el.style.transition = `transform ${REPOSITION_MS}ms ease`;
          el.style.transform = "";
          setTimeout(() => { el.style.transition = ""; }, REPOSITION_MS + 30);
        });
      });
    }, { flush: "post" });

    const openColumnMenuId = ref(null);
    function toggleColumnMenu(colId) {
      openColumnMenuId.value = openColumnMenuId.value === colId ? null : colId;
    }
    function closeColumnMenu() {
      openColumnMenuId.value = null;
    }
    onMounted(() => document.addEventListener("click", closeColumnMenu));
    onUnmounted(() => document.removeEventListener("click", closeColumnMenu));

    // A collapsed band's rows/cards aren't in the DOM at all (v-if below),
    // so a PDF export taken while a band happens to be collapsed would
    // silently drop its content. Force everything open for the duration of
    // the print only -- purely local rendering, no server call, so it
    // doesn't touch the real (shared, persisted) collapse state or fire a
    // WS broadcast to everyone else on the board.
    const forceExpandForPrint = ref(false);
    function onBeforePrint() { forceExpandForPrint.value = true; }
    function onAfterPrint() { forceExpandForPrint.value = false; }
    onMounted(() => {
      window.addEventListener("beforeprint", onBeforePrint);
      window.addEventListener("afterprint", onAfterPrint);
    });
    onUnmounted(() => {
      window.removeEventListener("beforeprint", onBeforePrint);
      window.removeEventListener("afterprint", onAfterPrint);
    });

    function visibleCard(rowId, colId) {
      return G.visibleCardForCell(byCell.value[rowId + "|" + colId]);
    }
    function isOccupied(rowId, colId) {
      return !!visibleCard(rowId, colId);
    }

    async function onCellClick(row, col) {
      if (props.pickCtx) {
        if (isOccupied(row.id, col.id)) return;
        emit("pick-cell", { rowId: row.id, columnId: col.id });
        return;
      }
    }

    async function onCardMoved({ card, targetRowId, targetColumnId }) {
      const result = await store.moveCard(card, targetRowId, targetColumnId);
      if (result && result.ok === false) {
        alert("Could not move card: " + result.error);
      }
    }

    async function insertColumn(col, side) { await store.insertColumn(col, side); }
    async function renameColumn(col) { await store.renameColumn(col); }
    async function removeColumn(col) { await store.removeColumn(col); }
    async function editBand(band) { await store.editBand(band); }
    async function toggleBandCollapse(band) { await store.toggleBandCollapse(band); }
    async function toggleBandLock(band) { await store.toggleBandLock(band); }
    async function removeBand(band) { await store.removeBand(band); }
    async function renameRow(row) { await store.renameRow(row); }
    async function removeRow(row) { await store.removeRow(row); }
    async function addRow(band) { await store.addRow(band); }
    async function createCard(row, col) { await store.createCard(row, col); }

    // Middle-mouse panning (Kernel 82 spec 6) -- pure scroll-position
    // panning, no zoom/minimap, left drag stays owned by card drag/drop.
    const scrollerEl = ref(null);
    onMounted(() => {
      const scroller = scrollerEl.value;
      if (!scroller) return;
      let panning = false;
      let startX = 0, startY = 0, startScrollLeft = 0, startScrollTop = 0;
      scroller.addEventListener("pointerdown", (e) => {
        if (e.button !== 1) return;
        panning = true;
        startX = e.clientX; startY = e.clientY;
        startScrollLeft = scroller.scrollLeft; startScrollTop = scroller.scrollTop;
        try { scroller.setPointerCapture(e.pointerId); } catch (err) { /* ignore */ }
        scroller.style.cursor = "grabbing";
        e.preventDefault();
      });
      scroller.addEventListener("pointermove", (e) => {
        if (!panning) return;
        scroller.scrollLeft = startScrollLeft - (e.clientX - startX);
        scroller.scrollTop = startScrollTop - (e.clientY - startY);
        e.preventDefault();
      });
      function endPan(e) {
        if (!panning) return;
        panning = false;
        scroller.style.cursor = "";
        try { scroller.releasePointerCapture(e.pointerId); } catch (err) { /* ignore */ }
      }
      scroller.addEventListener("pointerup", endPan);
      scroller.addEventListener("pointercancel", endPan);
      scroller.addEventListener("mousedown", (e) => { if (e.button === 1) e.preventDefault(); });
      scroller.addEventListener("auxclick", (e) => { if (e.button === 1) e.preventDefault(); });
    });

    return {
      snapshot, affordances, layout, byCell, G, isSparse, forceExpandForPrint,
      bandsById, rowsById, sortedBands, rowsForBand,
      openColumnMenuId, toggleColumnMenu,
      visibleCard, isOccupied, onCellClick, onCardMoved,
      insertColumn, renameColumn, removeColumn,
      editBand, toggleBandCollapse, toggleBandLock, removeBand,
      renameRow, removeRow, addRow, createCard,
      scrollerEl,
    };
  },
  template: `
    <div class="board-column">
      <div v-if="isSparse" class="board-invite">A blank stage. Place your first card to begin.</div>
      <div class="board-scroll" id="board-scroll" ref="scrollerEl" :class="{ 'is-sparse': isSparse }">
      <div class="board-grid" id="board-grid" :style="{ '--col-count': String(snapshot.columns.length || 1) }">
        <div class="corner-cell" style="grid-row:1;grid-column:1;"></div>

        <div
          v-for="col in snapshot.columns"
          :key="col.id"
          class="col-header"
          :class="{ 'col-header-boundary': col.column_role === 'beginning' || col.column_role === 'ending' }"
          :style="{ gridRow: 1, gridColumn: layout.columnLine[col.id] }"
        >
          <div style="min-width:0;">
            <span v-if="col.column_role === 'beginning'" class="reference-field-type-tag">Beginning</span>
            <span v-else-if="col.column_role === 'ending'" class="reference-field-type-tag">Ending</span>
            <span class="col-header-title" :title="col.title">{{ col.title }}</span>
          </div>
          <div v-if="affordances.canInsertColumns" class="col-menu-wrap">
            <button class="col-menu-btn small" title="Column options" @click.stop="toggleColumnMenu(col.id)">⋮</button>
            <div class="col-menu-panel" :hidden="openColumnMenuId !== col.id">
              <button v-if="col.column_role !== 'beginning'" class="small" @click="openColumnMenuId = null; insertColumn(col, 'left')">Insert left</button>
              <button v-if="col.column_role !== 'ending'" class="small" @click="openColumnMenuId = null; insertColumn(col, 'right')">Insert right</button>
              <button class="small" @click="openColumnMenuId = null; renameColumn(col)">Rename</button>
              <button v-if="col.column_role === 'ordinary'" class="small" @click="openColumnMenuId = null; removeColumn(col)">Remove</button>
            </div>
          </div>
        </div>

        <template v-for="band in sortedBands" :key="band.id">
          <div
            class="band-bar"
            :style="{ gridColumn: '1 / -1', gridRow: layout.bandHeaderLine[band.id] }"
          >
            <div class="band-bar-inner">
              <span class="band-chip">
                <button class="band-collapse-btn small" :title="band.is_collapsed ? 'Expand band' : 'Collapse band'" @click="toggleBandCollapse(band)">{{ band.is_collapsed ? "▸" : "▾" }}</button>
                <span>{{ band.label }}<template v-if="band.is_locked"> 🔒</template></span>
              </span>
              <span class="band-actions">
                <button v-if="affordances.canEditCards && (affordances.canEditStructure || !band.is_locked)" class="small" @click="editBand(band)">Edit</button>
                <template v-if="affordances.canEditStructure">
                  <button class="small" @click="toggleBandLock(band)">{{ band.is_locked ? "Unlock" : "Lock" }}</button>
                  <button class="small danger" @click="removeBand(band)">Remove</button>
                </template>
              </span>
            </div>
          </div>

          <template v-if="!band.is_collapsed || forceExpandForPrint">
            <template v-for="row in rowsForBand(band.id)" :key="row.id">
              <div class="row-label" :style="{ gridRow: layout.rowLine[row.id], gridColumn: 1 }">
                <span class="row-label-title" :title="row.label">{{ row.label }}</span>
                <template v-if="affordances.canEditStructure">
                  <button class="icon-btn" @click="renameRow(row)">✎</button>
                  <button class="icon-btn danger" title="Remove row" @click="removeRow(row)">×</button>
                </template>
              </div>

              <div
                v-for="col in snapshot.columns"
                :key="col.id"
                class="cell"
                :class="{ 'is-pick-target': pickCtx && !isOccupied(row.id, col.id) }"
                :style="{ gridRow: layout.rowLine[row.id], gridColumn: layout.columnLine[col.id] }"
                :data-row-id="row.id"
                :data-column-id="col.id"
                @click="onCellClick(row, col)"
              >
                <CardTile
                  v-if="visibleCard(row.id, col.id)"
                  :card="visibleCard(row.id, col.id)"
                  :affordances="affordances"
                  :snapshot="snapshot"
                  @open="$emit('open-card', $event)"
                  @open-image="$emit('open-image', $event)"
                  @occupied="$emit('occupied', $event)"
                  @moved="onCardMoved"
                />
                <button
                  v-else-if="affordances.canEditCards && !pickCtx"
                  type="button"
                  class="cell-empty-add"
                  @click.stop="createCard(row, col)"
                ><span class="cell-empty-add-glyph">+</span><span class="cell-empty-add-label">card</span></button>
              </div>
            </template>

            <div
              v-if="affordances.canEditStructure"
              class="add-row-cell"
              :style="{ gridRow: layout.addRowLine[band.id], gridColumn: '1 / -1', padding: '4px 10px' }"
            >
              <button class="small" @click="addRow(band)">+ row in {{ band.label }}</button>
            </div>
          </template>
        </template>
      </div>
      </div>
    </div>
  `,
};

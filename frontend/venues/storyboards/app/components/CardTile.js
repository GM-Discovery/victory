// Kernel 94: one Storyboards card. Pointer-based drag/drop is ported
// verbatim from the pre-Vue board.html (spec 12: dragging must remain
// physical/understandable) -- it manipulates the ghost element and the
// hovered `.cell`'s highlight classes directly via the DOM rather than
// through Vue reactivity, same as before, since drop-target detection is
// inherently a raw hit-test (document.elementFromPoint) against cells that
// belong to a sibling component, not this one.
import { ref } from "/lib/vue.esm-browser.prod.js";
import { prefersReducedMotion } from "../motion.js";

const G = window.VictoryStoryboardGrid;
const FLIP_ANIM_MS = 280;

export default {
  name: "CardTile",
  props: {
    card: { type: Object, required: true },
    affordances: { type: Object, required: true },
    snapshot: { type: Object, required: true },
  },
  emits: ["open", "occupied", "moved", "open-image"],
  setup(props, { emit }) {
    const flipped = ref(false);
    const isFlipping = ref(false);
    const el = ref(null);
    const suppressClick = ref(false);

    // Kernel 94 spec 9: "card flip" is one of the named-appropriate motions.
    // A real 3D flip (backface-visibility + perspective) fights this card's
    // own overflow:hidden/box-shadow -- fragile cleverness the spec
    // explicitly warns against. A scaleX squash to ~0 with the content swap
    // at the midpoint reads as a flip and never touches overflow/3D.
    function toggleFlip() {
      if (prefersReducedMotion()) { flipped.value = !flipped.value; return; }
      if (isFlipping.value) return;
      isFlipping.value = true;
      setTimeout(() => { flipped.value = !flipped.value; }, FLIP_ANIM_MS / 2);
      setTimeout(() => { isFlipping.value = false; }, FLIP_ANIM_MS);
    }

    let dragState = null;
    let ghostEl = null;
    let currentDropTarget = null;

    function canDrag() {
      if (!props.affordances.canDragCards) return false;
      if (!props.card.is_locked) return true;
      return props.affordances.canEditStructure;
    }

    function onPointerDown(e) {
      if (!canDrag()) return;
      if (e.button !== 0 && e.pointerType === "mouse") return;
      dragState = {
        cardId: props.card.id,
        startX: e.clientX, startY: e.clientY,
        pointerId: e.pointerId, dragging: false,
      };
    }

    function onPointerMove(e) {
      if (!dragState || dragState.cardId !== props.card.id) return;
      const dx = e.clientX - dragState.startX;
      const dy = e.clientY - dragState.startY;
      if (!dragState.dragging) {
        if (Math.hypot(dx, dy) < 6) return;
        dragState.dragging = true;
        try { el.value.setPointerCapture(dragState.pointerId); } catch (err) { /* ignore */ }
        startGhost(e);
        el.value.classList.add("is-dragging");
      }
      updateGhost(e);
      autoScrollNearEdges(e);
    }

    async function onPointerUp(e) {
      if (!dragState || dragState.cardId !== props.card.id) return;
      const wasDragging = dragState.dragging;
      if (wasDragging) {
        suppressClick.value = true;
        await finishDrag();
        el.value.classList.remove("is-dragging");
      }
      try { el.value.releasePointerCapture(dragState.pointerId); } catch (err) { /* ignore */ }
      dragState = null;
    }

    function onPointerCancel() {
      cancelGhost();
      if (dragState && el.value) el.value.classList.remove("is-dragging");
      dragState = null;
    }

    function startGhost(e) {
      ghostEl = el.value.cloneNode(true);
      ghostEl.classList.add("drag-ghost");
      ghostEl.classList.remove("is-dragging");
      ghostEl.style.left = e.clientX + "px";
      ghostEl.style.top = e.clientY + "px";
      document.body.appendChild(ghostEl);
    }
    function updateGhost(e) {
      if (!ghostEl) return;
      ghostEl.style.left = e.clientX + "px";
      ghostEl.style.top = e.clientY + "px";
      const hit = document.elementFromPoint(e.clientX, e.clientY);
      const cell = hit ? hit.closest(".cell") : null;
      if (cell !== currentDropTarget) {
        if (currentDropTarget) currentDropTarget.classList.remove("is-drop-target", "is-drop-occupied");
        currentDropTarget = cell;
        if (cell) {
          const occupied = !!G.visibleCardForCell(
            G.cardsByCell(props.snapshot.cards)[cell.dataset.rowId + "|" + cell.dataset.columnId]
          );
          cell.classList.add(occupied ? "is-drop-occupied" : "is-drop-target");
        }
      }
    }
    function cancelGhost() {
      if (ghostEl) { ghostEl.remove(); ghostEl = null; }
      if (currentDropTarget) { currentDropTarget.classList.remove("is-drop-target", "is-drop-occupied"); currentDropTarget = null; }
    }
    function autoScrollNearEdges(e) {
      const scroller = document.getElementById("board-scroll");
      if (!scroller) return;
      const rect = scroller.getBoundingClientRect();
      const margin = 36;
      if (e.clientX - rect.left < margin) scroller.scrollLeft -= 14;
      else if (rect.right - e.clientX < margin) scroller.scrollLeft += 14;
      if (e.clientY - rect.top < margin) scroller.scrollTop -= 14;
      else if (rect.bottom - e.clientY < margin) scroller.scrollTop += 14;
    }

    async function finishDrag() {
      const target = currentDropTarget;
      cancelGhost();
      if (!target) return;
      const targetRowId = target.dataset.rowId;
      const targetColumnId = target.dataset.columnId;
      const cellCards = G.cardsByCell(props.snapshot.cards)[targetRowId + "|" + targetColumnId];
      const existing = G.visibleCardForCell(cellCards);
      const kind = G.dropResolutionKind(existing, props.card.id);
      if (kind === "noop") return;
      if (kind === "move") {
        emit("moved", { card: props.card, targetRowId, targetColumnId });
        return;
      }
      emit("occupied", { draggedCard: props.card, existingCard: existing, targetRowId, targetColumnId });
    }

    function onClick(e) {
      if (suppressClick.value) { suppressClick.value = false; return; }
      if (e.target.closest(".sb-card-flip-btn, .sb-card-move-btn, .sb-card-thumb-wrap img")) return;
      emit("open", props.card);
    }
    function onKeydown(e) {
      if (e.key === "Enter" || e.key === " ") { e.preventDefault(); emit("open", props.card); }
    }

    return { flipped, isFlipping, toggleFlip, el, canDrag, onPointerDown, onPointerMove, onPointerUp, onPointerCancel, onClick, onKeydown, G };
  },
  template: `
    <div
      ref="el"
      class="sb-card"
      :class="[G.colorTokenClass(card.color_token), { 'is-locked': card.is_locked, 'is-flipped': flipped, 'is-flipping': isFlipping }]"
      tabindex="0"
      role="button"
      :data-card-id="card.id"
      @pointerdown="onPointerDown"
      @pointermove="onPointerMove"
      @pointerup="onPointerUp"
      @pointercancel="onPointerCancel"
      @click="onClick"
      @keydown="onKeydown"
    >
      <div class="sb-card-badges">
        <span v-if="card.is_locked" class="sb-card-badge">locked</span>
        <span v-if="card.hidden_from_audience" class="sb-card-badge">hidden</span>
      </div>

      <div class="sb-card-front">
        <div v-if="card.image_asset_id" class="sb-card-thumb-wrap">
          <img
            :src="'/api/assets/' + encodeURIComponent(card.image_asset_id) + '/content?variant=thumbnail'"
            :alt="'Card image for ' + (card.title || 'untitled card')"
            loading="lazy"
            @click.stop="$emit('open-image', card.image_asset_id)"
          />
        </div>
        <div class="sb-card-title">{{ card.title || "(untitled)" }}</div>
        <div v-if="card.category" class="sb-card-category">{{ card.category }}</div>
        <div v-if="card.front_text" class="sb-card-front-text">{{ card.front_text }}</div>
      </div>

      <div class="sb-card-back">
        <div class="sb-card-back-label">Back</div>
        {{ card.back_text || "(no back text)" }}
      </div>

      <button type="button" class="sb-card-flip-btn" title="Flip card" @click.stop="toggleFlip">⟲</button>
      <button
        v-if="canDrag()"
        type="button"
        class="sb-card-move-btn"
        title="Drag to move, or open card to use the Move fallback"
        @click.stop="$emit('open', card)"
      >⠿</button>
    </div>
  `,
};

import { provide, ref, onMounted, onUnmounted } from "/lib/vue.esm-browser.prod.js";
import { createBoardStore } from "./store.js";
import Toolbar from "./components/Toolbar.js";
import PresenceTray from "./components/PresenceTray.js";
import BoardGrid from "./components/BoardGrid.js";
import ReferencePanel from "./components/ReferencePanel.js";
import CardModal from "./components/CardModal.js";
import OccupiedModal from "./components/OccupiedModal.js";
import PickBanner from "./components/PickBanner.js";
import Lightbox from "./components/Lightbox.js";
import ShareModal from "./components/ShareModal.js";

export default {
  name: "AppRoot",
  components: {
    Toolbar, PresenceTray, BoardGrid, ReferencePanel,
    CardModal, OccupiedModal, PickBanner, Lightbox, ShareModal,
  },
  props: {
    boardId: { type: String, required: true },
  },
  setup(props) {
    const store = createBoardStore(props.boardId);
    provide("store", store);

    const activeCard = ref(null);
    const occupiedCtx = ref(null);
    const pickCtx = ref(null);
    const shareOpen = ref(false);
    const lightboxAssetId = ref("");

    function openCard(card) { activeCard.value = card; }
    function closeCard() { activeCard.value = null; }
    function openOccupied(ctx) { occupiedCtx.value = ctx; }
    function closeOccupied() { occupiedCtx.value = null; }
    function startPick(ctx) { occupiedCtx.value = null; pickCtx.value = ctx; }
    async function onPickCell({ rowId, columnId }) {
      const ctx = pickCtx.value;
      pickCtx.value = null;
      if (!ctx) return;
      const moveExisting = await store.moveCard(ctx.existingCard, rowId, columnId);
      if (!moveExisting.ok) {
        alert("Could not relocate existing card: " + moveExisting.error);
        await store.loadSnapshot();
        return;
      }
      const moveIncoming = await store.moveCard(ctx.draggedCard, ctx.targetRowId, ctx.targetColumnId);
      if (!moveIncoming.ok) {
        alert("Could not place the dragged card: " + moveIncoming.error);
      }
    }
    function cancelPick() {
      pickCtx.value = null;
      store.loadSnapshot();
    }
    function openImage(assetId) { lightboxAssetId.value = assetId; }
    function closeImage() { lightboxAssetId.value = ""; }

    function onKeydown(e) {
      if (e.key !== "Escape") return;
      if (pickCtx.value) cancelPick();
      if (lightboxAssetId.value) closeImage();
    }

    let socketHandle = null;
    onMounted(async () => {
      document.addEventListener("keydown", onKeydown);
      const ok = await store.loadSnapshot();
      if (!ok) return;
      socketHandle = window.watchStoryboard(props.boardId, {
        onSnapshot: (data) => store.applySnapshot(data),
        onEvent: () => store.loadSnapshot(),
        onError: (err) => console.warn("storyboard ws error", err),
      });
      window.addEventListener("beforeunload", () => { if (socketHandle) socketHandle.close(); });
    });
    onUnmounted(() => {
      document.removeEventListener("keydown", onKeydown);
      if (socketHandle) socketHandle.close();
    });

    return {
      store, activeCard, occupiedCtx, pickCtx, shareOpen, lightboxAssetId,
      openCard, closeCard, openOccupied, closeOccupied, startPick, onPickCell, cancelPick,
      openImage, closeImage,
    };
  },
  template: `
    <template v-if="store.state.snapshot">
      <Toolbar @share="shareOpen = true" />
      <PresenceTray />
      <div id="board-root">
        <BoardGrid
          :pick-ctx="pickCtx"
          @open-card="openCard"
          @open-image="openImage"
          @occupied="openOccupied"
          @pick-cell="onPickCell"
        />
        <ReferencePanel />
      </div>

      <CardModal :card="activeCard" @close="closeCard" @occupied="openOccupied" @open-image="openImage" />
      <OccupiedModal :ctx="occupiedCtx" @close="closeOccupied" @pick="startPick" />
      <PickBanner :active="!!pickCtx" @cancel="cancelPick" />
      <Lightbox :asset-id="lightboxAssetId" @close="closeImage" />
      <ShareModal :open="shareOpen" @close="shareOpen = false" />
    </template>
    <div v-else class="status-line" style="padding:18px;">Loading…</div>
  `,
};

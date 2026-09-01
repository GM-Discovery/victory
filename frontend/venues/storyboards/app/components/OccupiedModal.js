import { inject } from "/lib/vue.esm-browser.prod.js";

export default {
  name: "OccupiedModal",
  props: {
    ctx: { type: Object, default: null },
  },
  emits: ["close", "pick"],
  setup(props, { emit }) {
    const store = inject("store");
    async function swap() {
      const ctx = props.ctx;
      emit("close");
      await store.swapCards(ctx.draggedCard, ctx.existingCard);
    }
    function moveExisting() {
      emit("pick", props.ctx);
    }
    function cancel() {
      emit("close");
      store.loadSnapshot();
    }
    return { swap, moveExisting, cancel };
  },
  template: `
    <Transition name="modal">
    <div class="modal-backdrop" v-if="ctx">
      <div class="modal" style="width:min(400px,92vw);">
        <h2>Cell occupied</h2>
        <p class="status-line">This cell already has "{{ ctx.existingCard.title || "untitled" }}". What would you like to do?</p>
        <div class="modal-actions">
          <button @click="swap">Swap</button>
          <button @click="moveExisting">Move existing…</button>
          <button @click="cancel">Cancel</button>
        </div>
      </div>
    </div>
    </Transition>
  `,
};

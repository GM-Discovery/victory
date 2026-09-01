export default {
  name: "PickBanner",
  props: {
    active: { type: Boolean, default: false },
  },
  emits: ["cancel"],
  template: `
    <div class="pick-banner" v-if="active">
      <span>Click an empty cell for the existing card…</span>
      <button class="small" @click="$emit('cancel')">Cancel (Esc)</button>
    </div>
  `,
};

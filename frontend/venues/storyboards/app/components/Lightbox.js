export default {
  name: "Lightbox",
  props: {
    assetId: { type: String, default: "" },
  },
  emits: ["close"],
  template: `
    <Transition name="modal">
    <div class="lightbox-backdrop" v-if="assetId" @click="$event.target.classList.contains('lightbox-backdrop') && $emit('close')">
      <button class="lightbox-close" aria-label="Close" @click="$emit('close')">✕</button>
      <img :src="'/api/assets/' + encodeURIComponent(assetId) + '/content?variant=original'" alt="Full-size card image" />
    </div>
    </Transition>
  `,
};

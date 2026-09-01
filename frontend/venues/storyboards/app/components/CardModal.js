import { inject, ref, reactive, watch, nextTick } from "/lib/vue.esm-browser.prod.js";

const G = window.VictoryStoryboardGrid;

export default {
  name: "CardModal",
  props: {
    card: { type: Object, default: null },
  },
  emits: ["close", "occupied", "open-image"],
  setup(props, { emit }) {
    const store = inject("store");
    const form = reactive({
      title: "", front_text: "", back_text: "", category: "", color_token: "",
      hidden_from_audience: false, is_locked: false,
      move_row_id: "", move_column_id: "",
    });
    const message = ref("");
    const messageOk = ref(false);
    const imageStatus = ref("");
    const imageStatusOk = ref(false);
    const assetMeta = ref(null);
    const linkContainer = ref(null);
    const fileInput = ref(null);

    watch(() => props.card, async (card) => {
      message.value = "";
      imageStatus.value = "";
      assetMeta.value = null;
      if (!card) return;
      form.title = card.title || "";
      form.front_text = card.front_text || "";
      form.back_text = card.back_text || "";
      form.category = card.category || "";
      form.color_token = card.color_token && card.color_token !== "default" ? card.color_token : "";
      form.hidden_from_audience = !!card.hidden_from_audience;
      form.is_locked = !!card.is_locked;
      form.move_row_id = card.row_id;
      form.move_column_id = card.column_id;
      if (card.image_asset_id) {
        assetMeta.value = await store.loadAssetMeta(card.image_asset_id);
      }
      await nextTick();
      if (linkContainer.value) {
        linkContainer.value.innerHTML = "";
        if (window.attachEwriteRuleLink) {
          window.attachEwriteRuleLink(linkContainer.value, { objectType: "storyboard_card", objectId: card.id });
        }
      }
    }, { immediate: true });

    function canEdit() {
      if (!props.card) return false;
      return store.affordances.value.canEditCards && (!props.card.is_locked || store.affordances.value.canEditStructure);
    }
    function isGravestone() {
      return G.isImageGravestone(!!(props.card && props.card.image_asset_id), assetMeta.value);
    }

    function openFilePicker() {
      fileInput.value.click();
    }
    async function onFileChange(e) {
      const file = e.target.files && e.target.files[0];
      e.target.value = "";
      if (!file || !props.card) return;
      imageStatus.value = "Uploading…";
      imageStatusOk.value = false;
      const result = await store.uploadCardImage(props.card, file);
      if (!result.ok) {
        imageStatusOk.value = false;
        imageStatus.value = "Upload failed: " + result.error;
        return;
      }
      assetMeta.value = null;
      imageStatusOk.value = true;
      imageStatus.value = "Image attached.";
    }
    async function removeImage() {
      if (!props.card) return;
      const result = await store.removeCardImage(props.card);
      if (!result.ok) {
        imageStatusOk.value = false;
        imageStatus.value = "Remove failed: " + result.error;
        return;
      }
      assetMeta.value = null;
    }

    async function save() {
      const result = await store.saveCard(props.card, form);
      if (!result.ok) {
        messageOk.value = false;
        message.value = "Save failed: " + result.error;
        return;
      }
      emit("close");
    }
    async function del() {
      if (!confirm("Delete this card?")) return;
      const result = await store.deleteCard(props.card);
      if (!result.ok) {
        messageOk.value = false;
        message.value = "Delete failed: " + result.error;
        return;
      }
      emit("close");
    }
    async function moveTo() {
      const cellCards = G.cardsByCell(store.state.snapshot.cards)[form.move_row_id + "|" + form.move_column_id];
      const existing = G.visibleCardForCell(cellCards);
      const kind = G.dropResolutionKind(existing, props.card.id);
      if (kind === "noop") { emit("close"); return; }
      if (kind === "occupied") {
        emit("occupied", { draggedCard: props.card, existingCard: existing, targetRowId: form.move_row_id, targetColumnId: form.move_column_id });
        emit("close");
        return;
      }
      const result = await store.moveCard(props.card, form.move_row_id, form.move_column_id);
      if (!result.ok) {
        messageOk.value = false;
        message.value = "Move failed: " + result.error;
        return;
      }
      emit("close");
    }

    return {
      store, form, message, messageOk, imageStatus, imageStatusOk, assetMeta,
      linkContainer, fileInput, canEdit, isGravestone,
      openFilePicker, onFileChange, removeImage, save, del, moveTo, G,
    };
  },
  template: `
    <Transition name="modal">
    <div class="modal-backdrop" v-if="card">
      <div class="modal">
        <h2>Card</h2>
        <div class="field"><label>Title</label><input v-model="form.title" :disabled="!canEdit()" /></div>
        <div class="field"><label>Front</label><textarea v-model="form.front_text" rows="3" :disabled="!canEdit()"></textarea></div>
        <div class="field"><label>Back</label><textarea v-model="form.back_text" rows="3" :disabled="!canEdit()"></textarea></div>
        <div class="field"><label>Category</label><input v-model="form.category" :disabled="!canEdit()" /></div>
        <div class="field">
          <label>Color</label>
          <select v-model="form.color_token" :disabled="!canEdit()">
            <option v-for="name in G.colorTokenNames()" :key="name" :value="name === 'default' ? '' : name">{{ name.charAt(0).toUpperCase() + name.slice(1) }}</option>
          </select>
        </div>
        <div class="field">
          <label>Image</label>
          <div class="image-field-row">
            <div class="image-field-thumb">
              <span v-if="!card.image_asset_id" class="gravestone-label">No image</span>
              <span v-else-if="isGravestone()" class="gravestone-label">Image unavailable (removed)</span>
              <img
                v-else
                :src="'/api/assets/' + encodeURIComponent(card.image_asset_id) + '/content?variant=thumbnail'"
                alt="Card image"
                @click="$emit('open-image', card.image_asset_id)"
              />
            </div>
            <input type="file" ref="fileInput" accept="image/png,image/jpeg,image/webp,image/gif" hidden @change="onFileChange" />
            <button v-if="canEdit()" type="button" class="small" @click="openFilePicker">Choose…</button>
            <button v-if="canEdit() && card.image_asset_id" type="button" class="small danger" @click="removeImage">Remove</button>
          </div>
          <div class="status-line" :class="imageStatus ? (imageStatusOk ? 'form-ok' : 'form-error') : ''">{{ imageStatus }}</div>
        </div>
        <div class="field" v-if="store.affordances.value.canEditStructure">
          <label class="checkbox-field"><input type="checkbox" v-model="form.hidden_from_audience" /> Hidden from Audience/Cast</label>
        </div>
        <div class="field" v-if="store.affordances.value.canEditStructure">
          <label class="checkbox-field"><input type="checkbox" v-model="form.is_locked" /> Locked</label>
        </div>
        <div class="field">
          <label>Move to (keyboard-accessible fallback)</label>
          <div style="display:flex; gap:6px; flex-wrap: wrap;">
            <select v-model="form.move_row_id">
              <option v-for="r in store.state.snapshot.rows" :key="r.id" :value="r.id">{{ r.label }}</option>
            </select>
            <select v-model="form.move_column_id">
              <option v-for="c in store.state.snapshot.columns" :key="c.id" :value="c.id">{{ c.title }}</option>
            </select>
            <button class="small" :disabled="!canEdit()" @click="moveTo">Move</button>
          </div>
        </div>
        <div class="field">
          <label>Rule link</label>
          <div ref="linkContainer"></div>
        </div>
        <div :class="message ? (messageOk ? 'form-ok' : 'form-error') : ''">{{ message }}</div>
        <div class="modal-actions">
          <button v-if="canEdit()" @click="save">Save</button>
          <button v-if="canEdit()" class="danger" @click="del">Delete</button>
          <button @click="$emit('close')">Close</button>
        </div>
      </div>
    </div>
    </Transition>
  `,
};

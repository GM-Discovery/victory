import { inject, ref, watch } from "/lib/vue.esm-browser.prod.js";

export default {
  name: "ReferenceFieldModal",
  props: {
    // { mode: "add" | "rename", field: object|null } or null when closed
    ctx: { type: Object, default: null },
  },
  emits: ["close"],
  setup(props, { emit }) {
    const store = inject("store");
    const label = ref("");
    const fieldType = ref("short_text");
    const sublabelA = ref("");
    const sublabelB = ref("");
    const message = ref("");

    watch(() => props.ctx, (ctx) => {
      message.value = "";
      if (!ctx) return;
      label.value = ctx.field ? ctx.field.label : "";
      fieldType.value = ctx.field ? ctx.field.field_type : "short_text";
      sublabelA.value = ctx.field ? (ctx.field.sublabel_a || "") : "";
      sublabelB.value = ctx.field ? (ctx.field.sublabel_b || "") : "";
    }, { immediate: true });

    async function save() {
      const trimmed = label.value.trim();
      if (!trimmed) {
        message.value = "A label is required.";
        return;
      }
      const body = { label: trimmed, sublabel_a: sublabelA.value, sublabel_b: sublabelB.value };
      if (props.ctx.mode === "add") body.field_type = fieldType.value;
      const result = await store.saveReferenceField(props.ctx.mode, props.ctx.field, body);
      if (!result.ok) {
        message.value = "Save failed: " + result.error;
        return;
      }
      emit("close");
    }

    return { label, fieldType, sublabelA, sublabelB, message, save };
  },
  template: `
    <Transition name="modal">
    <div class="modal-backdrop" v-if="ctx">
      <div class="modal" style="width:min(420px,92vw);">
        <h2>{{ ctx.mode === "add" ? "Add Field" : "Edit Field" }}</h2>
        <div class="field"><label>Label</label><input v-model="label" /></div>
        <div class="field" v-if="ctx.mode === 'add'">
          <label>Type</label>
          <select v-model="fieldType">
            <option value="short_text">Short text</option>
            <option value="long_text">Long text</option>
            <option value="list">List</option>
            <option value="paired_list">Paired list</option>
          </select>
        </div>
        <div class="field" v-if="fieldType === 'paired_list'">
          <label>Sublabels</label>
          <div style="display:flex; gap:6px;">
            <input v-model="sublabelA" placeholder="e.g. Include" />
            <input v-model="sublabelB" placeholder="e.g. Exclude" />
          </div>
        </div>
        <div class="form-error">{{ message }}</div>
        <div class="modal-actions">
          <button @click="save">Save</button>
          <button @click="$emit('close')">Close</button>
        </div>
      </div>
    </div>
    </Transition>
  `,
};

import { inject, ref, computed, onMounted, onUnmounted } from "/lib/vue.esm-browser.prod.js";
import ReferenceField from "./ReferenceField.js";
import ReferenceFieldModal from "./ReferenceFieldModal.js";

const COLLAPSE_KEY_PREFIX = "sb-refpanel-collapsed-";

export default {
  name: "ReferencePanel",
  components: { ReferenceField, ReferenceFieldModal },
  setup() {
    const store = inject("store");
    const collapsed = ref(localStorage.getItem(COLLAPSE_KEY_PREFIX + store.state.boardId) === "1");
    const fieldModalCtx = ref(null);

    // Same reasoning as BoardGrid's forceExpandForPrint: a collapsed panel
    // means its content isn't rendered at all, so a PDF taken while it's
    // collapsed would just be missing it. Force it open for the print only.
    const forcePrint = ref(false);
    function onBeforePrint() { forcePrint.value = true; }
    function onAfterPrint() { forcePrint.value = false; }
    onMounted(() => {
      window.addEventListener("beforeprint", onBeforePrint);
      window.addEventListener("afterprint", onAfterPrint);
    });
    onUnmounted(() => {
      window.removeEventListener("beforeprint", onBeforePrint);
      window.removeEventListener("afterprint", onAfterPrint);
    });

    function setCollapsed(value) {
      collapsed.value = value;
      localStorage.setItem(COLLAPSE_KEY_PREFIX + store.state.boardId, value ? "1" : "0");
    }

    const coord = computed(() => store.state.snapshot.coordination);
    const coordSlotVisible = computed(() => {
      const c = coord.value;
      return !!(c && c.active && (c.group_leader_user_id || c.current_turn_user_id));
    });

    function openAddField() { fieldModalCtx.value = { mode: "add", field: null }; }
    function openRenameField(field) { fieldModalCtx.value = { mode: "rename", field }; }
    async function changeType(field) {
      const options = ["short_text", "long_text", "list", "paired_list"];
      const choice = prompt(
        'New type for "' + field.label + '" (short_text, long_text, list, paired_list). The field must be empty to change type.',
        field.field_type
      );
      if (choice === null) return;
      if (!options.includes(choice.trim())) { alert("Not a valid type."); return; }
      await store.changeReferenceFieldType(field, choice.trim());
    }
    async function removeField(field) { await store.removeReferenceField(field); }

    return {
      store, collapsed, setCollapsed, forcePrint, coordSlotVisible, coord,
      fieldModalCtx, openAddField, openRenameField, changeType, removeField,
    };
  },
  template: `
    <template v-if="(store.state.snapshot.reference_fields || []).length || store.affordances.value.canEditStructure">
      <button v-if="collapsed && !forcePrint" id="reference-panel-collapsed-toggle" class="small" @click="setCollapsed(false)">Reference Panel »</button>
      <div v-else class="reference-panel">
        <div class="reference-panel-header">
          <span>Reference Panel</span>
          <div style="display:flex; gap:4px;">
            <button v-if="store.affordances.value.canEditStructure" class="icon-btn" title="Add field" @click="openAddField">+ Field</button>
            <button class="icon-btn" title="Collapse" @click="setCollapsed(true)">«</button>
          </div>
        </div>
        <div>
          <div v-if="coordSlotVisible" class="reference-coordination-slot">
            <div v-if="coord.group_leader_user_id">
              <strong>Group Leader: </strong>{{ coord.group_leader_display_name || coord.group_leader_handle || "Someone" }}
              <span v-if="!coord.group_leader_present" class="coordination-absent-note"> (absent)</span>
            </div>
            <div v-if="coord.current_turn_user_id">
              <strong>Current Turn: </strong>{{ coord.current_turn_display_name || coord.current_turn_handle || "Someone" }}
              <span v-if="!coord.current_turn_present" class="coordination-absent-note"> (absent)</span>
            </div>
          </div>
          <ReferenceField
            v-for="field in (store.state.snapshot.reference_fields || [])"
            :key="field.id"
            :field="field"
            @rename="openRenameField"
            @change-type="changeType"
            @remove="removeField"
          />
        </div>
      </div>
      <ReferenceFieldModal :ctx="fieldModalCtx" @close="fieldModalCtx = null" />
    </template>
  `,
};

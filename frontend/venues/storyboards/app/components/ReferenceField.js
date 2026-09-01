import { inject } from "/lib/vue.esm-browser.prod.js";

const TYPE_NAMES = { short_text: "Text", long_text: "Text", list: "List", paired_list: "Paired List" };

export default {
  name: "ReferenceField",
  props: {
    field: { type: Object, required: true },
  },
  emits: ["rename", "change-type", "remove"],
  setup(props) {
    const store = inject("store");

    function itemsFor(side) {
      return (props.field.items || [])
        .filter((it) => it.side === side)
        .slice()
        .sort((a, b) => a.sort_order - b.sort_order);
    }

    async function onTextBlur(e) {
      const value = e.target.value;
      if (value === (props.field.text_content || "")) return;
      const result = await store.saveReferenceFieldContent(props.field, value);
      if (!result.ok) {
        alert("Could not save: " + result.error);
        e.target.value = props.field.text_content || "";
      }
    }
    async function onItemBlur(e, item) {
      const value = e.target.value;
      if (value === item.content) return;
      const result = await store.saveReferenceItemContent(props.field, item, value);
      if (!result.ok) {
        alert("Could not save item: " + result.error);
        e.target.value = item.content;
      }
    }
    async function addItem(side, inputEl) {
      const value = inputEl.value.trim();
      if (!value) return;
      await store.addReferenceItem(props.field, side, value);
      inputEl.value = "";
    }

    return {
      store, TYPE_NAMES, itemsFor,
      onTextBlur, onItemBlur, addItem,
    };
  },
  template: `
    <div class="reference-field">
      <div class="reference-field-label-row">
        <div>
          <span class="reference-field-label">{{ field.label }}</span>
          <span class="reference-field-type-tag">{{ TYPE_NAMES[field.field_type] || field.field_type }}</span>
        </div>
        <div class="reference-field-controls" v-if="store.affordances.value.canEditStructure">
          <button class="icon-btn" @click="$emit('rename', field)">✎</button>
          <button class="icon-btn" @click="$emit('change-type', field)">Type</button>
          <button class="icon-btn danger" @click="$emit('remove', field)">×</button>
        </div>
      </div>

      <div v-if="field.field_type === 'short_text'">
        <input type="text" :value="field.text_content || ''" :disabled="!store.affordances.value.canEditCards" @blur="onTextBlur" />
      </div>
      <div v-else-if="field.field_type === 'long_text'">
        <textarea :value="field.text_content || ''" :disabled="!store.affordances.value.canEditCards" @blur="onTextBlur"></textarea>
      </div>

      <div v-else-if="field.field_type === 'list'">
        <div class="reference-list-items">
          <div class="reference-list-item" v-for="(item, idx) in itemsFor('single')" :key="item.id">
            <input type="text" :value="item.content" :disabled="!store.affordances.value.canEditCards" @blur="onItemBlur($event, item)" />
            <template v-if="store.affordances.value.canEditCards">
              <button v-if="idx > 0" class="icon-btn" @click="store.moveReferenceItem(field, 'single', itemsFor('single'), idx, -1)">↑</button>
              <button v-if="idx < itemsFor('single').length - 1" class="icon-btn" @click="store.moveReferenceItem(field, 'single', itemsFor('single'), idx, 1)">↓</button>
              <button class="icon-btn danger" @click="store.removeReferenceItem(field, item)">×</button>
            </template>
          </div>
        </div>
        <div class="reference-add-item-row" v-if="store.affordances.value.canEditCards">
          <input type="text" placeholder="Add item…" ref="singleAddInput" @keydown.enter="addItem('single', $refs.singleAddInput)" />
          <button class="small" @click="addItem('single', $refs.singleAddInput)">Add</button>
        </div>
      </div>

      <div v-else-if="field.field_type === 'paired_list'" class="reference-paired">
        <div>
          <div class="reference-paired-sublabel">{{ field.sublabel_a || "A" }}</div>
          <div class="reference-list-items">
            <div class="reference-list-item" v-for="(item, idx) in itemsFor('a')" :key="item.id">
              <input type="text" :value="item.content" :disabled="!store.affordances.value.canEditCards" @blur="onItemBlur($event, item)" />
              <template v-if="store.affordances.value.canEditCards">
                <button v-if="idx > 0" class="icon-btn" @click="store.moveReferenceItem(field, 'a', itemsFor('a'), idx, -1)">↑</button>
                <button v-if="idx < itemsFor('a').length - 1" class="icon-btn" @click="store.moveReferenceItem(field, 'a', itemsFor('a'), idx, 1)">↓</button>
                <button class="icon-btn danger" @click="store.removeReferenceItem(field, item)">×</button>
              </template>
            </div>
          </div>
          <div class="reference-add-item-row" v-if="store.affordances.value.canEditCards">
            <input type="text" placeholder="Add item…" ref="aAddInput" @keydown.enter="addItem('a', $refs.aAddInput)" />
            <button class="small" @click="addItem('a', $refs.aAddInput)">Add</button>
          </div>
        </div>
        <div>
          <div class="reference-paired-sublabel">{{ field.sublabel_b || "B" }}</div>
          <div class="reference-list-items">
            <div class="reference-list-item" v-for="(item, idx) in itemsFor('b')" :key="item.id">
              <input type="text" :value="item.content" :disabled="!store.affordances.value.canEditCards" @blur="onItemBlur($event, item)" />
              <template v-if="store.affordances.value.canEditCards">
                <button v-if="idx > 0" class="icon-btn" @click="store.moveReferenceItem(field, 'b', itemsFor('b'), idx, -1)">↑</button>
                <button v-if="idx < itemsFor('b').length - 1" class="icon-btn" @click="store.moveReferenceItem(field, 'b', itemsFor('b'), idx, 1)">↓</button>
                <button class="icon-btn danger" @click="store.removeReferenceItem(field, item)">×</button>
              </template>
            </div>
          </div>
          <div class="reference-add-item-row" v-if="store.affordances.value.canEditCards">
            <input type="text" placeholder="Add item…" ref="bAddInput" @keydown.enter="addItem('b', $refs.bAddInput)" />
            <button class="small" @click="addItem('b', $refs.bAddInput)">Add</button>
          </div>
        </div>
      </div>
    </div>
  `,
};

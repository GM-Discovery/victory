import { inject } from "/lib/vue.esm-browser.prod.js";

export default {
  name: "Toolbar",
  emits: ["share"],
  setup(_, { emit }) {
    const store = inject("store");
    async function addColumn() {
      const title = prompt("New column title", "New Column");
      if (title === null) return;
      await store.addColumn(title);
    }
    async function addBand() {
      await store.addBand();
    }
    // Kernel 94 close-out: a bare `window.print()` call expression in the
    // template mis-resolved `window` to undefined (Vue's compiler handles
    // a call-shaped global reference differently from an assignment like
    // the Export JSON button's `window.location.href = ...`, which works
    // fine inline) -- real methods sidestep that entirely.
    function exportPdf() { window.print(); }
    function exportJson() { window.location.href = store.apiBase + "/export"; }
    return { store, addColumn, addBand, exportPdf, exportJson, emit };
  },
  template: `
    <div class="page-header">
      <div>
        <h1 id="board-title">{{ store.state.snapshot.board.title }}</h1>
        <div class="status-line">{{ (store.state.snapshot.board.description || "") + (store.state.snapshot.board.archived_at ? " · Archived" : "") }}</div>
      </div>
      <div class="header-actions">
        <span class="role-badge">{{ store.state.snapshot.viewer_tier }}</span>
        <a href="/venues/storyboards/">← Boards</a>
        <button v-if="store.affordances.value.canExport" @click="exportPdf" title="Opens your browser's print dialog -- choose Save as PDF">Export PDF</button>
        <button v-if="store.affordances.value.canExport" @click="exportJson">Export JSON</button>
        <button v-if="store.affordances.value.canManageSharing" @click="$emit('share')">Sharing</button>
        <button v-if="store.affordances.value.canArchive" @click="store.toggleArchive()">{{ store.state.snapshot.board.archived_at ? "Unarchive" : "Archive" }}</button>
      </div>
    </div>

    <div class="toolbar" v-if="store.affordances.value.canEditStructure">
      <button class="small" @click="addColumn">+ Column</button>
      <button class="small" @click="addBand">+ Band</button>
      <span class="status-line">{{ store.state.snapshot.columns.length >= 200 ? "200-column limit reached" : store.state.snapshot.columns.length + " / 200 columns" }}</span>
    </div>
  `,
};

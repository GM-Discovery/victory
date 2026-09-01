// Kernel 94 spec 15 fix: the pre-Vue Presence Tray only exposed group
// leader/turn assignment via `contextmenu` (right-click) on a plain <div>
// chip -- no keyboard path existed to a control that can matter for who
// gets to act next. Chips are now real <button>s (focusable, Enter/Space
// activate by default) and both click and right-click open the same menu,
// so the mouse shortcut survives without being the only path in.
import { inject, ref, nextTick, onMounted, onUnmounted } from "/lib/vue.esm-browser.prod.js";

export default {
  name: "PresenceTray",
  setup() {
    const store = inject("store");
    const menu = ref(null); // { entry, items: [[label, handler]] }
    const menuEl = ref(null);
    const menuPos = ref({ left: "0px", top: "0px" });

    function iAmGroupLeader() {
      const coord = store.state.snapshot.coordination;
      return !!(coord && coord.active && coord.group_leader_user_id && coord.group_leader_user_id === store.state.snapshot.viewer_user_id);
    }
    function iAmCurrentTurnHolder() {
      const coord = store.state.snapshot.coordination;
      return !!(coord && coord.active && coord.current_turn_user_id && coord.current_turn_user_id === store.state.snapshot.viewer_user_id);
    }

    async function openMenu(event, entry) {
      event.preventDefault();
      closeMenu();
      const coord = store.state.snapshot.coordination || {};
      const isAlreadyLeader = coord.active && coord.group_leader_user_id === entry.user_id;
      const isAlreadyTurn = coord.active && coord.current_turn_user_id === entry.user_id;
      const canAssignLeader = coord.active && (store.affordances.value.canEditStructure || iAmGroupLeader());
      const canAssignTurn = coord.active && (store.affordances.value.canEditStructure || iAmGroupLeader() || iAmCurrentTurnHolder());

      const items = [];
      if (canAssignLeader && !isAlreadyLeader) items.push(["Make Group Leader", () => store.postCoordinationAction("leader", entry.user_id)]);
      if (canAssignTurn && !isAlreadyTurn) items.push(["Give Turn", () => store.postCoordinationAction("turn", entry.user_id)]);
      if (items.length === 0) return; // only show actions the actor is authorized to perform

      menu.value = { entry, items };
      const left = Math.min(event.clientX, window.innerWidth - 210);
      const top = Math.min(event.clientY, window.innerHeight - 120);
      menuPos.value = { left: left + "px", top: top + "px" };
      await nextTick();
      const firstBtn = menuEl.value && menuEl.value.querySelector("button");
      if (firstBtn) firstBtn.focus();
    }
    function closeMenu() {
      menu.value = null;
    }
    function runItem(handler) {
      closeMenu();
      handler();
    }

    function onDocClick(event) {
      if (menu.value && menuEl.value && !menuEl.value.contains(event.target)) closeMenu();
    }
    function onDocKeydown(event) {
      if (event.key === "Escape") closeMenu();
    }
    onMounted(() => {
      document.addEventListener("click", onDocClick);
      document.addEventListener("keydown", onDocKeydown);
    });
    onUnmounted(() => {
      document.removeEventListener("click", onDocClick);
      document.removeEventListener("keydown", onDocKeydown);
    });

    return { store, menu, menuEl, menuPos, openMenu, closeMenu, runItem };
  },
  template: `
    <div class="presence-tray" v-if="(store.state.snapshot.presence || []).length">
      <button
        type="button"
        class="presence-chip"
        v-for="entry in store.state.snapshot.presence"
        :key="entry.user_id"
        :title="entry.handle ? ('@' + entry.handle) : ''"
        @click="openMenu($event, entry)"
        @contextmenu="openMenu($event, entry)"
      >
        <span class="presence-chip-dot"></span>
        <span class="presence-chip-name">{{ entry.display_name || entry.handle || "Someone" }}</span>
        <span v-if="store.state.snapshot.coordination && store.state.snapshot.coordination.active && store.state.snapshot.coordination.group_leader_user_id === entry.user_id" class="presence-badge presence-badge-leader">Leader</span>
        <span v-if="store.state.snapshot.coordination && store.state.snapshot.coordination.active && store.state.snapshot.coordination.current_turn_user_id === entry.user_id" class="presence-badge presence-badge-turn">Turn</span>
      </button>
    </div>

    <Transition name="popup">
    <div v-if="menu" class="presence-context-menu" ref="menuEl" :style="menuPos">
      <span class="presence-menu-name">{{ menu.entry.display_name || menu.entry.handle || "Someone" }}</span>
      <button v-for="([label, handler]) in menu.items" :key="label" type="button" @click="runItem(handler)">{{ label }}</button>
    </div>
    </Transition>
  `,
};

import { inject, ref, watch } from "/lib/vue.esm-browser.prod.js";

export default {
  name: "ShareModal",
  props: {
    open: { type: Boolean, default: false },
  },
  emits: ["close"],
  setup(props) {
    const store = inject("store");
    const grants = ref([]);
    const role = ref("audience");
    const handle = ref("");
    const message = ref("");
    const messageOk = ref(false);

    async function refresh() {
      grants.value = await store.fetchGrants();
    }
    watch(() => props.open, (open) => {
      if (open) { message.value = ""; refresh(); }
    });

    async function removeGrant(g) {
      const result = await store.removeGrant(g.id);
      if (!result.ok) { alert("Remove failed: " + result.error); return; }
      await refresh();
    }

    function openPeoplePicker() {
      window.VictoryPeoplePicker.open({
        title: "Share this Storyboard",
        hint: "Select someone from My People or Third Place. Their account handle is never shown or required.",
        onSelect: async (person) => {
          const result = await store.addGrant({ profile_id: person.profile_id, granted_role: role.value });
          if (!result.ok) {
            messageOk.value = false;
            message.value = "Share failed: " + result.error;
            return;
          }
          messageOk.value = true;
          message.value = "Shared with " + person.display_name + ".";
          await refresh();
        },
      });
    }

    async function addByHandle() {
      const value = handle.value.trim();
      if (!value) {
        messageOk.value = false;
        message.value = "Enter a handle first.";
        return;
      }
      const result = await store.addGrant({ user_handle: value, granted_role: role.value });
      if (!result.ok) {
        messageOk.value = false;
        message.value = "Share failed: " + result.error;
        return;
      }
      messageOk.value = true;
      message.value = "Shared with " + value + ".";
      handle.value = "";
      await refresh();
    }

    return { grants, role, handle, message, messageOk, removeGrant, openPeoplePicker, addByHandle };
  },
  template: `
    <Transition name="modal">
    <div class="modal-backdrop" v-if="open">
      <div class="modal">
        <h2>Sharing</h2>
        <div>
          <div class="grant-row" v-for="g in grants" :key="g.id">
            <span>{{ (g.user_handle || g.user_id) + " — " + g.granted_role }}</span>
            <button class="small danger" @click="removeGrant(g)">Remove</button>
          </div>
        </div>
        <div class="field">
          <label>Role for the person you pick or paste below</label>
          <select v-model="role">
            <option value="audience">Audience</option>
            <option value="cast">Cast</option>
            <option value="crew">Crew</option>
            <option value="director">Director</option>
            <option value="producer">Producer</option>
          </select>
        </div>
        <div class="field">
          <label>People you know</label>
          <button type="button" @click="openPeoplePicker">Choose a Person…</button>
        </div>
        <div class="field">
          <label>Handle (fallback — exact Victory handle, ID-only)</label>
          <input v-model="handle" placeholder="Only if you already know their exact handle" />
        </div>
        <div :class="message ? (messageOk ? 'form-ok' : 'form-error') : ''">{{ message }}</div>
        <div class="modal-actions">
          <button @click="addByHandle">Share by Handle</button>
          <button @click="$emit('close')">Close</button>
        </div>
      </div>
    </div>
    </Transition>
  `,
};

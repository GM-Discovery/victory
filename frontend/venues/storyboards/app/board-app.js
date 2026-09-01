import { createApp } from "/lib/vue.esm-browser.prod.js";
import AppRoot from "./AppRoot.js";

const params = new URLSearchParams(window.location.search);
const boardId = params.get("id");

if (!boardId) {
  document.getElementById("board-status").textContent = "No board id given.";
} else {
  createApp(AppRoot, { boardId }).mount("#board-app");
}

import { render } from "@solidjs/web";
import { App } from "./components/App";

const root = document.getElementById("app")!;
const mascot = root.dataset.mascot!;
root.replaceChildren();
render(() => <App mascot={mascot} />, root);

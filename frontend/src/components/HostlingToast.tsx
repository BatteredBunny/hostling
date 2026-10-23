import "./HostlingToast.css";
import { Show } from "solid-js";
import { Portal } from "solid-js/web";
import { toast } from "../toast";

export function HostlingToast(props: { inline?: boolean } = {}) {
  const notification = () => (
    <Show when={toast()} keyed>
      {(item) => (
        <div
          class="hostling-toast"
          classList={{ error: item.kind === "error" }}
          role={item.kind === "error" ? "alert" : "status"}
        >
          {item.message}
        </div>
      )}
    </Show>
  );
  return (
    <Show when={props.inline} fallback={<Portal>{notification()}</Portal>}>
      {notification()}
    </Show>
  );
}

import "./HostlingToast.css";
import { Show } from "solid-js";
import { Portal } from "solid-js/web";
import { toast } from "../toast";
import { cx } from "../utils";

export function HostlingToast(props: { inline?: boolean } = {}) {
  const notification = () => (
    <Show when={toast()} keyed>
      {(item) => (
        <div
          class={cx("hostling-toast", { error: item.kind === "error" })}
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

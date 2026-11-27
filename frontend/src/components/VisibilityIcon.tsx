import { Show } from "solid-js";
import LockIcon from "@lucide/solid/icons/lock";
import LockOpenIcon from "@lucide/solid/icons/lock-open";

export function VisibilityIcon(props: { public: boolean }) {
  return (
    <Show
      when={props.public}
      fallback={<LockIcon class="lucide-icon" aria-hidden="true" />}
    >
      <LockOpenIcon class="lucide-icon" aria-hidden="true" />
    </Show>
  );
}

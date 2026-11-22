import RotateCwIcon from "lucide-solid/icons/rotate-cw";
import TriangleAlertIcon from "lucide-solid/icons/triangle-exclamation-point";
import { cx } from "../utils";
import styles from "./LoadError.module.css";

export function LoadError(props: {
  message: string;
  onRetry: () => void;
  busy?: boolean;
}) {
  return (
    <div class={styles.loadError} role="alert">
      <TriangleAlertIcon
        class={cx("lucide-icon", styles.icon)}
        aria-hidden="true"
      />
      <p>{props.message}</p>
      <button
        type="button"
        class={styles.retry}
        disabled={props.busy}
        onClick={() => props.onRetry()}
      >
        <RotateCwIcon
          class={cx("lucide-icon", { spinning: props.busy })}
          aria-hidden="true"
        />
        Retry
      </button>
    </div>
  );
}

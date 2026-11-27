import type { JSX } from "@solidjs/web";
import TriangleAlertIcon from "@lucide/solid/icons/triangle-exclamation-point";
import styles from "./WarningCallout.module.css";

export function WarningCallout(props: {
  children: JSX.Element;
  class?: string;
}) {
  return (
    <div class={[styles.callout, props.class]} role="note">
      <TriangleAlertIcon
        class={`lucide-icon ${styles.icon}`}
        aria-hidden="true"
      />
      <div class={styles.content}>{props.children}</div>
    </div>
  );
}

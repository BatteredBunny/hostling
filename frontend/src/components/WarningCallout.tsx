import type { JSX } from "solid-js";
import TriangleAlertIcon from "lucide-solid/icons/triangle-exclamation-point";
import styles from "./WarningCallout.module.css";
import { cx } from "../utils";

export function WarningCallout(props: {
  children: JSX.Element;
  class?: string;
}) {
  return (
    <div class={cx(styles.callout, props.class)} role="note">
      <TriangleAlertIcon
        class={cx("lucide-icon", styles.icon)}
        aria-hidden="true"
      />
      <div class={styles.content}>{props.children}</div>
    </div>
  );
}

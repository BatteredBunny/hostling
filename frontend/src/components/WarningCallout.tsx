import type { JSX } from "solid-js";
import styles from "./WarningCallout.module.css";

export function WarningCallout(props: {
  children: JSX.Element;
  class?: string;
}) {
  return (
    <div class={`${styles.callout} ${props.class ?? ""}`}>{props.children}</div>
  );
}

import { Show, createUniqueId } from "solid-js";
import type { JSX } from "solid-js";
import styles from "./SectionPanel.module.css";

export function SectionPanel(props: {
  title?: JSX.Element;
  children: JSX.Element;
  id?: string;
  class?: string;
  bodyClass?: string;
  headerActions?: JSX.Element;
}) {
  const titleId = `section-${createUniqueId()}`;
  return (
    <section
      id={props.id}
      class={`${styles.panel} ${props.class ?? ""}`}
      aria-labelledby={props.title ? titleId : undefined}
    >
      <Show when={props.title || props.headerActions}>
        <header class={styles.header}>
          <h2 id={titleId}>{props.title}</h2>
          <Show when={props.headerActions}>{props.headerActions}</Show>
        </header>
      </Show>
      <div class={`${styles.body} ${props.bodyClass ?? ""}`}>
        {props.children}
      </div>
    </section>
  );
}

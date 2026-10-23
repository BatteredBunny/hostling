import { createEffect, createUniqueId, onCleanup } from "solid-js";
import type { JSX } from "solid-js";
import styles from "./NativePopover.module.css";

// Native light dismissal and top-layer rendering, with CSS anchoring where
// available and one shared positioning fallback for older browsers.
export function NativePopover(props: {
  id: string;
  open: boolean;
  anchor: () => HTMLButtonElement | undefined;
  onOpenChange: (open: boolean) => void;
  onOpen?: () => void;
  ref?: (element: HTMLDivElement) => void;
  class: string;
  role: "listbox" | "dialog";
  label?: string;
  activeDescendant?: string;
  onKeyDown?: JSX.EventHandler<HTMLDivElement, KeyboardEvent>;
  children: JSX.Element;
}) {
  let element!: HTMLDivElement;
  const anchorName = `--popover-${createUniqueId()}`;
  createEffect(() => {
    const button = props.anchor();
    if (!props.open || !button) {
      if (element.matches(":popover-open")) element.hidePopover();
      return;
    }
    const previousAnchor = button.style.getPropertyValue("anchor-name");
    button.style.setProperty("anchor-name", anchorName);
    const positionedByCSS = CSS.supports("position-anchor", anchorName);
    const sizedByCSS =
      positionedByCSS && CSS.supports("min-width", "anchor-size(width)");
    const position = () => {
      if (positionedByCSS && sizedByCSS) return;
      const rect = button.getBoundingClientRect();
      if (!sizedByCSS) element.style.minWidth = `${rect.width}px`;
      if (!positionedByCSS) {
        element.style.top = `${rect.bottom + 6}px`;
        element.style.right = `${Math.max(8, window.innerWidth - rect.right)}px`;
      }
    };
    position();
    element.showPopover({ source: button });
    props.onOpen?.();
    if (!positionedByCSS || !sizedByCSS) {
      window.addEventListener("scroll", position, true);
      window.addEventListener("resize", position);
    }
    onCleanup(() => {
      button.style.setProperty("anchor-name", previousAnchor);
      window.removeEventListener("scroll", position, true);
      window.removeEventListener("resize", position);
    });
  });
  onCleanup(() => {
    if (element.matches(":popover-open")) element.hidePopover();
  });
  return (
    <div
      id={props.id}
      ref={(el) => {
        element = el;
        el.style.setProperty("position-anchor", anchorName);
        props.ref?.(el);
      }}
      popover="auto"
      class={`${styles.popover} ${props.class}`}
      role={props.role}
      aria-label={props.label}
      tabindex="-1"
      aria-activedescendant={props.activeDescendant}
      onToggle={(event) => props.onOpenChange(event.newState === "open")}
      onKeyDown={(event) => props.onKeyDown?.(event)}
    >
      {props.children}
    </div>
  );
}

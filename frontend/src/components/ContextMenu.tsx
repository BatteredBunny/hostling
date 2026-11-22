import { For, Show, createSignal, onCleanup, onMount } from "solid-js";
import { Portal } from "solid-js/web";
import type LinkIcon from "lucide-solid/icons/link";
import "./ContextMenu.css";
import { cx } from "../utils";

export interface ContextMenuItem {
  label: string;
  icon: typeof LinkIcon;
  danger?: boolean;
  disabled?: boolean;
  separatorBefore?: boolean;
  // Links render as anchors; everything else runs `action`.
  href?: string;
  download?: string;
  action?: () => void;
}

const ITEM_SELECTOR = ".context-menu-item:not(:disabled)";

// A positioned action menu with keyboard navigation. The owner keeps the open
// state; `onClose(restoreFocus)` asks it to unmount the menu, returning focus
// to the trigger unless the menu was dismissed by pointer, scroll or resize.
export function ContextMenu(props: {
  x: number;
  y: number;
  label: string;
  items: ContextMenuItem[];
  onClose: (restoreFocus: boolean) => void;
}) {
  let menuEl!: HTMLDivElement;
  // Placed in onMount once the menu's size is known, before the first paint.
  const [position, setPosition] = createSignal({ left: 0, top: 0 });

  const menuItems = () =>
    Array.from(menuEl.querySelectorAll<HTMLElement>(ITEM_SELECTOR));

  const moveFocus = (delta: number) => {
    const items = menuItems();
    const index = items.indexOf(document.activeElement as HTMLElement);
    const next = (index + delta + items.length) % items.length;
    items[next]?.focus();
  };

  onMount(() => {
    // Keep the menu on screen, flipping it to the other side of the
    // cursor when it would overflow.
    const rect = menuEl.getBoundingClientRect();
    const margin = 8;
    let left = props.x;
    let top = props.y;
    if (left + rect.width > window.innerWidth - margin)
      left = Math.max(margin, left - rect.width);
    if (top + rect.height > window.innerHeight - margin)
      top = Math.max(margin, top - rect.height);
    setPosition({ left, top });
    menuItems()[0]?.focus();

    // Capture phase so the page's own key handling (a modal's Escape and
    // arrows, for example) doesn't also react while the menu is open.
    const onKey = (e: KeyboardEvent) => {
      e.stopPropagation();
      if (e.key === "Escape") {
        e.preventDefault();
        props.onClose(true);
      } else if (e.key === "ArrowDown") {
        e.preventDefault();
        moveFocus(1);
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        moveFocus(-1);
      } else if (e.key === "Tab") {
        props.onClose(true);
      }
    };
    const onDismiss = () => props.onClose(false);

    window.addEventListener("keydown", onKey, true);
    window.addEventListener("scroll", onDismiss, true);
    window.addEventListener("resize", onDismiss);
    onCleanup(() => {
      window.removeEventListener("keydown", onKey, true);
      window.removeEventListener("scroll", onDismiss, true);
      window.removeEventListener("resize", onDismiss);
    });
  });

  const runItem = (item: ContextMenuItem) => {
    // Return focus before the action runs, so dialogs it opens can take
    // focus and later hand it back to the trigger.
    props.onClose(true);
    item.action?.();
  };

  return (
    // Inside an open modal dialog, the menu must be too to appear above it.
    <Portal
      mount={
        document.querySelector<HTMLDialogElement>("dialog:modal") ?? undefined
      }
    >
      {/* Swallows the dismissing click so it doesn't also hit what's underneath. */}
      <div
        class="context-menu-overlay"
        onClick={() => props.onClose(false)}
        onContextMenu={(e) => {
          e.preventDefault();
          props.onClose(false);
        }}
      />
      <div
        ref={menuEl}
        class="context-menu"
        role="menu"
        aria-label={props.label}
        style={{ left: `${position().left}px`, top: `${position().top}px` }}
        onContextMenu={(e) => e.preventDefault()}
      >
        <For each={props.items}>
          {(item) => (
            <>
              <Show when={item.separatorBefore}>
                <div class="context-menu-separator" role="separator" />
              </Show>
              <Show
                when={item.href}
                fallback={
                  <button
                    type="button"
                    role="menuitem"
                    class={cx("context-menu-item", { danger: item.danger })}
                    disabled={item.disabled}
                    onClick={() => runItem(item)}
                  >
                    <item.icon class="lucide-icon" aria-hidden="true" />
                    <span>{item.label}</span>
                  </button>
                }
              >
                <a
                  role="menuitem"
                  class="context-menu-item"
                  href={item.href}
                  target={item.download ? undefined : "_blank"}
                  rel="noopener noreferrer"
                  download={item.download}
                  onClick={() => props.onClose(true)}
                >
                  <item.icon class="lucide-icon" aria-hidden="true" />
                  <span>{item.label}</span>
                </a>
              </Show>
            </>
          )}
        </For>
      </div>
    </Portal>
  );
}

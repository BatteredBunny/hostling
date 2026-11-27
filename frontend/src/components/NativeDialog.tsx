import { createEffect, onCleanup } from "solid-js";
import { Portal, type JSX } from "@solidjs/web";
import styles from "./NativeDialog.module.css";
import { ariaBool } from "../utils";

export function NativeDialog(props: {
  children: JSX.Element;
  id?: string;
  class?: string;
  open?: boolean;
  busy?: boolean;
  labelledBy?: string;
  label?: string;
  describedBy?: string;
  onDismiss?: () => void;
  onKeyDown?: (event: KeyboardEvent) => void;
  ref?: (element: HTMLDialogElement) => void;
}) {
  let dialog!: HTMLDialogElement;
  const previousFocus =
    document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null;
  let backdropPressed = false;
  createEffect(
    () => props.open,
    (open) => {
      if (open === undefined) return;
      if (open && !dialog.open) dialog.showModal();
      else if (!open && dialog.open) dialog.close();
    },
  );
  onCleanup(() => {
    dialog.close();
    if (previousFocus?.isConnected) previousFocus.focus();
  });
  const outside = (event: MouseEvent) => {
    if (event.target !== dialog) return false;
    const bounds = dialog.getBoundingClientRect();
    return (
      event.clientX < bounds.left ||
      event.clientX > bounds.right ||
      event.clientY < bounds.top ||
      event.clientY > bounds.bottom
    );
  };
  const dismiss = () => {
    if (props.busy) return;
    props.onDismiss?.();
    if (dialog.open) dialog.close();
  };
  return (
    <Portal>
      <dialog
        ref={(element) => {
          dialog = element;
          props.ref?.(element);
        }}
        id={props.id}
        class={[styles.dialog, props.class]}
        aria-labelledby={props.labelledBy}
        aria-label={props.label}
        aria-describedby={props.describedBy}
        aria-busy={ariaBool(props.busy)}
        onCancel={(event) => {
          event.preventDefault();
          dismiss();
        }}
        onKeyDown={(event) => {
          props.onKeyDown?.(event);
          event.stopPropagation();
        }}
        onMouseDown={(event) => {
          backdropPressed = outside(event);
        }}
        onClick={(event) => {
          if (backdropPressed && outside(event)) dismiss();
          backdropPressed = false;
        }}
      >
        {props.children}
      </dialog>
    </Portal>
  );
}

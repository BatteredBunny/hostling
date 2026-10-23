import { createUniqueId, Show } from "solid-js";
import { NativeDialog } from "./NativeDialog";
import styles from "./NativeConfirmation.module.css";

export function NativeConfirmation(props: {
  id?: string;
  open?: boolean;
  title: string;
  message: string;
  busy?: boolean;
  error?: string;
  danger?: boolean;
  confirmLabel?: string;
  onConfirm: () => void;
  onDismiss?: () => void;
  ref?: (element: HTMLDialogElement) => void;
}) {
  const uid = createUniqueId();
  let dialog!: HTMLDialogElement;
  const dismiss = () => {
    props.onDismiss?.();
    dialog.close();
  };
  return (
    <NativeDialog
      id={props.id}
      open={props.open}
      busy={props.busy}
      ref={(element) => {
        dialog = element;
        props.ref?.(element);
      }}
      labelledBy={`${uid}-title`}
      describedBy={`${uid}-message`}
      onDismiss={props.onDismiss}
      class={styles.confirmation}
    >
      <h3 id={`${uid}-title`}>{props.title}</h3>
      <p id={`${uid}-message`}>{props.message}</p>
      <Show when={props.error}>
        <p role="alert">{props.error}</p>
      </Show>
      <div class={styles.actions}>
        <button
          type="button"
          class={styles.cancel}
          disabled={props.busy}
          autofocus
          onClick={dismiss}
        >
          Cancel
        </button>
        <button
          type="button"
          class={props.danger === false ? "create-button" : "delete-button"}
          disabled={props.busy}
          onClick={() => props.onConfirm()}
        >
          {props.confirmLabel ?? "Confirm deletion"}
        </button>
      </div>
    </NativeDialog>
  );
}

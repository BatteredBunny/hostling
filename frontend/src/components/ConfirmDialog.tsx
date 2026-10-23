import { onCleanup, onMount } from 'solid-js';
import { Portal } from 'solid-js/web';
import './ConfirmDialog.css';
import { setConfirmRequest, type ConfirmRequest } from '../store';

export function ConfirmDialog(props: { request: ConfirmRequest }) {
  let cancelButton: HTMLButtonElement | undefined;
  const previouslyFocused = document.activeElement as HTMLElement | null;

  const finish = (confirmed: boolean) => {
    setConfirmRequest(null);
    props.request.resolve(confirmed);
  };

  onMount(() => {
    // Default to the harmless choice so a stray Enter doesn't delete anything.
    cancelButton?.focus();

    // Capture phase so the file modal's own key handling doesn't also react.
    const onKey = (e: KeyboardEvent) => {
      e.stopPropagation();
      if (e.key === 'Escape') {
        e.preventDefault();
        finish(false);
      }
    };
    window.addEventListener('keydown', onKey, true);
    onCleanup(() => {
      window.removeEventListener('keydown', onKey, true);
      if (previouslyFocused?.isConnected) previouslyFocused.focus();
    });
  });

  return (
    <Portal>
      <div
        class="confirm-dialog-backdrop"
        onClick={(e) => {
          if (e.target === e.currentTarget) finish(false);
        }}
      >
        <div
          class="confirm-dialog"
          role="alertdialog"
          aria-modal="true"
          aria-labelledby="confirm-dialog-title"
          aria-describedby="confirm-dialog-message"
        >
          <h3 id="confirm-dialog-title">{props.request.title}</h3>
          <p id="confirm-dialog-message">{props.request.message}</p>
          <div class="confirm-dialog-actions">
            <button type="button" ref={cancelButton} class="confirm-dialog-cancel" onClick={() => finish(false)}>
              Cancel
            </button>
            <button
              type="button"
              class={props.request.danger ? 'delete-button' : 'create-button'}
              onClick={() => finish(true)}
            >
              {props.request.confirmLabel}
            </button>
          </div>
        </div>
      </div>
    </Portal>
  );
}

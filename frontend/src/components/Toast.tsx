import { Show } from 'solid-js';
import { Portal } from 'solid-js/web';
import './Toast.css';
import { toastMessage } from '../store';

export function Toast() {
  return (
    <Show when={toastMessage()}>
      {(message) => (
        <Portal>
          <div class="toast" role="status">{message()}</div>
        </Portal>
      )}
    </Show>
  );
}

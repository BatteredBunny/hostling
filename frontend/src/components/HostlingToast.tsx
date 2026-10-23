import { Show } from 'solid-js';
import { Portal } from 'solid-js/web';
import { toast } from '../toast';
import styles from './HostlingToast.css?inline';

export function HostlingToast() {
  return (
    <Portal>
      {/* Keep the shared component's styles with it across frontend entry points. */}
      <style>{styles}</style>
      <Show when={toast()} keyed>
        {(notification) => (
          <div class="hostling-toast" role="status">{notification.message}</div>
        )}
      </Show>
    </Portal>
  );
}

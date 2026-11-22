import {
  Show,
  createSignal,
  createUniqueId,
  onCleanup,
  onMount,
} from "solid-js";
import CheckIcon from "lucide-solid/icons/check";
import CopyIcon from "lucide-solid/icons/copy";
import { NativeDialog } from "./NativeDialog";
import { HostlingToast } from "./HostlingToast";
import { showToast } from "../toast";
import { copyText } from "../utils";
import "./SecretDialog.css";

// Shows a freshly created secret (an upload token or invite code) once,
// with a copy button that takes focus.
export function SecretDialog(props: {
  title: string;
  notice: string;
  value: string;
  // What is being copied, e.g. "token" for "Copy token" and "Token copied.".
  noun: string;
  onDismiss: () => void;
}) {
  const uid = createUniqueId();
  const [copied, setCopied] = createSignal(false);
  const [copyError, setCopyError] = createSignal("");
  let dialog: HTMLDialogElement | undefined;
  let copyButton: HTMLButtonElement | undefined;
  let copyTimeout: ReturnType<typeof setTimeout> | undefined;
  onMount(() =>
    queueMicrotask(() => copyButton?.isConnected && copyButton.focus()),
  );
  onCleanup(() => clearTimeout(copyTimeout));

  const noun = () => props.noun[0].toUpperCase() + props.noun.slice(1);

  const copy = async () => {
    setCopyError("");
    try {
      await copyText(props.value, dialog ?? document.body);
      setCopied(true);
      showToast(`${noun()} copied.`);
      clearTimeout(copyTimeout);
      copyTimeout = setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopyError(
        `Could not copy the ${props.noun}. You can select and copy it below.`,
      );
    }
  };
  return (
    <NativeDialog
      open
      ref={(element) => {
        dialog = element;
      }}
      class="secret-dialog"
      labelledBy={`${uid}-title`}
      describedBy={`${uid}-notice`}
      onDismiss={props.onDismiss}
    >
      <header class="secret-dialog-titlebar">
        <h2 id={`${uid}-title`}>{props.title}</h2>
      </header>
      <div class="secret-dialog-contents">
        <p id={`${uid}-notice`}>{props.notice}</p>
        <pre class="secret-dialog-code">
          <code>{props.value}</code>
        </pre>
        <Show when={copyError()}>
          <p class="secret-dialog-error" role="alert">
            {copyError()}
          </p>
        </Show>
      </div>
      <footer class="secret-dialog-footer">
        <button
          ref={copyButton}
          type="button"
          class="secret-dialog-copy"
          onClick={copy}
        >
          <Show
            when={copied()}
            fallback={<CopyIcon class="lucide-icon" aria-hidden="true" />}
          >
            <CheckIcon class="lucide-icon" aria-hidden="true" />
          </Show>
          {copied() ? "Copied" : `Copy ${props.noun}`}
        </button>
        <button
          type="button"
          class="secret-dialog-dismiss"
          onClick={() => props.onDismiss()}
        >
          Dismiss
        </button>
      </footer>
      <HostlingToast inline />
    </NativeDialog>
  );
}

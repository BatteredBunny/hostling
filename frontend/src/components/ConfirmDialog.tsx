import { Show, onCleanup } from "solid-js";
import {
  confirmRequest,
  setConfirmRequest,
  cancelConfirmation,
  type ConfirmRequest,
} from "../store";
import { NativeConfirmation } from "./NativeConfirmation";

function PendingConfirmation(props: { request: ConfirmRequest }) {
  const request = () => props.request;
  const finish = (confirmed: boolean) => {
    request().resolve(confirmed);
    setConfirmRequest(null);
  };
  onCleanup(() => request().resolve(false));
  return (
    <NativeConfirmation
      open
      title={request().title}
      message={request().message}
      danger={request().danger}
      confirmLabel={request().confirmLabel}
      onConfirm={() => finish(true)}
      onDismiss={() => finish(false)}
    />
  );
}

export function ConfirmDialog() {
  onCleanup(cancelConfirmation);
  return (
    <Show when={confirmRequest()} keyed>
      {(request) => <PendingConfirmation request={request} />}
    </Show>
  );
}

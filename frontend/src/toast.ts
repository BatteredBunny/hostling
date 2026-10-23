import { createSignal } from "solid-js";

interface ToastNotification {
  id: number;
  message: string;
  kind: "success" | "error";
}

export const [toast, setToast] = createSignal<ToastNotification | null>(null);
let nextId = 0;
let timeout: ReturnType<typeof setTimeout> | undefined;

export function dismissToast() {
  clearTimeout(timeout);
  setToast(null);
}

export function showToast(
  message: string,
  kind: ToastNotification["kind"] = "success",
) {
  clearTimeout(timeout);
  setToast({ id: ++nextId, message, kind });
  timeout = setTimeout(dismissToast, 3000);
}

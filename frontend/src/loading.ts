import { createSignal } from "solid-js";

const [pending, setPending] = createSignal(0);

export const isPageLoading = () => pending() > 0;

export function trackPageLoad<T>(load: Promise<T>): Promise<T> {
  setPending((n) => n + 1);
  return load.finally(() => setPending((n) => n - 1));
}

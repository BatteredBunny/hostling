import { createSignal, onCleanup, onMount } from "solid-js";
import { getContext, type FrontendContext } from "./generated";
import { apiData } from "./client";

const [context, setContext] = createSignal<FrontendContext>();
const [error, setError] = createSignal("");
let pending: Promise<FrontendContext> | undefined;
let consumers = 0;
let stopListening: (() => void) | undefined;

export function loadContext(refresh = false): Promise<FrontendContext> {
  if (pending) return pending;
  const current = context();
  if (!refresh && current) return Promise.resolve(current);
  setError("");
  pending = apiData(getContext())
    .then((data) => {
      setContext(data);
      return data;
    })
    .catch((failure) => {
      setError("Could not load server information.");
      throw failure;
    })
    .finally(() => {
      pending = undefined;
    });
  return pending;
}

function listenForChanges() {
  const channel =
    typeof BroadcastChannel === "undefined"
      ? undefined
      : new BroadcastChannel("hostling-context");
  const refresh = () => {
    if (document.visibilityState === "visible")
      void loadContext(true).catch(() => {});
  };
  const onNavigation = () => {
    refresh();
    channel?.postMessage("refresh");
  };
  const onPageShow = (event: PageTransitionEvent) => {
    if (event.persisted) refresh();
  };
  if (channel)
    channel.onmessage = (event) => {
      if (event.data !== "refresh" || document.visibilityState !== "visible")
        return;
      // A request already in flight may have used the old cookie. Revalidate
      // after it settles, coalescing multiple notifications into one request.
      void (pending ?? Promise.resolve())
        .catch(() => {})
        .then(() => loadContext(true))
        .catch(() => {});
    };
  window.addEventListener("focus", refresh);
  window.addEventListener("pageshow", onPageShow);
  document.addEventListener("visibilitychange", refresh);
  document.addEventListener("astro:page-load", onNavigation);
  // A new document follows login, logout and provider-link redirects. Notify
  // other visible tabs once it has read the current cookie-backed state.
  void loadContext(true)
    .then(() => channel?.postMessage("refresh"))
    .catch(() => {});
  return () => {
    channel?.close();
    window.removeEventListener("focus", refresh);
    window.removeEventListener("pageshow", onPageShow);
    document.removeEventListener("visibilitychange", refresh);
    document.removeEventListener("astro:page-load", onNavigation);
  };
}

export function useFrontendContext() {
  onMount(() => {
    if (consumers++ === 0) stopListening = listenForChanges();
    void loadContext().catch(() => {});
    onCleanup(() => {
      if (--consumers === 0) {
        stopListening?.();
        stopListening = undefined;
      }
    });
  });
  return {
    context,
    error,
    retry: () => {
      void loadContext(true).catch(() => {});
    },
  };
}

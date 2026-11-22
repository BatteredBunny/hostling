import { createSignal } from "solid-js";

export const APP_PATHS = [
  "/",
  "/login",
  "/register",
  "/gallery",
  "/settings",
  "/tokens",
  "/admin",
] as const;
export type AppPath = (typeof APP_PATHS)[number];

const isAppPath = (path: string): path is AppPath =>
  (APP_PATHS as readonly string[]).includes(path);

const currentPath = (): AppPath =>
  typeof window !== "undefined" && isAppPath(window.location.pathname)
    ? window.location.pathname
    : "/";

const [path, setPath] = createSignal<AppPath>(currentPath());
const [visit, setVisit] = createSignal(0);
export { path, visit };

export function navigate(to: string, options: { replace?: boolean } = {}) {
  const url = new URL(to, window.location.href);
  if (url.origin !== window.location.origin || !isAppPath(url.pathname)) {
    window.location.assign(url.href);
    return;
  }
  const target = url.pathname + url.search + url.hash;
  if (options.replace) history.replaceState(null, "", target);
  else history.pushState(null, "", target);
  setPath(url.pathname);
  setVisit((n) => n + 1);
  if (!url.hash) window.scrollTo(0, 0);
}

export function startRouter() {
  const onClick = (event: MouseEvent) => {
    if (
      event.defaultPrevented ||
      event.button !== 0 ||
      event.metaKey ||
      event.ctrlKey ||
      event.shiftKey ||
      event.altKey
    )
      return;
    const link = (event.target as Element | null)?.closest("a");
    if (
      !link ||
      (link.target && link.target !== "_self") ||
      link.hasAttribute("download") ||
      link.origin !== window.location.origin ||
      !isAppPath(link.pathname)
    )
      return;
    event.preventDefault();
    navigate(link.href);
  };
  const onPopState = () => setPath(currentPath());

  document.addEventListener("click", onClick);
  window.addEventListener("popstate", onPopState);
  return () => {
    document.removeEventListener("click", onClick);
    window.removeEventListener("popstate", onPopState);
  };
}

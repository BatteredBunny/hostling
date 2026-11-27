import { Show, createEffect, createMemo, onSettled } from "solid-js";
import type { Component } from "solid-js";
import { dynamic } from "@solidjs/web";
import { Navigation } from "./Navigation";
import { LoadingScreen, StartupError } from "./LoadingScreen";
import { HostlingToast } from "./HostlingToast";
import { ConfirmDialog } from "./ConfirmDialog";
import { UploadPage } from "./UploadPage";
import { LoginPage } from "./LoginPage";
import { RegistrationPage } from "./RegistrationPage";
import { FileLibrary } from "./FileLibrary";
import { SettingsPage } from "./SettingsPage";
import { TokensPage } from "./TokensPage";
import { AdminPage } from "./AdminPage";
import { PAGE_TITLES, path, startRouter, visit, type AppPath } from "../router";
import { isPageLoading } from "../loading";
import { useFrontendContext } from "../context";
import { cachedBranding } from "../branding";
import { resetGalleryFromUrl } from "../store";

interface Route {
  // Names the page for the navigation and the page's container id.
  page: string;
  component: Component;
  // The id styles hang off, defaulting to `${page}-page`.
  id?: string;
  // Runs on navigation to the page, before it mounts.
  enter?: () => void;
}

const ROUTES: Record<AppPath, Route> = {
  "/": { page: "home", component: UploadPage },
  "/login": { page: "login", component: LoginPage },
  "/register": {
    page: "register",
    component: RegistrationPage,
  },
  "/gallery": {
    page: "gallery",
    component: FileLibrary,
    id: "file-library",
    enter: resetGalleryFromUrl,
  },
  "/settings": { page: "settings", component: SettingsPage },
  "/tokens": { page: "tokens", component: TokensPage },
  "/admin": { page: "admin", component: AdminPage },
};

// The whole frontend: one app whose pages switch without reloading.
export function App(props: { mascot: string }) {
  const { context, error, retry } = useFrontendContext();
  const route = () => ROUTES[path()];
  const Page = dynamic(() => route().component);

  onSettled(() => startRouter((to) => ROUTES[to].enter?.()));

  // The home page takes the server's branding (see Navigation).
  createEffect(path, (current) => {
    if (ROUTES[current].page !== "home") document.title = PAGE_TITLES[current];
  });

  const loading = () => (!context() && !error()) || isPageLoading();
  // The first load shows the loading screen at once, continuing the one in
  // the static HTML; later ones only if they take a moment.
  let firstLoadDone = false;
  const delayed = createMemo(() => {
    const isLoading = loading();
    const delay = firstLoadDone;
    if (!isLoading) firstLoadDone = true;
    return delay;
  });

  // A failed refresh keeps the last context, so this only covers the start.
  const startupFailed = () => !context() && !!error();

  return (
    <Show
      when={!startupFailed()}
      fallback={
        <StartupError
          mascot={props.mascot}
          title={cachedBranding()}
          message={error()}
          onRetry={retry}
        />
      }
    >
      <div class={["app", { loading: loading(), delayed: delayed() }]}>
        <Navigation currentPage={route().page} />
        <main>
          <div class="container">
            <Show when={route().page !== "home"}>
              <h1>{PAGE_TITLES[path()]}</h1>
            </Show>
            {/* Keyed on the visit too, so following a link to the current page
                starts it afresh. */}
            <Show when={`${path()}#${visit()}`} keyed>
              <div
                class="account-sections"
                id={route().id ?? `${route().page}-page`}
              >
                <Page />
              </div>
            </Show>
          </div>
        </main>
        <HostlingToast />
        <ConfirmDialog />
        <Show when={loading()}>
          <LoadingScreen
            mascot={props.mascot}
            title={context()?.branding ?? cachedBranding()}
          />
        </Show>
      </div>
    </Show>
  );
}

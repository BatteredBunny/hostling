import { Show, createEffect, createMemo, onCleanup, onMount } from "solid-js";
import type { Component } from "solid-js";
import { Navigation } from "./Navigation";
import { LoadingScreen } from "./LoadingScreen";
import { HostlingToast } from "./HostlingToast";
import { ConfirmDialog } from "./ConfirmDialog";
import { UploadPage } from "./UploadPage";
import { LoginPage } from "./LoginPage";
import { RegistrationPage } from "./RegistrationPage";
import { FileLibrary } from "./FileLibrary";
import { SettingsPage } from "./SettingsPage";
import { TokensPage } from "./TokensPage";
import { AdminPage } from "./AdminPage";
import { path, startRouter, visit, type AppPath } from "../router";
import { isPageLoading } from "../loading";
import { useFrontendContext } from "../context";
import { cx } from "../utils";
import { cachedBranding } from "../branding";

interface Route {
  // Names the page for the navigation and the page's container id.
  page: string;
  title: string;
  component: Component;
  // The id styles hang off, defaulting to `${page}-page`.
  id?: string;
}

export const ROUTES: Record<AppPath, Route> = {
  "/": { page: "home", title: "Upload", component: UploadPage },
  "/login": { page: "login", title: "Login", component: LoginPage },
  "/register": {
    page: "register",
    title: "Register",
    component: RegistrationPage,
  },
  "/gallery": {
    page: "gallery",
    title: "Gallery",
    component: FileLibrary,
    id: "file-library",
  },
  "/settings": { page: "settings", title: "Settings", component: SettingsPage },
  "/tokens": { page: "tokens", title: "Tokens", component: TokensPage },
  "/admin": { page: "admin", title: "Admin", component: AdminPage },
};

// Rendered once per visit (see the keyed Show below), so reading the
// component once is enough.
function PageContent(props: { route: Route }) {
  return <props.route.component />;
}

// The whole frontend: one app whose pages switch without reloading.
export function App(props: { mascot: string }) {
  const { context, error } = useFrontendContext();
  const route = () => ROUTES[path()];

  onMount(() => onCleanup(startRouter()));

  // The home page takes the server's branding (see Navigation).
  createEffect(() => {
    if (route().page !== "home") document.title = route().title;
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

  return (
    <div class={cx("app", { loading: loading(), delayed: delayed() })}>
      <Navigation currentPage={route().page} />
      <main>
        <div class="container">
          <Show when={route().page !== "home"}>
            <h1>{route().title}</h1>
          </Show>
          {/* Keyed on the visit too, so following a link to the current page
              starts it afresh. */}
          <Show when={`${path()}#${visit()}`} keyed>
            <div
              class="account-sections"
              id={route().id ?? `${route().page}-page`}
            >
              <PageContent route={route()} />
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
  );
}

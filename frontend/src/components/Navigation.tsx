import styles from "./Navigation.module.css";
import { For, Show, createEffect } from "solid-js";
import { useFrontendContext } from "../context";
import {
  GitMerge as GitMergeIcon,
  House as HouseIcon,
  Images as ImagesIcon,
  KeyRound as KeyRoundIcon,
  LogIn as LogInIcon,
  LogOut as LogOutIcon,
  Server as ServerIcon,
  Settings as SettingsIcon,
  TriangleAlert as TriangleAlertIcon,
  UserPlus as UserPlusIcon,
} from "lucide-solid";
import { Dynamic } from "solid-js/web";

export function Navigation(props: { currentPage: string }) {
  const { context, error, retry } = useFrontendContext();
  let previousAccount: number | undefined;
  createEffect(() => {
    const info = context();
    if (!info) return;
    const protectedPage = ["gallery", "settings", "tokens", "admin"].includes(
      props.currentPage,
    );
    if (!info.logged_in && protectedPage) window.location.assign("/login");
    else if (!info.is_admin && props.currentPage === "admin")
      window.location.assign("/gallery");
    else if (
      protectedPage &&
      previousAccount !== undefined &&
      previousAccount !== info.account_id
    )
      window.location.reload();
    previousAccount = info.account_id;
    if (props.currentPage === "home") document.title = info.branding;
    for (const name of ["description", "twitter:description"]) {
      let element = document.querySelector<HTMLMetaElement>(
        `meta[name="${name}"]`,
      );
      if (!element) {
        element = document.createElement("meta");
        element.name = name;
        document.head.append(element);
      }
      element.content = info.tagline;
    }
  });
  const links = [
    { page: "gallery", label: "Gallery", icon: ImagesIcon },
    { page: "settings", label: "Settings", icon: SettingsIcon },
    { page: "tokens", label: "Tokens", icon: KeyRoundIcon },
  ];
  return (
    <div id="toolbar" class={styles.navigation}>
      <div class={styles.brand}>
        <strong class="brand-text">{context()?.branding ?? "…"}</strong>
      </div>
      <div class={styles.links}>
        <a
          class={styles.option}
          classList={{ [styles.active]: props.currentPage === "home" }}
          href="/"
          title="home"
        >
          <HouseIcon class="lucide-icon" aria-hidden="true" />
          <span>Home</span>
        </a>
        <Show
          when={context()?.logged_in}
          fallback={
            <Show when={context()}>
              <a
                class={styles.option}
                classList={{ [styles.active]: props.currentPage === "login" }}
                href="/login"
              >
                <LogInIcon class="lucide-icon" aria-hidden="true" />
                <span>Login</span>
              </a>
              <a
                class={styles.option}
                classList={{
                  [styles.active]: props.currentPage === "register",
                }}
                href="/register"
              >
                <UserPlusIcon class="lucide-icon" aria-hidden="true" />
                <span>Register</span>
              </a>
            </Show>
          }
        >
          <Show when={context()?.is_admin}>
            <a
              href="/admin"
              class={styles.option}
              classList={{
                [styles.active]: props.currentPage === "admin",
                [styles.warning]: context()?.has_admin_warning,
              }}
            >
              <ServerIcon class="lucide-icon" aria-hidden="true" />
              <span>Admin</span>
              <Show when={context()?.has_admin_warning}>
                <TriangleAlertIcon
                  class={`lucide-icon ${styles.warningIcon}`}
                  aria-hidden="true"
                />
              </Show>
            </a>
          </Show>
          <div class={styles.divider} />
          <For each={links}>
            {(link) => (
              <a
                href={`/${link.page}`}
                class={styles.option}
                classList={{ [styles.active]: props.currentPage === link.page }}
              >
                <Dynamic
                  component={link.icon}
                  class="lucide-icon"
                  aria-hidden="true"
                />
                <span>{link.label}</span>
              </a>
            )}
          </For>
          <div class={styles.divider} />
          <form method="post" action="/logout" class={styles.logoutForm}>
            <button class={styles.option} type="submit" title="logout">
              <LogOutIcon class="lucide-icon" aria-hidden="true" />
              <span>Logout</span>
            </button>
          </form>
        </Show>
        <Show when={error()}>
          <button class={styles.option} onClick={retry}>
            Retry navigation
          </button>
        </Show>
      </div>
      <div class={styles.footer}>
        <a
          class={styles.option}
          href="https://github.com/BatteredBunny/hostling"
          target="_blank"
          rel="noopener noreferrer"
          title="code"
        >
          <GitMergeIcon class="lucide-icon" aria-hidden="true" />
          <span>Code</span>
        </a>
      </div>
    </div>
  );
}

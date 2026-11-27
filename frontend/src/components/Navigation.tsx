import styles from "./Navigation.module.css";
import { For, Show, createEffect } from "solid-js";
import { useFrontendContext } from "../context";
import { navigate } from "../router";
import GitMergeIcon from "@lucide/solid/icons/git-merge";
import HouseIcon from "@lucide/solid/icons/house";
import ImagesIcon from "@lucide/solid/icons/images";
import KeyRoundIcon from "@lucide/solid/icons/key-round";
import LogInIcon from "@lucide/solid/icons/log-in";
import LogOutIcon from "@lucide/solid/icons/log-out";
import ServerIcon from "@lucide/solid/icons/server";
import SettingsIcon from "@lucide/solid/icons/settings";
import TriangleAlertIcon from "@lucide/solid/icons/triangle-exclamation-point";
import UserPlusIcon from "@lucide/solid/icons/user-plus";
import { logout } from "../generated";
import { apiData } from "../client";
import { showToast } from "../toast";

export function Navigation(props: { currentPage: string }) {
  const { context, error, retry } = useFrontendContext();
  let previousAccount: number | undefined;
  createEffect(
    () => ({ info: context(), page: props.currentPage }),
    ({ info, page }) => {
      if (!info) return;
      const protectedPage = ["gallery", "settings", "tokens", "admin"].includes(
        page,
      );
      if (!info.logged_in && protectedPage)
        navigate("/login", { replace: true });
      else if (info.logged_in && ["login", "register"].includes(page))
        navigate("/gallery", { replace: true });
      else if (!info.is_admin && page === "admin")
        navigate("/gallery", { replace: true });
      else if (
        protectedPage &&
        previousAccount !== undefined &&
        previousAccount !== info.account_id
      )
        window.location.reload();
      previousAccount = info.account_id;
      if (page === "home") document.title = info.branding;
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
    },
  );
  const signOut = async () => {
    try {
      await apiData(logout());
    } catch (failure) {
      showToast(
        failure instanceof Error ? failure.message : "Could not log out.",
        "error",
      );
      return;
    }
    // A fresh document drops everything loaded for the account.
    window.location.assign("/");
  };
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
          class={[
            styles.option,
            {
              [styles.active]: props.currentPage === "home",
            },
          ]}
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
                class={[
                  styles.option,
                  {
                    [styles.active]: props.currentPage === "login",
                  },
                ]}
                href="/login"
              >
                <LogInIcon class="lucide-icon" aria-hidden="true" />
                <span>Login</span>
              </a>
              <a
                class={[
                  styles.option,
                  {
                    [styles.active]: props.currentPage === "register",
                  },
                ]}
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
              class={[
                styles.option,
                {
                  [styles.active]: props.currentPage === "admin",
                  [styles.warning]: context()?.has_admin_warning,
                },
              ]}
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
                class={[
                  styles.option,
                  {
                    [styles.active]: props.currentPage === link.page,
                  },
                ]}
              >
                <link.icon class="lucide-icon" aria-hidden="true" />
                <span>{link.label}</span>
              </a>
            )}
          </For>
          <div class={styles.divider} />
          <button
            class={styles.option}
            type="button"
            title="logout"
            disabled={import.meta.env.PUBLIC_DEMO}
            onClick={signOut}
          >
            <LogOutIcon class="lucide-icon" aria-hidden="true" />
            <span>Logout</span>
          </button>
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

import { For, Show } from "solid-js";
import EllipsisIcon from "@lucide/solid/icons/ellipsis";
import LoaderCircleIcon from "@lucide/solid/icons/loader-circle";
import type { AccountStats as Account } from "../generated";
import {
  formatTimeDate,
  hasDate,
  humanizeBytes,
  relativeTime,
  ariaBool,
} from "../utils";
import { createContextMenuTrigger } from "../contextMenuTrigger";
import { ProviderIcon, providerLabel } from "./ProviderAction";
import styles from "./AdminUserCard.module.css";

export function linkedAccounts(account: Account) {
  return [
    { provider: "github", username: account.GithubUsername },
    { provider: "openid-connect", username: account.OIDCUsername },
  ].filter((link) => link.username);
}

// The name people recognise: a linked username, else the account number.
export function accountName(account: Account): string {
  return linkedAccounts(account)[0]?.username ?? `User ${account.ID}`;
}

export function AdminUserCard(props: {
  account: Account;
  busy: boolean;
  menuOpen: boolean;
  // The trigger gets focus back when the menu closes.
  onOpenMenu: (x: number, y: number, trigger: HTMLElement) => void;
}) {
  const account = () => props.account;
  let menuButton!: HTMLButtonElement;

  const menuTrigger = createContextMenuTrigger((x, y) => {
    if (!props.menuOpen && !props.busy) props.onOpenMenu(x, y, menuButton);
  }, `.${styles.menuButton}`);

  const openFromButton = () => {
    if (props.busy) return;
    const rect = menuButton.getBoundingClientRect();
    // Open leftwards from the button's right edge; the menu flips if needed.
    props.onOpenMenu(rect.right, rect.bottom + 4, menuButton);
  };

  const stats = () => [
    {
      label: "Storage",
      value: humanizeBytes(account().SpaceUsed),
      title: `${account().SpaceUsed} bytes`,
    },
    { label: "Files", value: String(account().FilesUploaded) },
    { label: "Sessions", value: String(account().SessionsCount) },
    {
      label: "Tokens",
      value: String(account().UploadTokensCount),
      title: "Upload tokens",
    },
  ];

  return (
    <article
      class={[styles.card, { [styles.menuOpen]: props.menuOpen }]}
      aria-label={`User ${account().ID}`}
      aria-busy={ariaBool(props.busy)}
      onContextMenu={menuTrigger.onContextMenu}
      onTouchStart={menuTrigger.onTouchStart}
      onTouchMove={menuTrigger.onTouchMove}
      onTouchEnd={menuTrigger.onTouchEnd}
      onTouchCancel={menuTrigger.onTouchCancel}
    >
      <header class={styles.header}>
        <h3 class={styles.name}>
          <span>{accountName(account())}</span>
          <span class={styles.id}>#{account().ID}</span>
        </h3>
        <div class={styles.badges}>
          <Show when={account().You}>
            <span class={[styles.badge, styles.you]}>You</span>
          </Show>
          <span class={styles.badge}>{account().AccountType}</span>
        </div>
        <button
          type="button"
          ref={menuButton}
          class={styles.menuButton}
          aria-label={`Actions for ${accountName(account())}`}
          aria-haspopup="menu"
          aria-expanded={ariaBool(props.menuOpen)}
          // aria-disabled rather than disabled: a disabled button loses focus,
          // which should stay put while the action runs.
          aria-disabled={ariaBool(props.busy)}
          onClick={openFromButton}
        >
          <Show
            when={props.busy}
            fallback={<EllipsisIcon class="lucide-icon" aria-hidden="true" />}
          >
            <LoaderCircleIcon class="lucide-icon spinning" aria-hidden="true" />
          </Show>
        </button>
      </header>

      <ul class={styles.links} aria-label="Linked accounts">
        <For
          each={linkedAccounts(account())}
          fallback={<li class={styles.muted}>No linked accounts</li>}
        >
          {(link) => (
            <li>
              <ProviderIcon name={link.provider} />
              <span>
                {providerLabel(link.provider)}
                {/* The title already shows the first linked username. */}
                <Show when={link.username !== accountName(account())}>
                  <span class={styles.muted}> as {link.username}</span>
                </Show>
              </span>
            </li>
          )}
        </For>
      </ul>

      <dl class={styles.stats}>
        <For each={stats()}>
          {(stat) => (
            <div class={styles.stat} title={stat.title}>
              <dt>{stat.label}</dt>
              <dd>{stat.value}</dd>
            </div>
          )}
        </For>
      </dl>

      <p class={styles.meta}>
        <span>
          Joined{" "}
          <span title={formatTimeDate(account().CreatedAt)}>
            {relativeTime(account().CreatedAt)}
          </span>
        </span>
        <span>
          Active{" "}
          <Show when={hasDate(account().LastActivity)} fallback="never">
            <span title={formatTimeDate(account().LastActivity)}>
              {relativeTime(account().LastActivity)}
            </span>
          </Show>
        </span>
        <Show when={account().InvitedBy}>
          <span>Invited by {account().InvitedBy}</span>
        </Show>
      </p>
    </article>
  );
}

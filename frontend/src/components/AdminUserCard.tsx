import { For, Show } from "solid-js";
import { Dynamic } from "solid-js/web";
import type { AccountStats as Account } from "../generated";
import { formatTimeDate, humanizeBytes, relativeTime } from "../utils";
import {
  Calendar as CalendarIcon,
  File as FileIcon,
  HardDrive as HardDriveIcon,
  Key as KeyIcon,
  User as UserIcon,
} from "lucide-solid";
import styles from "./AdminUserCard.module.css";
import { providerPresentation } from "./ProviderAction";

export type AdminAction =
  | "user"
  | "files"
  | "sessions"
  | "upload_tokens"
  | "give_invite_code";

function rows(account: Account) {
  const github = providerPresentation("github");
  const oidc = providerPresentation("openid-connect");
  return [
    {
      icon: github.icon,
      name: github.label,
      value: account.GithubUsername || "-",
    },
    { icon: oidc.icon, name: oidc.label, value: account.OIDCUsername || "-" },
    {
      icon: CalendarIcon,
      name: "Creation",
      value: relativeTime(account.CreatedAt),
      title: formatTimeDate(account.CreatedAt),
    },
    {
      icon: CalendarIcon,
      name: "Last activity",
      value: account.LastActivity.startsWith("0001-")
        ? "Never"
        : relativeTime(account.LastActivity),
      title: formatTimeDate(account.LastActivity),
    },
    { icon: UserIcon, name: "Invited by", value: account.InvitedBy },
    {
      icon: HardDriveIcon,
      name: "Space used",
      value: humanizeBytes(account.SpaceUsed),
      title: `${account.SpaceUsed} bytes`,
    },
    {
      icon: FileIcon,
      name: "Files uploaded",
      value: String(account.FilesUploaded),
    },
    {
      icon: KeyIcon,
      name: "Sessions/Upload tokens",
      value: `${account.SessionsCount}/${account.UploadTokensCount}`,
    },
  ];
}

export function AdminUserCard(props: {
  account: Account;
  busy: boolean;
  onAction: (kind: AdminAction, message: string) => void;
}) {
  return (
    <article class={styles.userCard} aria-label={`User ${props.account.ID}`}>
      <div class={styles.topRow}>
        <div class={styles.left}>User {props.account.ID}</div>
        <div class={styles.right}>
          <Show when={props.account.You}>
            <div class={styles.badge}>You</div>
          </Show>
          <div class={styles.badge}>{props.account.AccountType}</div>
        </div>
      </div>
      <div class={styles.middleRow}>
        <For each={rows(props.account)}>
          {(row) => (
            <div class={styles.entry}>
              <div class={styles.name}>
                <Dynamic
                  component={row.icon}
                  class="lucide-icon"
                  aria-hidden="true"
                />
                <span>{row.name}</span>
              </div>
              <div title={row.title}>{row.value}</div>
            </div>
          )}
        </For>
      </div>
      <div class={styles.bottomRow}>
        <Show when={!props.account.You}>
          <button
            type="button"
            class="delete-button"
            disabled={props.busy}
            onClick={() =>
              props.onAction(
                "user",
                "Delete this user and all their data permanently?",
              )
            }
          >
            Delete user
          </button>
        </Show>
        <button
          type="button"
          class="delete-button"
          disabled={props.busy}
          onClick={() =>
            props.onAction(
              "files",
              "Delete all files for this user permanently?",
            )
          }
        >
          Delete files
        </button>
        <button
          type="button"
          class="delete-button"
          disabled={props.busy}
          onClick={() =>
            props.onAction(
              "sessions",
              "Delete all sessions for this user? They will be logged out immediately.",
            )
          }
        >
          Delete sessions
        </button>
        <button
          type="button"
          class="delete-button"
          disabled={props.busy}
          onClick={() =>
            props.onAction(
              "upload_tokens",
              "Delete all upload tokens for this user?",
            )
          }
        >
          Delete upload tokens
        </button>
        <button
          type="button"
          class="create-button"
          disabled={props.busy}
          onClick={() => props.onAction("give_invite_code", "")}
        >
          Give invitation code
        </button>
      </div>
    </article>
  );
}

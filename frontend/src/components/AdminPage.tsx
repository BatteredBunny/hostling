import "./AdminPage.css";
import { LoadError } from "./LoadError";
import { AdminUserCard, accountName } from "./AdminUserCard";
import { WarningCallout } from "./WarningCallout";
import { SectionPanel } from "./SectionPanel";
import {
  For,
  Show,
  createSignal,
  createStore,
  onSettled,
  reconcile,
} from "solid-js";
import KeyRoundIcon from "@lucide/solid/icons/key-round";
import LogOutIcon from "@lucide/solid/icons/log-out";
import TicketIcon from "@lucide/solid/icons/ticket";
import Trash2Icon from "@lucide/solid/icons/trash";
import UserXIcon from "@lucide/solid/icons/user-x";
import * as sdk from "../generated";
import type {
  AccountStats as Account,
  AdminOverviewOutput,
} from "../generated";
import { apiData } from "../client";
import { countLabel, humanizeBytes } from "../utils";
import { showToast } from "../toast";
import { ContextMenu, type ContextMenuItem } from "./ContextMenu";
import { SecretDialog } from "./SecretDialog";
import { confirmDialog } from "../store";
import { trackPageLoad } from "../loading";
import { ProviderIcon, providerLabel } from "./ProviderAction";

interface MenuState {
  account: Account;
  x: number;
  y: number;
  trigger: HTMLElement;
}

export function AdminPage() {
  const [overview, setOverview] = createSignal<AdminOverviewOutput>();
  // Reconciled by ID so refreshing after an action updates cards in place,
  // keeping focus on the card's menu button.
  const [accounts, setAccounts] = createStore<Account[]>([]);
  const [loadError, setLoadError] = createSignal(false);
  const [error, setError] = createSignal("");
  const [pending, setPending] = createSignal<ReadonlySet<number>>(new Set());
  const [menu, setMenu] = createSignal<MenuState | null>(null);
  const [invite, setInvite] = createSignal<{ code: string; for: Account }>();

  async function load() {
    try {
      const data = await apiData(sdk.getAdminOverview());
      setAccounts(reconcile(data.accounts ?? [], "ID"));
      setOverview(data);
      setLoadError(false);
    } catch {
      setLoadError(true);
    }
  }
  onSettled(() => void trackPageLoad(load()));

  const retry = () => {
    setLoadError(false);
    void trackPageLoad(load());
  };

  const closeMenu = (restoreFocus: boolean) => {
    const current = menu();
    setMenu(null);
    if (restoreFocus) current?.trigger.focus();
  };

  const setAccountPending = (id: number, busy: boolean) =>
    setPending((previous) => {
      const next = new Set(previous);
      if (busy) next.add(id);
      else next.delete(id);
      return next;
    });

  // Resolves to the response, or undefined after reporting a failure.
  async function run<T>(
    account: Account,
    request: () => Promise<T>,
  ): Promise<T | undefined> {
    setAccountPending(account.ID, true);
    setError("");
    try {
      return await request();
    } catch (failure) {
      setError(
        failure instanceof Error
          ? failure.message
          : "Could not complete the action. Please try again.",
      );
      return undefined;
    } finally {
      setAccountPending(account.ID, false);
    }
  }

  async function destructive(
    account: Account,
    dialog: { title: string; message: string; confirmLabel: string },
    request: () => Promise<unknown>,
    done: string,
  ) {
    if (!(await confirmDialog({ ...dialog, danger: true }))) return;
    if ((await run(account, request)) === undefined) return;
    showToast(done);
    await load();
  }

  async function giveInvite(account: Account) {
    const code = await run(account, () =>
      apiData(sdk.giveInviteCode({ body: { id: account.ID } })),
    );
    if (code) setInvite({ code, for: account });
  }

  function menuItems(account: Account): ContextMenuItem[] {
    const name = accountName(account);
    const files = `${countLabel(account.FilesUploaded, "file")} (${humanizeBytes(account.SpaceUsed)})`;
    const items: ContextMenuItem[] = [
      {
        label: "Give invitation code",
        icon: TicketIcon,
        action: () => giveInvite(account),
      },
      {
        label: "Sign out of all sessions",
        icon: LogOutIcon,
        separatorBefore: true,
        disabled: account.SessionsCount === 0,
        action: async () => {
          if (account.You) {
            const confirmed = await confirmDialog({
              title: "Sign out of all sessions?",
              message:
                "This signs you out everywhere, including here. You'll need to log in again.",
              confirmLabel: "Sign out",
              danger: true,
            });
            if (
              confirmed &&
              (await run(account, () =>
                apiData(sdk.deleteUserSessions({ body: { id: account.ID } })),
              )) !== undefined
            )
              window.location.assign("/login");
            return;
          }
          await destructive(
            account,
            {
              title: "Sign out of all sessions?",
              message: `${name} will be signed out of ${countLabel(account.SessionsCount, "session")}.`,
              confirmLabel: "Sign out",
            },
            () => apiData(sdk.deleteUserSessions({ body: { id: account.ID } })),
            `Signed out ${name}.`,
          );
        },
      },
      {
        label: "Revoke upload tokens",
        icon: KeyRoundIcon,
        disabled: account.UploadTokensCount === 0,
        action: () =>
          destructive(
            account,
            {
              title: "Revoke upload tokens?",
              message: `${countLabel(account.UploadTokensCount, "upload token")} belonging to ${name} will stop working. Apps and scripts using them can no longer upload.`,
              confirmLabel: "Revoke",
            },
            () =>
              apiData(sdk.deleteUserUploadTokens({ body: { id: account.ID } })),
            `Revoked ${name}'s upload tokens.`,
          ),
      },
      {
        label: "Delete all files",
        icon: Trash2Icon,
        danger: true,
        disabled: account.FilesUploaded === 0,
        action: () =>
          destructive(
            account,
            {
              title: "Delete all files?",
              message: `${files} uploaded by ${name} will be permanently deleted.`,
              confirmLabel: "Delete files",
            },
            () => apiData(sdk.deleteUserFiles({ body: { id: account.ID } })),
            `Deleted ${name}'s files.`,
          ),
      },
    ];
    if (!account.You)
      items.push({
        label: "Delete user",
        icon: UserXIcon,
        danger: true,
        separatorBefore: true,
        action: () =>
          destructive(
            account,
            {
              title: "Delete user?",
              message: `${name} and all their data will be permanently deleted, including ${files}.`,
              confirmLabel: "Delete user",
            },
            () => apiData(sdk.deleteUser({ body: { id: account.ID } })),
            `Deleted ${name}.`,
          ),
      });
    return items;
  }

  return (
    <>
      <Show when={loadError()}>
        <LoadError message="Could not load admin data." onRetry={retry} />
      </Show>
      <Show when={overview()}>
        {(info) => (
          <>
            <SectionPanel id="server-config-panel" title="Server info">
              <ServerWarnings overview={info()} />
              <dl class="server-info">
                <div>
                  <dt>Version</dt>
                  <dd>{info().version}</dd>
                </div>
                <div>
                  <dt>Max upload size</dt>
                  <dd title={`${info().max_upload_size} bytes`}>
                    {humanizeBytes(info().max_upload_size)}
                  </dd>
                </div>
                <div>
                  <dt>File storage</dt>
                  <dd>{info().file_storage_method}</dd>
                </div>
                <div>
                  <dt>Login providers</dt>
                  <dd class="server-info-providers">
                    <For
                      each={info().login_providers}
                      fallback="None configured"
                    >
                      {(provider) => (
                        <span>
                          <ProviderIcon name={provider} />
                          {providerLabel(provider)}
                        </span>
                      )}
                    </For>
                  </dd>
                </div>
              </dl>
            </SectionPanel>
            <SectionPanel
              id="users-panel"
              title="Users"
              headerActions={
                <span class="users-count">
                  {countLabel(accounts.length, "user")}
                </span>
              }
            >
              <Show when={error()}>
                <p class="admin-error" role="alert">
                  {error()}
                </p>
              </Show>
              <div class="users-grid">
                <For each={accounts}>
                  {(account) => (
                    <AdminUserCard
                      account={account}
                      busy={pending().has(account.ID)}
                      menuOpen={menu()?.account.ID === account.ID}
                      onOpenMenu={(x, y, trigger) =>
                        setMenu({ account, x, y, trigger })
                      }
                    />
                  )}
                </For>
              </div>
            </SectionPanel>
          </>
        )}
      </Show>
      <Show when={menu()} keyed>
        {(current) => (
          <ContextMenu
            x={current.x}
            y={current.y}
            label={accountName(current.account)}
            items={menuItems(current.account)}
            onClose={closeMenu}
          />
        )}
      </Show>
      <Show when={invite()}>
        {(given) => (
          <SecretDialog
            title="Invitation code created"
            notice={
              given().for.You
                ? "Share this code to let someone create an account. You can also find it on your Tokens page."
                : `Share this code to let someone create an account. ${accountName(given().for)} can also find it on their Tokens page.`
            }
            noun="code"
            value={given().code}
            onDismiss={() => setInvite(undefined)}
          />
        )}
      </Show>
    </>
  );
}

function ServerWarnings(props: { overview: AdminOverviewOutput }) {
  return (
    <>
      <Show when={props.overview.no_providers_configured}>
        <WarningCallout>
          <p>
            <strong>Critical Server Configuration Error:</strong> No
            authentication providers are configured on this server.
          </p>
          <p>Configure at least one authentication provider:</p>
          <ul>
            <li>
              <strong>GitHub OAuth:</strong> Set{" "}
              <strong>GITHUB_CLIENT_ID</strong> and{" "}
              <strong>GITHUB_SECRET</strong>.
            </li>
            <li>
              <strong>OpenID Connect:</strong> Set{" "}
              <strong>OPENID_CONNECT_CLIENT_ID</strong>,{" "}
              <strong>OPENID_CONNECT_CLIENT_SECRET</strong>, and{" "}
              <strong>OPENID_CONNECT_DISCOVERY_URL</strong>.
            </li>
          </ul>
          <p>
            Without authentication providers, users cannot log in or link
            accounts. Restart the server after setting the required environment
            variables.
          </p>
        </WarningCallout>
      </Show>
      <Show when={props.overview.failed_providers?.length}>
        <WarningCallout>
          <p>
            <strong>Provider Initialization Error:</strong> These providers
            failed to initialize:
          </p>
          <ul>
            <For each={props.overview.failed_providers}>
              {(provider) => (
                <li>
                  <strong>{providerLabel(provider)}</strong>
                </li>
              )}
            </For>
          </ul>
          <p>
            Login will be attempted on demand. Check that the provider is
            running and its configuration is correct.
          </p>
        </WarningCallout>
      </Show>
    </>
  );
}

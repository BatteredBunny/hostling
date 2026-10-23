import "./AdminPage.css";
import { AdminUserCard } from "./AdminUserCard";
import { WarningCallout } from "./WarningCallout";
import { SectionPanel } from "./SectionPanel";
import { LoadingSections } from "./LoadingSections";
import { For, Show, createResource, createSignal } from "solid-js";
import * as sdk from "../generated";
import type { AccountStats as Account } from "../generated";
import { apiData } from "../client";
import { humanizeBytes } from "../utils";
import { showToast } from "../toast";
import { HostlingToast } from "./HostlingToast";
import { ConfirmDialog } from "./ConfirmDialog";
import { confirmDialog } from "../store";
import { providerLabel } from "./ProviderAction";

export function AdminPage() {
  const [data, { refetch }] = createResource(() =>
    apiData(sdk.getAdminOverview()),
  );
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal("");
  const actions = {
    user: sdk.deleteUser,
    files: sdk.deleteUserFiles,
    sessions: sdk.deleteUserSessions,
    upload_tokens: sdk.deleteUserUploadTokens,
    give_invite_code: sdk.giveInviteCode,
  };
  async function action(
    account: Account,
    kind: keyof typeof actions,
    message: string,
  ) {
    if (busy()) return;
    if (
      kind !== "give_invite_code" &&
      !(await confirmDialog({
        title: "Confirm action",
        message,
        confirmLabel: "Delete",
        danger: true,
      }))
    )
      return;
    setBusy(true);
    setError("");
    try {
      await apiData(actions[kind]({ body: { id: account.ID } }));
      showToast(
        kind === "give_invite_code" ? "Invitation code given." : "Deleted.",
      );
      if (kind === "sessions" && account.You) window.location.assign("/login");
      else await refetch();
    } catch {
      setError("Could not complete the action. Please try again.");
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <Show when={data.error}>
        <p role="alert">
          Could not load admin data.{" "}
          <button onClick={() => refetch()}>Retry</button>
        </p>
      </Show>
      <Show when={data.loading && !data.error && !data()}>
        <LoadingSections sections={["Server info", "Users"]} />
      </Show>
      <Show when={error()}>
        <p role="alert">{error()}</p>
      </Show>
      <Show when={!data.error && data()}>
        {(overview) => (
          <>
            <Show when={overview().no_providers_configured}>
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
                  accounts. Restart the server after setting the required
                  environment variables.
                </p>
              </WarningCallout>
            </Show>
            <Show when={overview().failed_providers?.length}>
              <WarningCallout>
                <p>
                  <strong>Provider Initialization Error:</strong> These
                  providers failed to initialize:
                </p>
                <ul>
                  <For each={overview().failed_providers}>
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
            <SectionPanel id="server-config-panel" title="Server info">
              <p>
                Max upload size:{" "}
                <span title={`${overview().max_upload_size} bytes`}>
                  {humanizeBytes(overview().max_upload_size)}
                </span>
              </p>
              <p>File storage method: {overview().file_storage_method}</p>
              <p>
                Login providers:{" "}
                {overview().login_providers?.map(providerLabel).join(", ") ||
                  "None configured"}
              </p>
              <p>Hostling version: {overview().version}</p>
            </SectionPanel>
            <SectionPanel id="users-panel" title="Users">
              <div class="users-grid">
                <For each={overview().accounts}>
                  {(account) => (
                    <AdminUserCard
                      account={account}
                      busy={busy()}
                      onAction={(kind, message) =>
                        action(account, kind, message)
                      }
                    />
                  )}
                </For>
              </div>
            </SectionPanel>
          </>
        )}
      </Show>
      <ConfirmDialog />
      <HostlingToast />
    </>
  );
}

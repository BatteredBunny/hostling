import "./SettingsPage.css";
import { WarningCallout } from "./WarningCallout";
import { SectionPanel } from "./SectionPanel";
import { LoadingSections } from "./LoadingSections";
import { For, Show, createResource, createSignal } from "solid-js";
import { getSettings, deleteAccount, deleteAllFiles } from "../generated";
import { apiData } from "../client";
import { showToast } from "../toast";
import { HostlingToast } from "./HostlingToast";
import { NativeConfirmation } from "./NativeConfirmation";
import { Unlink } from "lucide-solid";
import { ProviderButton, ProviderIcon, providerLabel } from "./ProviderAction";
import { ConfirmDialog } from "./ConfirmDialog";
import { confirmDialog } from "../store";
import { useFrontendContext } from "../context";

export function SettingsPage() {
  const {
    context,
    error: contextError,
    retry: retryContext,
  } = useFrontendContext();
  const [data, { refetch }] = createResource(
    () => (context()?.logged_in ? context() : undefined),
    () => apiData(getSettings()),
  );
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal("");
  let accountDialog: HTMLDialogElement | undefined;
  let filesDialog: HTMLDialogElement | undefined;
  async function remove(account: boolean) {
    if (busy()) return;
    setBusy(true);
    setError("");
    try {
      await apiData(account ? deleteAccount() : deleteAllFiles());
      if (account) window.location.assign("/");
      else {
        setBusy(false);
        filesDialog?.close();
        showToast("All files have been deleted.");
      }
    } catch (failure) {
      setError(
        failure instanceof Error
          ? failure.message
          : "Could not complete deletion.",
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <Show when={data.error || contextError()}>
        <p role="alert">
          Could not load settings.{" "}
          <button
            onClick={() => {
              retryContext();
              refetch();
            }}
          >
            Retry
          </button>
        </p>
      </Show>
      <Show
        when={
          !data.error &&
          !contextError() &&
          (!context() || (data.loading && !data()))
        }
      >
        <LoadingSections sections={["Social login", "Account"]} />
      </Show>
      <Show when={!data.error && data()}>
        {(settings) => (
          <>
            <SectionPanel
              id="social-login"
              title="Social login"
              bodyClass="provider-list"
            >
              <Show
                when={settings().no_providers_configured}
                fallback={
                  <Show when={settings().unlinked_account}>
                    <WarningCallout>
                      <p>
                        Link your account with at least one platform so you can
                        log in again.
                      </p>
                    </WarningCallout>
                  </Show>
                }
              >
                <WarningCallout>
                  <p>
                    <strong>Server Configuration Error:</strong> No
                    authentication providers are configured. Contact the server
                    administrator before your session expires.
                  </p>
                </WarningCallout>
              </Show>
              <For each={settings().providers}>
                {(provider) => (
                  <Show
                    when={provider.IsLinked}
                    fallback={
                      <form
                        class="provider"
                        method="post"
                        action={`/api/auth/link/${provider.Name}`}
                      >
                        <ProviderButton type="submit" provider={provider.Name}>
                          Link with {providerLabel(provider.Name)}
                        </ProviderButton>
                      </form>
                    }
                  >
                    <div class="linked-provider">
                      <Show
                        when={provider.ProfileURL}
                        fallback={
                          <div class="linked-account">
                            <ProviderIcon name={provider.Name} />
                            <span>{provider.Username}</span>
                          </div>
                        }
                      >
                        <a
                          href={provider.ProfileURL}
                          target="_blank"
                          rel="noopener noreferrer"
                          class="linked-account"
                        >
                          <ProviderIcon name={provider.Name} />
                          <span>{provider.Username}</span>
                        </a>
                      </Show>
                      <form
                        method="post"
                        action={`/api/auth/unlink/${provider.Name}`}
                        class="unlink-form"
                        onSubmit={async (event) => {
                          event.preventDefault();
                          const form = event.currentTarget;
                          if (
                            await confirmDialog({
                              title: "Unlink provider?",
                              message:
                                "You won't be able to log in with this provider anymore.",
                              confirmLabel: "Unlink",
                              danger: true,
                            })
                          )
                            form.submit();
                        }}
                      >
                        <button
                          type="submit"
                          class="unlink-button"
                          title={`Unlink ${providerLabel(provider.Name)}`}
                        >
                          <Unlink class="lucide-icon" aria-hidden="true" />
                        </button>
                      </form>
                    </div>
                  </Show>
                )}
              </For>
            </SectionPanel>
            <SectionPanel id="account-settings" title="Account">
              <div class="account-settings-button-row">
                <button
                  class="delete-button"
                  disabled={busy()}
                  onClick={() => {
                    setError("");
                    accountDialog?.showModal();
                  }}
                >
                  Delete my account
                </button>
                <button
                  class="delete-button"
                  disabled={busy()}
                  onClick={() => {
                    setError("");
                    filesDialog?.showModal();
                  }}
                >
                  Delete all files
                </button>
              </div>
            </SectionPanel>
            <NativeConfirmation
              id="confirm-account"
              ref={(element) => {
                accountDialog = element;
              }}
              title="Confirm deletion"
              message="Delete your account and all its data permanently? This cannot be undone."
              busy={busy()}
              error={error()}
              onConfirm={() => remove(true)}
            />
            <NativeConfirmation
              id="confirm-files"
              ref={(element) => {
                filesDialog = element;
              }}
              title="Confirm deletion"
              message="Delete all your files permanently? This cannot be undone."
              busy={busy()}
              error={error()}
              onConfirm={() => remove(false)}
            />
          </>
        )}
      </Show>
      <ConfirmDialog />
      <HostlingToast />
    </>
  );
}

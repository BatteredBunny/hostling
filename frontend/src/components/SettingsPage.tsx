import "./SettingsPage.css";
import { LoadError } from "./LoadError";
import { WarningCallout } from "./WarningCallout";
import { SectionPanel } from "./SectionPanel";
import { For, Show, createSignal, onMount } from "solid-js";
import {
  getSettings,
  deleteAccount,
  deleteAllFiles,
  unlinkProvider,
  type SettingsOutput,
  type UnlinkProviderData,
} from "../generated";
import { toastAuthError } from "../authErrors";
import { trackPageLoad } from "../loading";
import { apiData } from "../client";
import { showToast } from "../toast";
import { NativeConfirmation } from "./NativeConfirmation";
import Unlink from "lucide-solid/icons/unlink";
import { ProviderButton, ProviderIcon, providerLabel } from "./ProviderAction";
import { confirmDialog } from "../store";
import { useFrontendContext } from "../context";

export function SettingsPage() {
  const { error: contextError, retry: retryContext } = useFrontendContext();
  const [data, setData] = createSignal<SettingsOutput>();
  const [loadError, setLoadError] = createSignal(false);
  async function load() {
    setLoadError(false);
    try {
      setData(await apiData(getSettings()));
    } catch {
      setLoadError(true);
    }
  }
  onMount(() => void trackPageLoad(load()));
  toastAuthError();

  async function unlink(provider: UnlinkProviderData["body"]["provider"]) {
    const name = providerLabel(provider);
    const confirmed = await confirmDialog({
      title: `Unlink ${name}?`,
      message: `You won't be able to log in with ${name} anymore.`,
      confirmLabel: "Unlink",
      danger: true,
    });
    if (!confirmed) return;
    try {
      await apiData(unlinkProvider({ body: { provider } }));
    } catch (failure) {
      showToast(
        failure instanceof Error
          ? failure.message
          : `Could not unlink ${name}.`,
        "error",
      );
      return;
    }
    showToast(`Unlinked ${name}.`);
    await load();
  }
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
      <Show when={loadError() || contextError()}>
        <LoadError
          message="Could not load settings."
          onRetry={() => {
            retryContext();
            void trackPageLoad(load());
          }}
        />
      </Show>
      <Show when={data()}>
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
                      <button
                        type="button"
                        class="unlink-button"
                        title={`Unlink ${providerLabel(provider.Name)}`}
                        aria-label={`Unlink ${providerLabel(provider.Name)}`}
                        onClick={() =>
                          unlink(
                            provider.Name as UnlinkProviderData["body"]["provider"],
                          )
                        }
                      >
                        <Unlink class="lucide-icon" aria-hidden="true" />
                      </button>
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
    </>
  );
}

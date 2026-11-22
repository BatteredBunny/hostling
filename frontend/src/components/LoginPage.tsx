import "./LoginPage.css";
import { LoadError } from "./LoadError";
import { WarningCallout } from "./WarningCallout";
import { SectionPanel } from "./SectionPanel";
import { For, Show } from "solid-js";
import { useFrontendContext } from "../context";
import { toastAuthError } from "../authErrors";
import { ProviderLink, providerLabel } from "./ProviderAction";

export function LoginPage() {
  const { context, error, retry } = useFrontendContext();
  toastAuthError();
  return (
    <div id="login-page">
      <SectionPanel class="login-panel" title="Welcome back">
        <Show when={error()}>
          <LoadError message={error()} onRetry={retry} />
        </Show>
        <Show when={context()}>
          {(info) => (
            <Show
              when={info().providers?.length}
              fallback={
                <WarningCallout>
                  <p>
                    <strong>Server configuration error:</strong> No
                    authentication providers are configured. Please contact the
                    server administrator to set up {info().branding} correctly.
                  </p>
                </WarningCallout>
              }
            >
              <p class="login-description">
                Choose a linked account to log in.
              </p>
              <div class="login-providers">
                <For each={info().providers}>
                  {(provider) => (
                    <ProviderLink
                      provider={provider}
                      href={`/api/auth/login/${provider}`}
                    >
                      Log in with {providerLabel(provider)}
                    </ProviderLink>
                  )}
                </For>
              </div>
            </Show>
          )}
        </Show>
        <p class="login-register">
          Have an invitation code? <a href="/register">Create an account</a>
        </p>
      </SectionPanel>
    </div>
  );
}

import { providerLabel } from "./components/ProviderAction";
import { showToast } from "./toast";

// Matches the codes in internal/auth.go.
const messages: Record<string, (provider: string) => string> = {
  provider_unavailable: (provider) =>
    `${provider} is unavailable right now. Please try again later.`,
  login_failed: (provider) =>
    `Logging in with ${provider} didn't work. Please try again.`,
  account_not_found: (provider) =>
    `No account is linked to this ${provider} account. Create one with an invitation code, then link ${provider} in Settings.`,
  link_failed: (provider) =>
    `Linking ${provider} didn't work. Please try again.`,
  identity_taken: (provider) =>
    `This ${provider} account is already linked to another user.`,
};

function takeAuthError(): string | null {
  const params = new URLSearchParams(window.location.search);
  const code = params.get("error");
  if (!code) return null;
  const provider =
    providerLabel(params.get("provider") ?? "") || "the provider";
  params.delete("error");
  params.delete("provider");
  const query = params.toString();
  history.replaceState(
    history.state,
    "",
    window.location.pathname +
      (query ? `?${query}` : "") +
      window.location.hash,
  );
  return (
    messages[code]?.(provider) ??
    `Something went wrong with ${provider}. Please try again.`
  );
}

export function toastAuthError() {
  const message = takeAuthError();
  if (message) showToast(message, "error");
}

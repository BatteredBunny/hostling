import "./RegistrationPage.css";
import { SectionPanel } from "./SectionPanel";
import { createSignal } from "solid-js";
import { useFrontendContext } from "../context";
import { register } from "../generated";
import { apiData } from "../client";
import { showToast } from "../toast";
import { Input } from "./Input";
import UserPlusIcon from "lucide-solid/icons/user-plus";

// Invite links carry the code; fill it in and drop it from the address bar.
function takeInviteCode(): string {
  const params = new URLSearchParams(window.location.search);
  const code = params.get("code") ?? "";
  if (code) {
    params.delete("code");
    const query = params.toString();
    history.replaceState(
      history.state,
      "",
      window.location.pathname + (query ? `?${query}` : ""),
    );
  }
  return code;
}

export function RegistrationPage() {
  const { context } = useFrontendContext();
  const [code, setCode] = createSignal(takeInviteCode());
  const [busy, setBusy] = createSignal(false);
  async function submit(event: SubmitEvent) {
    event.preventDefault();
    if (busy()) return;
    setBusy(true);
    try {
      await apiData(register({ body: { code: code().trim() } }));
      // A fresh document picks up the new session everywhere.
      window.location.assign("/gallery");
    } catch (failure) {
      setCode("");
      showToast(
        failure instanceof Error
          ? failure.message
          : "Could not create your account.",
        "error",
      );
      setBusy(false);
    }
  }
  return (
    <div id="registration-page">
      <SectionPanel class="registration-panel" title="Create your account">
        <p class="registration-description">
          Enter your invitation code to join {context()?.branding ?? "…"}.
        </p>
        <form class="registration-form" onSubmit={submit}>
          <Input
            label="Invitation code"
            id="invitation-code"
            name="code"
            required
            autocomplete="off"
            autocapitalize="none"
            spellcheck={false}
            placeholder="Enter your invitation code"
            value={code()}
            onInput={(event) => setCode(event.currentTarget.value)}
            disabled={busy()}
          />
          <button class="registration-submit" type="submit" disabled={busy()}>
            <UserPlusIcon class="lucide-icon" aria-hidden="true" />
            Create account
          </button>
        </form>
        <p class="registration-login">
          Already have an account? <a href="/login">Log in</a>
        </p>
      </SectionPanel>
    </div>
  );
}

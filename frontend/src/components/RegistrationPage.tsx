import "./RegistrationPage.css";
import { SectionPanel } from "./SectionPanel";
import { createSignal, Show } from "solid-js";
import { useFrontendContext } from "../context";
import { register } from "../generated";
import { apiData } from "../client";
import { Input } from "./Input";
import { UserPlus as UserPlusIcon } from "lucide-solid";

export function RegistrationPage() {
  const { context } = useFrontendContext();
  const [code, setCode] = createSignal("");
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal("");
  async function submit(event: SubmitEvent) {
    event.preventDefault();
    if (busy()) return;
    setBusy(true);
    setError("");
    try {
      const result = await apiData(register({ body: { code: code().trim() } }));
      window.location.assign(result.redirect);
    } catch (failure) {
      setError(
        failure instanceof Error
          ? failure.message
          : "Could not create your account.",
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
          <Show when={error()}>
            <p role="alert">{error()}</p>
          </Show>
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

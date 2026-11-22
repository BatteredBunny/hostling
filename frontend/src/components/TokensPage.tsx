import { SectionPanel } from "./SectionPanel";
import { LoadError } from "./LoadError";
import { For, Show, createSignal, onCleanup, onMount } from "solid-js";
import { SecretDialog } from "./SecretDialog";
import LoaderCircleIcon from "lucide-solid/icons/loader-circle";
import PlusIcon from "lucide-solid/icons/plus";
import Trash2Icon from "lucide-solid/icons/trash";
import CheckIcon from "lucide-solid/icons/check";
import CopyIcon from "lucide-solid/icons/copy";
import LinkIcon from "lucide-solid/icons/link";
import { Input } from "./Input";
import { showToast } from "../toast";
import { trackPageLoad } from "../loading";
import {
  listUploadTokens,
  listInviteCodes,
  createUploadToken,
  deleteUploadToken,
  deleteInviteCode,
  type UiUploadToken as UploadToken,
  type CreatedUploadToken,
  type InviteCodeApiOutput as InviteCode,
} from "../generated";
import { apiData } from "../client";
import { UploadTokenNameSchema } from "../generated/schemas.gen";
import {
  copyText,
  countLabel,
  cx,
  formatTimeDate,
  relativeTime,
} from "../utils";
import { confirmDialog } from "../store";
import "./TokensPage.css";

export function TokensPage() {
  const [tokens, setTokens] = createSignal<UploadToken[]>([]);
  const [invites, setInvites] = createSignal<InviteCode[]>([]);
  const [loaded, setLoaded] = createSignal(false);
  const [loading, setLoading] = createSignal(false);
  const [loadError, setLoadError] = createSignal("");
  const [error, setError] = createSignal("");
  const [name, setName] = createSignal("");
  const [creating, setCreating] = createSignal(false);
  const [createdToken, setCreatedToken] = createSignal<CreatedUploadToken>();
  let nameInput: HTMLInputElement | undefined;

  const load = async () => {
    setLoading(true);
    setLoadError("");
    try {
      const [uploadTokens, inviteCodes] = await Promise.all([
        apiData(listUploadTokens()),
        apiData(listInviteCodes()),
      ]);
      setTokens(uploadTokens ?? []);
      setInvites(inviteCodes ?? []);
      setLoaded(true);
    } catch {
      setLoadError("Could not load your tokens. Please try again.");
    } finally {
      setLoading(false);
    }
  };
  onMount(() => void trackPageLoad(load()));

  const create = async (event: SubmitEvent) => {
    event.preventDefault();
    if (creating() || createdToken()) return;
    const nickname = name().trim();
    if (!nickname) {
      nameInput?.setCustomValidity("Give this token a name.");
      nameInput?.reportValidity();
      return;
    }
    if (Array.from(nickname).length > UploadTokenNameSchema.maxLength) {
      nameInput?.setCustomValidity(
        `Use at most ${UploadTokenNameSchema.maxLength} characters.`,
      );
      nameInput?.reportValidity();
      return;
    }
    setCreating(true);
    setError("");
    try {
      const token = await apiData(createUploadToken({ body: { nickname } }));
      const { token: secret, ...metadata } = token;
      setTokens((previous) => [metadata, ...previous]);
      setCreatedToken({ ...metadata, token: secret });
      if (name().trim() === nickname) setName("");
    } catch (err) {
      setError((err as Error).message || "Could not create the token.");
    } finally {
      setCreating(false);
    }
  };

  const removeToken = async (token: UploadToken) => {
    await apiData(deleteUploadToken({ body: { id: token.id } }));
    setTokens((previous) => previous.filter((item) => item.id !== token.id));
    if (createdToken()?.id === token.id) setCreatedToken(undefined);
    showToast(`Deleted “${token.name || "Unnamed token"}”.`);
    nameInput?.focus();
  };

  const removeInvite = async (invite: InviteCode) => {
    await apiData(deleteInviteCode({ body: { invite_code: invite.code } }));
    setInvites((previous) =>
      previous.filter((item) => item.code !== invite.code),
    );
    showToast("Invite code deleted.");
  };

  return (
    <>
      <SectionPanel id="upload-tokens" title="Upload tokens">
        <p class="tokens-description">
          Give each app or script its own named token to upload files to your
          account.
        </p>
        <Show when={loading()}>
          <p role="status">Loading tokens…</p>
        </Show>
        <Show when={loadError()}>
          <LoadError message={loadError()} onRetry={load} busy={loading()} />
        </Show>
        <form id="create-upload-token" onSubmit={create}>
          <Input
            fieldClass="token-name-field"
            label="Token name"
            ref={nameInput}
            id="token-name"
            name="nickname"
            type="text"
            required
            placeholder="e.g. Screenshot uploader"
            autocomplete="off"
            value={name()}
            onInput={(event) => {
              event.currentTarget.setCustomValidity("");
              setName(event.currentTarget.value);
            }}
          />
          <button
            class="create-button"
            type="submit"
            disabled={!loaded() || creating() || !!createdToken()}
            aria-label={
              creating() ? "Creating upload token" : "Create upload token"
            }
            title="Create upload token"
          >
            <Show
              when={creating()}
              fallback={<PlusIcon class="lucide-icon" aria-hidden="true" />}
            >
              <LoaderCircleIcon
                class="lucide-icon spinning"
                aria-hidden="true"
              />
            </Show>
          </button>
        </form>
        <Show when={createdToken()}>
          {(token) => (
            <SecretDialog
              title="Save your upload token"
              notice="This is your upload token. Please save it as you won't be able to view it again."
              noun="token"
              value={token().token}
              onDismiss={() => {
                setCreatedToken(undefined);
                nameInput?.focus();
              }}
            />
          )}
        </Show>
        <Show when={error()}>
          <p id="tokens-error" role="alert">
            {error()}
          </p>
        </Show>
        <Show when={loaded()}>
          <div class="tokens-list-heading">
            <h3>Tokens</h3>
          </div>
          <div class="upload-tokens-list">
            <For
              each={tokens()}
              fallback={<p id="tokens-empty">No upload tokens yet.</p>}
            >
              {(token) => (
                <TokenEntry
                  token={token}
                  onDelete={() => removeToken(token)}
                  onError={setError}
                />
              )}
            </For>
          </div>
        </Show>
      </SectionPanel>
      <Show when={invites().length > 0}>
        <SectionPanel id="invite-codes" title="Invite codes">
          <p class="tokens-description">
            Share a code or its link to let someone create an account.
          </p>
          <div class="upload-tokens-list">
            <For each={invites()}>
              {(invite) => (
                <InviteEntry
                  invite={invite}
                  onDelete={() => removeInvite(invite)}
                />
              )}
            </For>
          </div>
        </SectionPanel>
      </Show>
    </>
  );
}

function TokenEntry(props: {
  token: UploadToken;
  onDelete: () => Promise<void>;
  onError: (message: string) => void;
}) {
  const [deleting, setDeleting] = createSignal(false);
  const remove = async () => {
    if (deleting()) return;
    setDeleting(true);
    props.onError("");
    try {
      await props.onDelete();
    } catch (err) {
      props.onError((err as Error).message || "Could not delete the token.");
      setDeleting(false);
    }
  };

  return (
    <article
      class="upload-token-entry"
      aria-label={props.token.name || "Unnamed token"}
    >
      <div class="token-info">
        <h4 class="token-name">{props.token.name || "Unnamed token"}</h4>
        <span class="token-last-used">
          Created:{" "}
          <span title={formatTimeDate(props.token.created_at)}>
            {relativeTime(props.token.created_at)}
          </span>{" "}
          · Last used:{" "}
          <span
            title={
              props.token.last_used
                ? formatTimeDate(props.token.last_used)
                : undefined
            }
          >
            {props.token.last_used
              ? relativeTime(props.token.last_used)
              : "Never"}
          </span>
        </span>
      </div>
      <div class="token-actions">
        <button
          type="button"
          class="token-action token-delete"
          onClick={remove}
          disabled={deleting()}
          title="Delete token"
          aria-label="Delete token"
        >
          <Show
            when={deleting()}
            fallback={<Trash2Icon class="lucide-icon" aria-hidden="true" />}
          >
            <LoaderCircleIcon class="lucide-icon spinning" aria-hidden="true" />
          </Show>
        </button>
      </div>
    </article>
  );
}

function InviteEntry(props: {
  invite: InviteCode;
  onDelete: () => Promise<void>;
}) {
  const [deleting, setDeleting] = createSignal(false);
  const [copied, setCopied] = createSignal<"code" | "link" | null>(null);
  let copiedTimeout: ReturnType<typeof setTimeout> | undefined;
  onCleanup(() => clearTimeout(copiedTimeout));

  const link = () =>
    new URL(
      `/register?code=${encodeURIComponent(props.invite.code)}`,
      window.location.origin,
    ).href;

  const copy = async (what: "code" | "link") => {
    try {
      await copyText(what === "code" ? props.invite.code : link());
    } catch {
      showToast(`Could not copy the invite ${what}.`, "error");
      return;
    }
    showToast(`Invite ${what} copied.`);
    setCopied(what);
    clearTimeout(copiedTimeout);
    copiedTimeout = setTimeout(() => setCopied(null), 2000);
  };

  const remove = async () => {
    if (deleting()) return;
    const confirmed = await confirmDialog({
      title: "Delete invite code?",
      message: `"${props.invite.code}" will stop working, including links already shared.`,
      confirmLabel: "Delete",
      danger: true,
    });
    if (!confirmed) return;
    setDeleting(true);
    try {
      await props.onDelete();
    } catch (err) {
      showToast(
        (err as Error).message || "Could not delete the invite code.",
        "error",
      );
      setDeleting(false);
    }
  };

  return (
    <article
      class="upload-token-entry"
      aria-label={`Invite code ${props.invite.code}`}
    >
      <div class="token-info">
        <code class="invite-code">{props.invite.code}</code>
        <span class="token-last-used">
          {countLabel(props.invite.uses, "use")} left ·{" "}
          <span title={formatTimeDate(props.invite.expiry_date)}>
            Expires {relativeTime(props.invite.expiry_date)}
          </span>
        </span>
      </div>
      <div class="token-actions">
        <button
          type="button"
          class={cx("token-action token-copy", { copied: copied() === "link" })}
          title="Copy invite link"
          aria-label="Copy invite link"
          onClick={() => copy("link")}
        >
          <Show
            when={copied() === "link"}
            fallback={<LinkIcon class="lucide-icon" aria-hidden="true" />}
          >
            <CheckIcon class="lucide-icon" aria-hidden="true" />
          </Show>
        </button>
        <button
          type="button"
          class={cx("token-action token-copy", { copied: copied() === "code" })}
          title="Copy invite code"
          aria-label="Copy invite code"
          onClick={() => copy("code")}
        >
          <Show
            when={copied() === "code"}
            fallback={<CopyIcon class="lucide-icon" aria-hidden="true" />}
          >
            <CheckIcon class="lucide-icon" aria-hidden="true" />
          </Show>
        </button>
        <button
          type="button"
          class="token-action token-delete"
          title="Delete invite code"
          aria-label="Delete invite code"
          disabled={deleting()}
          onClick={remove}
        >
          <Show
            when={deleting()}
            fallback={<Trash2Icon class="lucide-icon" aria-hidden="true" />}
          >
            <LoaderCircleIcon class="lucide-icon spinning" aria-hidden="true" />
          </Show>
        </button>
      </div>
    </article>
  );
}

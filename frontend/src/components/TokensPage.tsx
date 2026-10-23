import { SectionPanel } from "./SectionPanel";
import { For, Show, createSignal, onCleanup, onMount } from "solid-js";
import { NativeDialog } from "./NativeDialog";
import { copyText } from "../utils";
import {
  Check as CheckIcon,
  Copy as CopyIcon,
  LoaderCircle as LoaderCircleIcon,
  Plus as PlusIcon,
  Trash2 as Trash2Icon,
} from "lucide-solid";
import { Dynamic } from "solid-js/web";
import { Input } from "./Input";
import { HostlingToast } from "./HostlingToast";
import { showToast } from "../toast";
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
import { formatTimeDate, relativeTime } from "../utils";
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
  onMount(load);

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
    nameInput?.focus();
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
          <p role="alert">{loadError()}</p>
          <button type="button" onClick={load} disabled={loading()}>
            Try again
          </button>
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
            <Dynamic
              component={creating() ? LoaderCircleIcon : PlusIcon}
              class="lucide-icon"
              aria-hidden="true"
            />
          </button>
        </form>
        <Show when={createdToken()} keyed>
          {(token) => (
            <CreatedTokenPanel
              token={token.token}
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
          <p>Share these codes to invite someone to create an account.</p>
          <div class="codes">
            <For each={invites()}>
              {(invite) => (
                <InviteEntry
                  invite={invite}
                  onDelete={() => removeInvite(invite)}
                  onError={setError}
                />
              )}
            </For>
          </div>
        </SectionPanel>
      </Show>
      <HostlingToast />
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
    <article class="upload-token-entry">
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
          <Dynamic
            component={deleting() ? LoaderCircleIcon : Trash2Icon}
            class="lucide-icon"
            aria-hidden="true"
          />
        </button>
      </div>
    </article>
  );
}

function CreatedTokenPanel(props: { token: string; onDismiss: () => void }) {
  const [copied, setCopied] = createSignal(false);
  const [copyError, setCopyError] = createSignal("");
  let dialog: HTMLDialogElement | undefined;
  let copyButton: HTMLButtonElement | undefined;
  let copyTimeout: ReturnType<typeof setTimeout> | undefined;
  onMount(() =>
    queueMicrotask(() => copyButton?.isConnected && copyButton.focus()),
  );
  onCleanup(() => clearTimeout(copyTimeout));

  const copy = async () => {
    setCopyError("");
    try {
      await copyText(props.token, dialog ?? document.body);
      setCopied(true);
      showToast("Token copied.");
      clearTimeout(copyTimeout);
      copyTimeout = setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopyError(
        "Could not copy the token. You can select and copy it below.",
      );
    }
  };
  return (
    <NativeDialog
      open
      ref={(element) => {
        dialog = element;
      }}
      class="created-token-modal"
      labelledBy="created-token-title"
      describedBy="created-token-notice"
      onDismiss={props.onDismiss}
    >
      <header class="created-token-titlebar">
        <h2 id="created-token-title">Save your upload token</h2>
      </header>
      <div class="created-token-contents">
        <p id="created-token-notice">
          This is your upload token. Please save it as you won't be able to view
          it again.
        </p>
        <pre class="created-token-code">
          <code>{props.token}</code>
        </pre>
        <Show when={copyError()}>
          <p class="created-token-error" role="alert">
            {copyError()}
          </p>
        </Show>
      </div>
      <footer class="created-token-footer">
        <button
          ref={copyButton}
          type="button"
          class="created-token-copy"
          onClick={copy}
        >
          <Dynamic
            component={copied() ? CheckIcon : CopyIcon}
            class="lucide-icon"
            aria-hidden="true"
          />
          {copied() ? "Copied" : "Copy token"}
        </button>
        <button
          type="button"
          class="created-token-dismiss"
          onClick={() => props.onDismiss()}
        >
          Dismiss
        </button>
      </footer>
      <HostlingToast inline />
    </NativeDialog>
  );
}

function InviteEntry(props: {
  invite: InviteCode;
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
      props.onError(
        (err as Error).message || "Could not delete the invite code.",
      );
      setDeleting(false);
    }
  };
  return (
    <div class="code">
      <code class="code-text">{props.invite.code}</code>
      <div class="info">
        <span class="uses-left">Uses left: {props.invite.uses}</span>
        <span
          class="invite-code-expires"
          title={formatTimeDate(props.invite.expiry_date)}
        >
          Expires {relativeTime(props.invite.expiry_date)}
        </span>
        <button
          type="button"
          class="delete-button"
          disabled={deleting()}
          onClick={remove}
        >
          {deleting() ? "Deleting…" : "Delete"}
        </button>
      </div>
    </div>
  );
}

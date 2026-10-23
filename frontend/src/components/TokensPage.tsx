import { For, Show, createSignal, onCleanup, onMount } from 'solid-js';
import { Portal } from 'solid-js/web';
import { Icon } from './Icon';
import { HostlingToast } from './HostlingToast';
import { showToast, toast } from '../toast';
import {
  fetchTokens, createUploadToken, deleteUploadToken, deleteInviteCode,
  type UploadToken, type CreatedUploadToken, type InviteCode,
} from '../tokensApi';
import { formatTimeDate, relativeTime } from '../utils';
import './TokensPage.css';

export function TokensPage() {
  const [tokens, setTokens] = createSignal<UploadToken[]>([]);
  const [invites, setInvites] = createSignal<InviteCode[]>([]);
  const [loaded, setLoaded] = createSignal(false);
  const [loading, setLoading] = createSignal(false);
  const [loadError, setLoadError] = createSignal('');
  const [error, setError] = createSignal('');
  const [name, setName] = createSignal('');
  const [creating, setCreating] = createSignal(false);
  const [createdToken, setCreatedToken] = createSignal<CreatedUploadToken>();
  let nameInput: HTMLInputElement | undefined;

  const load = async () => {
    setLoading(true);
    setLoadError('');
    try {
      const data = await fetchTokens();
      setTokens(data.upload_tokens);
      setInvites(data.invite_codes);
      setLoaded(true);
    } catch {
      setLoadError('Could not load your tokens. Please try again.');
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
      nameInput?.setCustomValidity('Give this token a name.');
      nameInput?.reportValidity();
      return;
    }
    setCreating(true);
    setError('');
    try {
      const token = await createUploadToken(nickname);
      const { token: secret, ...metadata } = token;
      setTokens((previous) => [metadata, ...previous]);
      setCreatedToken({ ...metadata, token: secret });
      if (name().trim() === nickname) setName('');
    } catch (err) {
      setError((err as Error).message || 'Could not create the token.');
    } finally {
      setCreating(false);
    }
  };

  const removeToken = async (token: UploadToken) => {
    await deleteUploadToken(token.id);
    setTokens((previous) => previous.filter((item) => item.id !== token.id));
    if (createdToken()?.id === token.id) setCreatedToken(undefined);
    showToast(`Deleted “${token.name || 'Unnamed token'}”.`);
    nameInput?.focus();
  };

  const removeInvite = async (invite: InviteCode) => {
    await deleteInviteCode(invite.code);
    setInvites((previous) => previous.filter((item) => item.code !== invite.code));
    showToast('Invite code deleted.');
    nameInput?.focus();
  };

  return (
    <>
      <setting-group id="upload-tokens">
        <div class="setting-group-header"><h2>Upload tokens</h2></div>
        <div class="setting-group-body">
          <p class="tokens-description">Give each app or script its own named token to upload files to your account.</p>
          <Show when={loading()}><p role="status">Loading tokens…</p></Show>
          <Show when={loadError()}>
            <p role="alert">{loadError()}</p>
            <button type="button" onClick={load} disabled={loading()}>Try again</button>
          </Show>
          <Show when={loaded()}>
            <form id="create-upload-token" onSubmit={create}>
              <div class="token-name-field">
                <label for="token-name">Token name</label>
                <input
                  ref={nameInput} id="token-name" name="nickname" type="text" required maxLength={64}
                  placeholder="e.g. Screenshot uploader" autocomplete="off"
                  value={name()}
                  onInput={(event) => {
                    event.currentTarget.setCustomValidity('');
                    setName(event.currentTarget.value);
                  }}
                />
              </div>
              <button class="create-button" type="submit" disabled={creating() || !!createdToken()} aria-label={creating() ? 'Creating upload token' : 'Create upload token'} title="Create upload token">
                <Icon name={creating() ? 'loader-circle' : 'plus'} />
              </button>
            </form>
          </Show>
          <Show when={createdToken()} keyed>
            {(token) => <CreatedTokenPanel token={token.token} onDismiss={() => { setCreatedToken(undefined); nameInput?.focus(); }} />}
          </Show>
          <Show when={error()}><p id="tokens-error" role="alert">{error()}</p></Show>
          <Show when={loaded()}>
            <div class="tokens-list-heading"><h3>Tokens</h3></div>
            <div class="upload-tokens-list">
              <For each={tokens()} fallback={<p id="tokens-empty">No upload tokens yet.</p>}>
                {(token) => (
                  <TokenEntry token={token} onDelete={() => removeToken(token)} onError={setError} />
                )}
              </For>
            </div>
          </Show>
        </div>
      </setting-group>
      <Show when={invites().length > 0}>
        <setting-group id="invite-codes">
          <div class="setting-group-header"><h2>Invite codes</h2></div>
          <div class="setting-group-body">
            <p>Share these codes to invite someone to create an account.</p>
            <div class="codes">
              <For each={invites()}>
                {(invite) => (
                  <InviteEntry invite={invite} onDelete={() => removeInvite(invite)} onError={setError} />
                )}
              </For>
            </div>
          </div>
        </setting-group>
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
    props.onError('');
    try {
      await props.onDelete();
    } catch (err) {
      props.onError((err as Error).message || 'Could not delete the token.');
      setDeleting(false);
    }
  };

  return (
    <article class="upload-token-entry">
      <div class="token-info">
        <h4 class="token-name">{props.token.name || 'Unnamed token'}</h4>
        <span class="token-last-used">Created: <span title={formatTimeDate(props.token.created_at)}>{relativeTime(props.token.created_at)}</span> · Last used: <span title={props.token.last_used ? formatTimeDate(props.token.last_used) : undefined}>
          {props.token.last_used ? relativeTime(props.token.last_used) : 'Never'}
        </span></span>
      </div>
      <div class="token-actions">
        <button type="button" class="token-action token-delete" onClick={remove} disabled={deleting()} title="Delete token" aria-label="Delete token">
          <Icon name={deleting() ? 'loader-circle' : 'trash-2'} />
        </button>
      </div>
    </article>
  );
}

function CreatedTokenPanel(props: { token: string; onDismiss: () => void }) {
  const [copied, setCopied] = createSignal(false);
  const [copyError, setCopyError] = createSignal('');
  let dialog: HTMLDialogElement | undefined;
  let copyButton: HTMLButtonElement | undefined;
  let dismissButton: HTMLButtonElement | undefined;
  let copyTimeout: ReturnType<typeof setTimeout> | undefined;
  let backdropPressed = false;

  onMount(() => {
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const wasScrollLocked = document.body.classList.contains('no-scroll');
    dialog?.showModal();
    queueMicrotask(() => copyButton?.isConnected && copyButton.focus());
    document.body.classList.add('no-scroll');
    onCleanup(() => {
      clearTimeout(copyTimeout);
      dialog?.close();
      if (!wasScrollLocked) document.body.classList.remove('no-scroll');
      if (previousFocus?.isConnected) previousFocus.focus();
    });
  });

  const outsideDialog = (event: MouseEvent) => {
    if (event.target !== dialog || !dialog) return false;
    const bounds = dialog.getBoundingClientRect();
    return event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom;
  };
  const copy = async () => {
    setCopyError('');
    try {
      await copyText(props.token, dialog ?? document.body);
      setCopied(true);
      showToast('Token copied.');
      clearTimeout(copyTimeout);
      copyTimeout = setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopyError('Could not copy the token. You can select and copy it below.');
    }
  };
  return (
    <Portal>
      <dialog
        ref={dialog} class="created-token-modal" aria-modal="true"
        aria-labelledby="created-token-title" aria-describedby="created-token-notice"
        onCancel={(event) => { event.preventDefault(); props.onDismiss(); }}
        onKeyDown={(event) => {
          if (event.key !== 'Tab') return;
          if (event.shiftKey && document.activeElement === copyButton) {
            event.preventDefault();
            dismissButton?.focus();
          } else if (!event.shiftKey && document.activeElement === dismissButton) {
            event.preventDefault();
            copyButton?.focus();
          }
        }}
        onMouseDown={(event) => { backdropPressed = outsideDialog(event); }}
        onClick={(event) => { if (backdropPressed && outsideDialog(event)) props.onDismiss(); backdropPressed = false; }}
      >
        <header class="created-token-titlebar"><h2 id="created-token-title">Save your upload token</h2></header>
        <div class="created-token-contents">
          <p id="created-token-notice">This is your upload token. Please save it as you won't be able to view it again.</p>
          <pre class="created-token-code"><code>{props.token}</code></pre>
          <Show when={copyError()}><p class="created-token-error" role="alert">{copyError()}</p></Show>
        </div>
        <footer class="created-token-footer">
          <button ref={copyButton} type="button" class="created-token-copy" onClick={copy}>
            <Icon name={copied() ? 'check' : 'copy'} />{copied() ? 'Copied' : 'Copy token'}
          </button>
          <button ref={dismissButton} type="button" class="created-token-dismiss" onClick={() => props.onDismiss()}>Dismiss</button>
        </footer>
        <Show when={toast()} keyed>{(notification) => <div class="hostling-toast" role="status">{notification.message}</div>}</Show>
      </dialog>
    </Portal>
  );
}

function InviteEntry(props: { invite: InviteCode; onDelete: () => Promise<void>; onError: (message: string) => void }) {
  const [deleting, setDeleting] = createSignal(false);
  const remove = async () => {
    if (deleting()) return;
    setDeleting(true);
    props.onError('');
    try {
      await props.onDelete();
    } catch (err) {
      props.onError((err as Error).message || 'Could not delete the invite code.');
      setDeleting(false);
    }
  };
  return (
    <div class="code">
      <code class="code-text">{props.invite.code}</code>
      <div class="info">
        <span class="uses-left">Uses left: {props.invite.uses}</span>
        <span class="invite-code-expires" title={formatTimeDate(props.invite.expiry_date)}>Expires {relativeTime(props.invite.expiry_date)}</span>
        <button type="button" class="delete-button" disabled={deleting()} onClick={remove}>{deleting() ? 'Deleting…' : 'Delete'}</button>
      </div>
    </div>
  );
}

async function copyText(text: string, container: HTMLElement) {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    const previousFocus = document.activeElement;
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    container.appendChild(textarea);
    try {
      textarea.select();
      if (!document.execCommand('copy')) throw new Error('Copy failed');
    } finally {
      textarea.remove();
      if (previousFocus instanceof HTMLElement) previousFocus.focus();
    }
  }
}

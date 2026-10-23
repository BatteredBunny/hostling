import { For, Show, createSignal, onCleanup, onMount } from 'solid-js';
import { Icon } from './Icon';
import { HostlingToast } from './HostlingToast';
import { showToast } from '../toast';
import {
  fetchTokens, createUploadToken, deleteUploadToken, deleteInviteCode,
  type UploadToken, type InviteCode,
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
    if (creating()) return;
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
      setTokens((previous) => [token, ...previous]);
      if (name().trim() === nickname) setName('');
      nameInput?.focus();
    } catch (err) {
      setError((err as Error).message || 'Could not create the token.');
    } finally {
      setCreating(false);
    }
  };

  const removeToken = async (token: UploadToken) => {
    await deleteUploadToken(token.token);
    setTokens((previous) => previous.filter((item) => item.token !== token.token));
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
                  placeholder="e.g. Screenshot uploader" autocomplete="off" aria-describedby="token-name-hint"
                  value={name()}
                  onInput={(event) => {
                    event.currentTarget.setCustomValidity('');
                    setName(event.currentTarget.value);
                  }}
                />
              </div>
              <button class="create-button" type="submit" disabled={creating()} aria-label={creating() ? 'Creating upload token' : 'Create upload token'} title="Create upload token">
                <Icon name={creating() ? 'loader-circle' : 'plus'} />
              </button>
            </form>
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
  const [revealed, setRevealed] = createSignal(false);
  const [copied, setCopied] = createSignal(false);
  const [deleting, setDeleting] = createSignal(false);
  let copyTimeout: ReturnType<typeof setTimeout> | undefined;
  const hide = () => setRevealed(false);
  onMount(() => {
    window.addEventListener('blur', hide);
    document.addEventListener('visibilitychange', hide);
  });
  onCleanup(() => {
    clearTimeout(copyTimeout);
    window.removeEventListener('blur', hide);
    document.removeEventListener('visibilitychange', hide);
  });

  const copy = async () => {
    props.onError('');
    try {
      await copyText(props.token.token);
      setCopied(true);
      showToast('Token copied.');
      clearTimeout(copyTimeout);
      copyTimeout = setTimeout(() => setCopied(false), 2000);
    } catch {
      props.onError('Could not copy the token.');
    }
  };
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
        <span class="token-last-used">Last used: <span title={props.token.last_used ? formatTimeDate(props.token.last_used) : undefined}>
          {props.token.last_used ? relativeTime(props.token.last_used) : 'Never'}
        </span></span>
      </div>
      <div class="token-secret-row">
        <code class="token-secret" aria-label={revealed() ? undefined : 'Hidden upload token'}>
          {revealed() ? props.token.token : '••••••••••••••••••••••••'}
        </code>
        <div class="token-actions">
          <button
            type="button" class="token-action token-reveal" title="Hold to reveal" aria-label="Hold to reveal token" aria-pressed={revealed()}
            onPointerDown={(event) => {
              if (event.button !== 0) return;
              event.currentTarget.setPointerCapture(event.pointerId);
              setRevealed(true);
            }}
            onPointerUp={hide} onPointerCancel={hide} onLostPointerCapture={hide} onPointerLeave={hide} onBlur={hide}
            onContextMenu={(event) => event.preventDefault()}
            onKeyDown={(event) => {
              if (event.key === ' ' || event.key === 'Enter') {
                event.preventDefault();
                setRevealed(true);
              }
            }}
            onKeyUp={(event) => {
              if (event.key === ' ' || event.key === 'Enter') {
                event.preventDefault();
                hide();
              }
            }}
          ><Icon name={revealed() ? 'eye-off' : 'eye'} /></button>
          <button type="button" class="token-action token-copy" classList={{ copied: copied() }} onClick={copy} title={copied() ? 'Copied' : 'Copy token'} aria-label={copied() ? 'Token copied' : 'Copy token'}>
            <Icon name={copied() ? 'check' : 'copy'} />
          </button>
          <button type="button" class="token-action token-delete" onClick={remove} disabled={deleting()} title="Delete token" aria-label="Delete token">
            <Icon name={deleting() ? 'loader-circle' : 'trash-2'} />
          </button>
        </div>
      </div>
    </article>
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

async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    const previousFocus = document.activeElement;
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    try {
      textarea.select();
      if (!document.execCommand('copy')) throw new Error('Copy failed');
    } finally {
      textarea.remove();
      if (previousFocus instanceof HTMLElement) previousFocus.focus();
    }
  }
}

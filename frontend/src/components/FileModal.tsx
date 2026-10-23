import { Show, createSignal, createEffect, on, onCleanup, onMount } from 'solid-js';
import { Portal } from 'solid-js/web';
import './FileModal.css';
import {
  closeModal,
  closeFileMenu,
  fileMenu,
  files,
  openFileMenu,
  updateFileInList,
  tagFilters,
  statsTags,
} from '../store';
import {
  mimeIsImage,
  mimeIsVideo,
  mimeIsAudio,
  formatTimeDate,
  relativeTime,
  humanizeBytes,
  hasExpiry,
  fileUrl,
} from '../utils';
import { addFileTag, removeFileTag } from '../api';
import { copyFileLink, toggleVisibility } from '../fileActions';
import { loadStats } from './FileStats';
import { Icon } from './Icon';
import { TagEditor } from './TagEditor';
import { loadFiles, showAdjacentFile, hasAdjacentFile } from './FileGrid';
import type { FileData } from '../types';

export function FileModal(props: { file: FileData }) {
  const [localTags, setLocalTags] = createSignal<string[]>([]);

  let closeButton: HTMLButtonElement | undefined;
  let menuButton: HTMLButtonElement | undefined;
  let previouslyFocused: HTMLElement | null = null;
  let backdropMouseDownInside = false;

  onCleanup(() => {
    previouslyFocused?.focus?.();
  });

  onMount(() => {
    previouslyFocused = document.activeElement as HTMLElement | null;
    closeButton?.focus();
    document.body.classList.add('no-scroll');
    onCleanup(() => document.body.classList.remove('no-scroll'));

    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
        // Leave arrows alone where they already mean something:
        // text cursor movement and media seeking.
        if (e.altKey || e.ctrlKey || e.metaKey || e.shiftKey) return;
        const target = e.target as HTMLElement | null;
        if (target?.closest('input, textarea, video, audio')) return;
        e.preventDefault();
        showAdjacentFile(e.key === 'ArrowLeft' ? -1 : 1);
        return;
      }
      if (e.key !== 'Escape') return;
      e.preventDefault();
      closeModal();
    };
    document.addEventListener('keydown', onKey);
    onCleanup(() => document.removeEventListener('keydown', onKey));
  });

  // Keep localTags in sync whenever the underlying file's tag set changes,
  // including external updates while the same file stays open.
  createEffect(on(() => props.file.Tags, (tags) => {
    setLocalTags(tags?.map((t) => t.Name) || []);
  }));

  const file = () => props.file;

  // Warm the cache for neighbouring images so stepping through feels instant.
  createEffect(on(() => props.file.FileName, (name) => {
    const list = files();
    const index = list.findIndex((f) => f.FileName === name);
    if (index === -1) return;
    for (const neighbour of [list[index - 1], list[index + 1]]) {
      if (neighbour && mimeIsImage(neighbour.MimeType)) {
        new Image().src = fileUrl(neighbour.FileName);
      }
    }
  }));

  // Tags of a file as currently known, which may no longer be the open one
  // if the user moved on while a tag request was in flight.
  const currentTags = (fileName: string) =>
    file().FileName === fileName
      ? localTags()
      : files().find((f) => f.FileName === fileName)?.Tags?.map((t) => t.Name) ?? [];

  const [copied, setCopied] = createSignal(false);
  let copiedTimeoutId: ReturnType<typeof setTimeout> | undefined;
  onCleanup(() => clearTimeout(copiedTimeoutId));
  createEffect(on(() => props.file.FileName, () => {
    clearTimeout(copiedTimeoutId);
    setCopied(false);
  }, { defer: true }));

  const handleCopyLink = async () => {
    if (!(await copyFileLink(file()))) return;
    clearTimeout(copiedTimeoutId);
    setCopied(true);
    copiedTimeoutId = setTimeout(() => setCopied(false), 1500);
  };

  const isMenuOpen = () => fileMenu()?.returnFocus === menuButton;
  const handleMenuButton = () => {
    if (isMenuOpen()) {
      closeFileMenu();
      return;
    }
    // Open leftwards from the button's right edge; the menu flips if needed.
    const rect = menuButton!.getBoundingClientRect();
    openFileMenu(file(), rect.right, rect.bottom + 4, menuButton);
  };

  const handleAddTag = async (tag: string): Promise<string | null> => {
    const f = file();
    const result = await addFileTag(f.FileName, tag);
    if (!result.ok) return `Failed to add tag: ${result.error || 'unknown error'}`;

    const newTags = [...currentTags(f.FileName), tag];
    updateFileInList(f.FileName, {
      Tags: newTags.map((name) => ({ Name: name })),
    });
    if (file().FileName === f.FileName) setLocalTags(newTags);
    loadStats();
    return null;
  };

  const handleRemoveTag = async (tagName: string): Promise<string | null> => {
    const f = file();
    const result = await removeFileTag(f.FileName, tagName);
    if (!result.ok) return `Failed to remove tag: ${result.error || 'unknown error'}`;

    const newTags = currentTags(f.FileName).filter((t) => t !== tagName);
    updateFileInList(f.FileName, {
      Tags: newTags.map((name) => ({ Name: name })),
    });
    if (file().FileName === f.FileName) setLocalTags(newTags);
    loadStats();

    if (tagFilters().includes(tagName)) {
      loadFiles(0);
    }
    return null;
  };

  // Don't close if a drag started inside the modal and released on the
  // backdrop (e.g. text selection in an input).
  const onBackdropMouseDown = (e: MouseEvent) => {
    backdropMouseDownInside = e.target !== e.currentTarget;
  };
  let touchStartX = 0;
  let touchStartY = 0;
  let touchIgnored = false;
  const onWindowTouchStart = (e: TouchEvent) => {
    touchStartX = e.changedTouches[0].clientX;
    touchStartY = e.changedTouches[0].clientY;
    // Swipes over inputs are text selection, over audio likely scrubbing.
    touchIgnored = !!(e.target as HTMLElement).closest('input, textarea, audio');
  };
  const onWindowTouchEnd = (e: TouchEvent) => {
    if (touchIgnored) return;
    const dx = e.changedTouches[0].clientX - touchStartX;
    const dy = e.changedTouches[0].clientY - touchStartY;
    if (Math.abs(dx) < 50 || Math.abs(dx) < Math.abs(dy) * 1.5) return;
    showAdjacentFile(dx > 0 ? -1 : 1);
  };

  const onBackdropClick = (e: MouseEvent) => {
    if (e.target === e.currentTarget && !backdropMouseDownInside) closeModal();
    backdropMouseDownInside = false;
  };

  return (
    <Portal>
      <div
        id="file-modal"
        role="dialog"
        aria-modal="true"
        aria-label={file().OriginalFileName || file().FileName}
        onMouseDown={onBackdropMouseDown}
        onClick={onBackdropClick}
      >
        <div
          class="file-modal-window"
          onTouchStart={onWindowTouchStart}
          onTouchEnd={onWindowTouchEnd}
        >
          <div class="file-modal-titlebar">
            <div class="file-modal-titlebar-actions">
              <button
                type="button"
                class="file-modal-titlebar-button file-modal-copy"
                classList={{ copied: copied() }}
                aria-label={copied() ? 'Link copied' : 'Copy link'}
                title={copied() ? 'Link copied' : 'Copy link'}
                onClick={handleCopyLink}
              >
                <Show when={copied()} fallback={<Icon name="link" />}>
                  <Icon name="check" />
                </Show>
              </button>
              <button
                type="button"
                ref={menuButton}
                class="file-modal-titlebar-button file-modal-more"
                aria-label="File actions"
                title="More actions"
                aria-haspopup="menu"
                aria-expanded={isMenuOpen()}
                onClick={handleMenuButton}
              >
                <Icon name="ellipsis" />
              </button>
              <button
                type="button"
                ref={closeButton}
                class="file-modal-titlebar-button file-modal-close"
                aria-label="Close"
                onClick={closeModal}
              >
                <Icon name="x" />
              </button>
            </div>
          </div>

          <div class="file-modal-contents">
            <div class="file-big-preview">
              <Show when={hasAdjacentFile(-1)}>
                <button
                  type="button"
                  class="file-modal-nav file-modal-nav-prev"
                  aria-label="Previous file"
                  onClick={() => showAdjacentFile(-1)}
                >
                  <Icon name="chevron-left" />
                </button>
              </Show>
              <Show when={hasAdjacentFile(1)}>
                <button
                  type="button"
                  class="file-modal-nav file-modal-nav-next"
                  aria-label="Next file"
                  onClick={() => showAdjacentFile(1)}
                >
                  <Icon name="chevron-right" />
                </button>
              </Show>
              <Show when={mimeIsImage(file().MimeType)}>
                <img
                  id="file-preview-image"
                  src={fileUrl(file().FileName)}
                  alt={file().OriginalFileName || file().FileName}
                />
              </Show>
              <Show when={mimeIsVideo(file().MimeType)}>
                <video
                  id="file-preview-video"
                  src={fileUrl(file().FileName)}
                  controls
                  preload="metadata"
                />
              </Show>
              <Show when={mimeIsAudio(file().MimeType)}>
                <audio
                  id="file-preview-audio"
                  src={fileUrl(file().FileName)}
                  controls
                  preload="metadata"
                />
              </Show>
              <Show
                when={
                  !mimeIsImage(file().MimeType) &&
                  !mimeIsVideo(file().MimeType) &&
                  !mimeIsAudio(file().MimeType)
                }
              >
                <a
                  id="file-preview-generic"
                  href={fileUrl(file().FileName)}
                >
                  <div class="file-icon">
                    <Icon name="file" />
                  </div>
                </a>
              </Show>
            </div>

            <div class="file-modal-info">
              <div class="file-properties">
                <Show when={file().OriginalFileName}>
                  <div class="original-file-name" title="Original filename">
                    <Icon name="case-sensitive" />
                    <code id="file-modal-original-filename">{file().OriginalFileName}</code>
                  </div>
                </Show>

                <div class="file-id" title="File ID">
                  <Icon name="hash" />
                  <code id="file-modal-filename">{file().FileName}</code>
                </div>

                <div class="views">
                  <Icon name="eye" />
                  <span id="file-modal-views">
                    {(file().ViewsCount ?? 0) === 1 ? '1 view' : `${file().ViewsCount ?? 0} views`}
                  </span>
                </div>

                <div id="file-modal-filesize-wrapper" class="file-size" title={`${file().FileSize} bytes`}>
                  <Icon name="hard-drive" />
                  <span id="file-modal-filesize">{humanizeBytes(file().FileSize)}</span>
                </div>

                <Show when={file().MimeType}>
                  <div class="mime-type" title="MIME type">
                    <Icon name="file-type" />
                    <span id="file-modal-mimetype">{file().MimeType}</span>
                  </div>
                </Show>

                <div id="file-modal-createdat-wrapper" title={`Uploaded ${relativeTime(file().CreatedAt)}`}>
                  <Icon name="clock" />
                  <span id="file-modal-createdat">{formatTimeDate(file().CreatedAt)}</span>
                </div>

                <Show when={hasExpiry(file().ExpiryDate)}>
                  <div
                    id="file-modal-expirydate-wrapper"
                    class="expires-info"
                    title={formatTimeDate(file().ExpiryDate)}
                  >
                    <Icon name="trash-2" />
                    <span id="file-modal-expirydate">Expires {relativeTime(file().ExpiryDate)}</span>
                  </div>
                </Show>

                <button
                  type="button"
                  class="visibility-status"
                  id="file-modal-toggle-public-button"
                  title={file().Public
                    ? 'This file can be viewed by anyone with the link. Click to make it private.'
                    : 'This file can only be viewed by you. Click to make it public.'}
                  onClick={() => toggleVisibility(file())}
                >
                  <Icon name={file().Public ? 'lock-open' : 'lock'} />
                  <span id="file-modal-visibility">{file().Public ? 'Public' : 'Private'}</span>
                </button>
              </div>

              <div class="file-modal-tags">
                <div class="file-modal-tags-header">
                  <Icon name="tag" />
                  <span>Tags</span>
                </div>
                <TagEditor
                  tags={localTags()}
                  allTags={statsTags()}
                  resetKey={file().FileName}
                  onAdd={handleAddTag}
                  onRemove={handleRemoveTag}
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </Portal>
  );
}

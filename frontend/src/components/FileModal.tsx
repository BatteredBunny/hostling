import { Show, For, createSignal, createEffect, createMemo, on, onCleanup, onMount } from 'solid-js';
import { Portal } from 'solid-js/web';
import './FileModal.css';
import {
  closeModal,
  closeFileMenu,
  fileMenu,
  files,
  openFileMenu,
  updateFileInList,
  tagFilter,
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
import { addFileTag, removeFileTag, TAG_MAX_LENGTH, MAX_TAGS_PER_FILE } from '../api';
import { copyFileLink, toggleVisibility } from '../fileActions';
import { loadStats } from './FileStats';
import { Icon } from './Icon';
import { Tag } from './Tag';
import { loadFiles, showAdjacentFile, hasAdjacentFile } from './FileGrid';
import type { FileData } from '../types';

export function FileModal(props: { file: FileData }) {
  const [tagInput, setTagInput] = createSignal('');
  const [localTags, setLocalTags] = createSignal<string[]>([]);
  const [showAutocomplete, setShowAutocomplete] = createSignal(false);
  const [selectedSuggestionIndex, setSelectedSuggestionIndex] = createSignal(0);
  const [isAddingTag, setIsAddingTag] = createSignal(false);
  const [isRemovingTag, setIsRemovingTag] = createSignal(false);

  let blurTimeoutId: ReturnType<typeof setTimeout> | undefined;
  let closeButton: HTMLButtonElement | undefined;
  let menuButton: HTMLButtonElement | undefined;
  let previouslyFocused: HTMLElement | null = null;
  let backdropMouseDownInside = false;

  onCleanup(() => {
    if (blurTimeoutId) clearTimeout(blurTimeoutId);
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
      // If autocomplete is open, let its own onKeyDown dismiss it first.
      if (showAutocomplete()) return;
      e.preventDefault();
      closeModal();
    };
    document.addEventListener('keydown', onKey);
    onCleanup(() => document.removeEventListener('keydown', onKey));
  });

  createEffect(on(() => props.file.FileName, () => {
    setTagInput('');
    setShowAutocomplete(false);
    setSelectedSuggestionIndex(0);
  }));

  // Keep localTags in sync whenever the underlying file's tag set changes,
  // including external updates while the same file stays open.
  createEffect(on(() => props.file.Tags, (tags) => {
    setLocalTags(tags?.map((t) => t.Name) || []);
  }));

  createEffect(() => {
    const currentSuggestions = suggestions();
    if (tagInput().trim() && currentSuggestions.length === 0 && showAutocomplete()) {
      setShowAutocomplete(false);
    }
  });

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

  const suggestions = createMemo(() => {
    const input = tagInput().trim().toLowerCase();
    if (!input) return [];

    // Shows 5 first suggestions, should probably be done some other way
    const existingTags = localTags();
    return statsTags()
      .filter(tag =>
        tag.toLowerCase().includes(input) &&
        !existingTags.includes(tag)
      )
      .slice(0, 5);
  });

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

  const handleAddTag = async (tag?: string) => {
    if (isAddingTag()) return;
    const f = file();
    const tagToAdd = tag || tagInput().trim();
    if (!tagToAdd) return;
    if (tagToAdd.length > TAG_MAX_LENGTH) {
      alert(`Tag must be ${TAG_MAX_LENGTH} characters or fewer`);
      return;
    }
    if (localTags().length >= MAX_TAGS_PER_FILE) {
      alert(`A file can have at most ${MAX_TAGS_PER_FILE} tags`);
      return;
    }
    if (localTags().includes(tagToAdd)) return;

    setIsAddingTag(true);
    try {
      const result = await addFileTag(f.FileName, tagToAdd);
      if (result.ok) {
        const newTags = [...currentTags(f.FileName), tagToAdd];
        updateFileInList(f.FileName, {
          Tags: newTags.map((name) => ({ Name: name })),
        });
        if (file().FileName === f.FileName) {
          setLocalTags(newTags);
          setTagInput('');
          setShowAutocomplete(false);
          setSelectedSuggestionIndex(0);
        }
        loadStats();
      } else {
        alert(`Failed to add tag: ${result.error || 'unknown error'}`);
      }
    } finally {
      setIsAddingTag(false);
    }
  };

  const handleRemoveTag = async (tagName: string) => {
    if (isRemovingTag()) return;
    setIsRemovingTag(true);
    try {
      const f = file();
      const result = await removeFileTag(f.FileName, tagName);
      if (result.ok) {
        const newTags = currentTags(f.FileName).filter((t) => t !== tagName);
        updateFileInList(f.FileName, {
          Tags: newTags.map((name) => ({ Name: name })),
        });
        if (file().FileName === f.FileName) setLocalTags(newTags);
        loadStats();

        if (tagFilter() === tagName) {
          loadFiles(0);
        }
      } else {
        alert(`Failed to remove tag: ${result.error || 'unknown error'}`);
      }
    } finally {
      setIsRemovingTag(false);
    }
  };

  const handleTagInputChange = (value: string) => {
    setTagInput(value);
    setShowAutocomplete(value.trim().length > 0 && suggestions().length > 0);
    setSelectedSuggestionIndex(0);
  };

  const handleSelectSuggestion = (tag: string) => {
    handleAddTag(tag);
  };

  const handleKeyDown = (e: KeyboardEvent) => {
    const suggestionList = suggestions();

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (showAutocomplete() && suggestionList.length > 0) {
        setSelectedSuggestionIndex((prev) =>
          prev < suggestionList.length - 1 ? prev + 1 : prev
        );
      }
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (showAutocomplete() && suggestionList.length > 0) {
        setSelectedSuggestionIndex((prev) => (prev > 0 ? prev - 1 : 0));
      }
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (showAutocomplete() && suggestionList.length > 0) {
        handleSelectSuggestion(suggestionList[selectedSuggestionIndex()]);
      } else {
        handleAddTag();
      }
    } else if (e.key === 'Escape') {
      if (showAutocomplete()) {
        e.stopPropagation();
        setShowAutocomplete(false);
        setSelectedSuggestionIndex(0);
      }
    }
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
                <div id="file-modal-tags-list">
                  <For each={localTags()}>
                    {(tag) => <Tag name={tag} onRemove={handleRemoveTag} />}
                  </For>
                </div>
                <div class="file-modal-add-tag">
                  <div class="tag-input-wrapper">
                    <input
                      type="text"
                      id="file-modal-tag-input"
                      placeholder={localTags().length >= MAX_TAGS_PER_FILE ? `Tag limit reached (${MAX_TAGS_PER_FILE})` : 'Add tag...'}
                      maxLength={TAG_MAX_LENGTH}
                      disabled={localTags().length >= MAX_TAGS_PER_FILE}
                      value={tagInput()}
                      onInput={(e) => handleTagInputChange(e.currentTarget.value)}
                      onKeyDown={handleKeyDown}
                      onFocus={() => {
                        if (tagInput().trim() && suggestions().length > 0) {
                          setShowAutocomplete(true);
                        }
                      }}
                      onBlur={() => {
                        // Delay to allow click on suggestion
                        if (blurTimeoutId) clearTimeout(blurTimeoutId);
                        blurTimeoutId = setTimeout(() => setShowAutocomplete(false), 200);
                      }}
                    />
                    <Show when={showAutocomplete() && suggestions().length > 0}>
                      <div class="tag-autocomplete-dropdown">
                        <For each={suggestions()}>
                          {(suggestion, index) => (
                            <div
                              class="tag-autocomplete-item"
                              classList={{
                                selected: index() === selectedSuggestionIndex(),
                              }}
                              onClick={() => handleSelectSuggestion(suggestion)}
                              onMouseEnter={() => setSelectedSuggestionIndex(index())}
                            >
                              {suggestion}
                            </div>
                          )}
                        </For>
                      </div>
                    </Show>
                  </div>
                  <button
                    type="button"
                    id="file-modal-add-tag-btn"
                    class="create-button"
                    disabled={isAddingTag() || !tagInput().trim() || localTags().length >= MAX_TAGS_PER_FILE}
                    onClick={() => handleAddTag()}
                  >
                    Add
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Portal>
  );
}

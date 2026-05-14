import { Show, For, createSignal, createEffect, createMemo, on, onCleanup, onMount } from 'solid-js';
import { Portal } from 'solid-js/web';
import './FileModal.css';
import {
  closeModal,
  updateFileInList,
  removeFileFromList,
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
import { toggleFileVisibility, deleteFile, addFileTag, removeFileTag, TAG_MAX_LENGTH, MAX_TAGS_PER_FILE } from '../api';
import { loadStats } from './FileStats';
import { Icon } from './Icon';
import { Tag } from './Tag';
import { loadFiles } from './FileGrid';
import type { FileData } from '../types';

export function FileModal(props: { file: FileData }) {
  const [tagInput, setTagInput] = createSignal('');
  const [localTags, setLocalTags] = createSignal<string[]>([]);
  const [showAutocomplete, setShowAutocomplete] = createSignal(false);
  const [selectedSuggestionIndex, setSelectedSuggestionIndex] = createSignal(0);
  const [isDeleting, setIsDeleting] = createSignal(false);
  const [isAddingTag, setIsAddingTag] = createSignal(false);
  const [isRemovingTag, setIsRemovingTag] = createSignal(false);
  const [isTogglingVisibility, setIsTogglingVisibility] = createSignal(false);

  let blurTimeoutId: ReturnType<typeof setTimeout> | undefined;
  let closeButton: HTMLButtonElement | undefined;
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

  const handleToggleVisibility = async () => {
    if (isTogglingVisibility()) return;
    setIsTogglingVisibility(true);
    try {
      const f = file();
      const result = await toggleFileVisibility(f.FileName);
      if (result.ok) {
        updateFileInList(f.FileName, { Public: !f.Public });
      } else {
        alert(`Failed to change visibility: ${result.error || 'unknown error'}`);
      }
    } finally {
      setIsTogglingVisibility(false);
    }
  };

  const handleDelete = async () => {
    if (isDeleting()) return;
    const f = file();
    if (!confirm(`Are you sure you want to delete "${f.FileName}"?`)) return;
    setIsDeleting(true);
    try {
      const result = await deleteFile(f.FileName);
      if (result.ok) {
        removeFileFromList(f.FileName);
        loadStats();
        closeModal();
      } else {
        alert(`Failed to delete file: ${result.error || 'unknown error'}`);
      }
    } finally {
      setIsDeleting(false);
    }
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
        const newTags = [...localTags(), tagToAdd];
        setLocalTags(newTags);
        updateFileInList(f.FileName, {
          Tags: newTags.map((name) => ({ Name: name })),
        });
        setTagInput('');
        setShowAutocomplete(false);
        setSelectedSuggestionIndex(0);
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
        const newTags = localTags().filter((t) => t !== tagName);
        setLocalTags(newTags);
        updateFileInList(f.FileName, {
          Tags: newTags.map((name) => ({ Name: name })),
        });
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
        aria-labelledby="file-modal-filename"
        onMouseDown={onBackdropMouseDown}
        onClick={onBackdropClick}
      >
        <div class="file-modal-window">
          <div class="file-modal-titlebar">
            <a
              id="file-modal-filename-url"
              href={fileUrl(file().FileName)}
              target="_blank"
              rel="noopener noreferrer"
            >
              <span id="file-modal-filename">{file().FileName}</span>
            </a>
            <button
              type="button"
              ref={closeButton}
              class="file-modal-close"
              aria-label="Close"
              onClick={closeModal}
            >
              <Icon name="x" />
            </button>
          </div>

          <div class="file-modal-contents">
            <div class="file-big-preview">
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
                  <div class="original-file-name">
                    <code id="file-modal-original-filename">{file().OriginalFileName}</code>
                  </div>
                </Show>

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

                <div class="visibility-status" title={file().Public ? 'This file can be viewed by anyone with the link.' : 'This file can only be viewed by you.'}>
                  <Icon name={file().Public ? 'lock-open' : 'lock'} />
                  <span id="file-modal-visibility">{file().Public ? 'Public' : 'Private'}</span>
                </div>
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

              <div class="file-actions">
                <button
                  type="button"
                  class="toggle-visibility-button create-button"
                  id="file-modal-toggle-public-button"
                  disabled={isTogglingVisibility()}
                  onClick={handleToggleVisibility}
                >
                  {file().Public ? 'Make Private' : 'Make Public'}
                </button>
                <button
                  type="button"
                  class="delete-button"
                  id="file-modal-delete-button"
                  disabled={isDeleting()}
                  onClick={handleDelete}
                >
                  Delete
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Portal>
  );
}

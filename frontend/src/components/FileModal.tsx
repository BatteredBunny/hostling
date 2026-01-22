import { Show, For, createSignal, createEffect, createMemo } from 'solid-js';
import './FileModal.css';
import {
  modalFile,
  isModalOpen,
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
} from '../utils';
import { toggleFileVisibility, deleteFile, addFileTag, removeFileTag } from '../api';
import { loadStats } from './FileStats';
import { Icon } from './Icon';
import { Tag } from './Tag';
import { loadFiles } from './FileGrid';

export function FileModal() {
  const [tagInput, setTagInput] = createSignal('');
  const [localTags, setLocalTags] = createSignal<string[]>([]);
  const [showAutocomplete, setShowAutocomplete] = createSignal(false);
  const [selectedSuggestionIndex, setSelectedSuggestionIndex] = createSignal(0);

  createEffect(() => {
    const file = modalFile();
    if (file) {
      setLocalTags(file.Tags?.map((t) => t.Name) || []);
      setTagInput('');
      setShowAutocomplete(false);
      setSelectedSuggestionIndex(0);
    }
  });

  createEffect(() => {
    const currentSuggestions = suggestions();
    if (tagInput().trim() && currentSuggestions.length === 0 && showAutocomplete()) {
      setShowAutocomplete(false);
    }
  });

  const file = () => modalFile();

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
    const f = file();
    if (!f) return;

    const success = await toggleFileVisibility(f.FileName);
    if (success) {
      updateFileInList(f.FileName, { Public: !f.Public });
    } else {
      alert('Failed to change visibility');
    }
  };

  const handleDelete = async () => {
    const f = file();
    if (!f) return;

    if (confirm(`Are you sure you want to delete "${f.FileName}"?`)) {
      const success = await deleteFile(f.FileName);
      if (success) {
        removeFileFromList(f.FileName);
        loadStats();
        closeModal();
      } else {
        alert('Failed to delete file');
      }
    }
  };

  const handleAddTag = async (tag?: string) => {
    const f = file();
    const tagToAdd = tag || tagInput().trim();
    if (!f || !tagToAdd) return;

    const success = await addFileTag(f.FileName, tagToAdd);
    if (success) {
      const newTags = [...localTags(), tagToAdd];
      setLocalTags(newTags);
      updateFileInList(f.FileName, {
        Tags: newTags.map((name, i) => ({ ID: i, Name: name })),
      });
      setTagInput('');
      setShowAutocomplete(false);
      setSelectedSuggestionIndex(0);
      loadStats();
    } else {
      alert('Failed to add tag');
    }
  };

  const handleRemoveTag = async (tagName: string) => {
    const f = file();
    if (!f) return;

    const success = await removeFileTag(f.FileName, tagName);
    if (success) {
      const newTags = localTags().filter((t) => t !== tagName);
      setLocalTags(newTags);
      updateFileInList(f.FileName, {
        Tags: newTags.map((name, i) => ({ ID: i, Name: name })),
      });
      loadStats();

      if (tagFilter() === tagName) {
        loadFiles(0);
      }
    } else {
      alert('Failed to remove tag');
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
      setShowAutocomplete(false);
      setSelectedSuggestionIndex(0);
    }
  };

  return (
    <div
      id="file-modal"
      classList={{
        'file-modal-hidden': !isModalOpen(),
        'file-modal-visible': isModalOpen(),
      }}
    >
      <Show when={file()}>
        {(f) => (
          <div class="file-modal-window">
            <div class="file-modal-titlebar">
              <a
                id="file-modal-filename-url"
                href={`/${f().FileName}`}
                target="_blank"
                rel="noopener noreferrer"
              >
                <span id="file-modal-filename">{f().FileName}</span>
              </a>
              <button class="file-modal-close" onClick={closeModal}>
                <Icon name="x" />
              </button>
            </div>

            <div class="file-modal-contents">
              <div class="file-big-preview">
                <Show when={mimeIsImage(f().MimeType)}>
                  <img
                    id="file-preview-image"
                    src={`/${f().FileName}`}
                    alt="Uploaded image"
                    style={{ display: 'block' }}
                  />
                </Show>
                <Show when={mimeIsVideo(f().MimeType)}>
                  <video
                    id="file-preview-video"
                    src={`/${f().FileName}`}
                    controls
                    style={{ display: 'block' }}
                  />
                </Show>
                <Show when={mimeIsAudio(f().MimeType)}>
                  <audio
                    id="file-preview-audio"
                    src={`/${f().FileName}`}
                    controls
                    style={{ display: 'block' }}
                  />
                </Show>
                <Show
                  when={
                    !mimeIsImage(f().MimeType) &&
                    !mimeIsVideo(f().MimeType) &&
                    !mimeIsAudio(f().MimeType)
                  }
                >
                  <a
                    id="file-preview-generic"
                    href={`/${f().FileName}`}
                    style={{ display: 'block' }}
                  >
                    <div class="file-icon">
                      <Icon name="file" />
                    </div>
                  </a>
                </Show>
              </div>

              <div class="file-modal-info">
                <div class="file-properties">
                  <Show when={f().OriginalFileName}>
                    <div class="original-file-name">
                      <code id="file-modal-original-filename">{f().OriginalFileName}</code>
                    </div>
                  </Show>

                  <div class="views">
                    <Icon name="eye" />
                    <span id="file-modal-views">
                      {f().ViewsCount || 0} view{f().ViewsCount !== 1 ? 's' : ''}
                    </span>
                  </div>

                  <div id="file-modal-filesize-wrapper" class="file-size" title={`${f().FileSize} bytes`}>
                    <Icon name="hard-drive" />
                    <span id="file-modal-filesize">{humanizeBytes(f().FileSize)}</span>
                  </div>

                  <div id="file-modal-createdat-wrapper" title={`Uploaded ${relativeTime(f().CreatedAt)}`}>
                    <Icon name="clock" />
                    <span id="file-modal-createdat">{formatTimeDate(f().CreatedAt)}</span>
                  </div>

                  <Show when={hasExpiry(f().ExpiryDate)}>
                    <div
                      id="file-modal-expirydate-wrapper"
                      class="expires-info"
                      title={formatTimeDate(f().ExpiryDate)}
                    >
                      <Icon name="trash-2" />
                      <span id="file-modal-expirydate">Expires {relativeTime(f().ExpiryDate)}</span>
                    </div>
                  </Show>

                  <div class="visibility-status" title={f().Public ? 'This file can be viewed by anyone with the link.' : 'This file can only be viewed by you.'}>
                    <Icon name={f().Public ? 'lock-open' : 'lock'} />
                    <span id="file-modal-visibility">{f().Public ? 'Public' : 'Private'}</span>
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
                        placeholder="Add tag..."
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
                          setTimeout(() => setShowAutocomplete(false), 200);
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
                    <button id="file-modal-add-tag-btn" class="create-button" onClick={() => handleAddTag()}>
                      Add
                    </button>
                  </div>
                </div>

                <div class="file-actions">
                  <button
                    class="toggle-visibility-button create-button"
                    id="file-modal-toggle-public-button"
                    onClick={handleToggleVisibility}
                  >
                    {f().Public ? 'Make Private' : 'Make Public'}
                  </button>
                  <button
                    class="delete-button"
                    id="file-modal-delete-button"
                    onClick={handleDelete}
                  >
                    Delete
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
      </Show>
    </div>
  );
}

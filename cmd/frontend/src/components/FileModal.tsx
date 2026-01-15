import { Show, For, createSignal, createEffect } from 'solid-js';
import './FileModal.css';
import {
  modalFile,
  isModalOpen,
  closeModal,
  updateFileInList,
  removeFileFromList,
  tagFilter,
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

  createEffect(() => {
    const file = modalFile();
    if (file) {
      setLocalTags(file.Tags?.map((t) => t.Name) || []);
    }
  });

  const file = () => modalFile();

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

  const handleAddTag = async () => {
    const f = file();
    const tag = tagInput().trim();
    if (!f || !tag) return;

    const success = await addFileTag(f.FileName, tag);
    if (success) {
      const newTags = [...localTags(), tag];
      setLocalTags(newTags);
      updateFileInList(f.FileName, {
        Tags: newTags.map((name, i) => ({ ID: i, Name: name })),
      });
      setTagInput('');
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

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleAddTag();
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

                  <div class="visbility-status">
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
                    <input
                      type="text"
                      id="file-modal-tag-input"
                      placeholder="Add tag..."
                      value={tagInput()}
                      onInput={(e) => setTagInput(e.currentTarget.value)}
                      onKeyDown={handleKeyDown}
                    />
                    <button id="file-modal-add-tag-btn" class="create-button" onClick={handleAddTag}>
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

import { Show } from 'solid-js';
import './FileEntry.css';
import type { FileData } from '../types';
import {
  mimeIsImage,
  mimeIsVideo,
  mimeIsAudio,
  formatTimeDate,
  relativeTime,
  hasExpiry,
} from '../utils';
import { openModal } from '../store';
import { Icon } from './Icon';

interface FileEntryProps {
  file: FileData;
  onDelete: (fileName: string) => void;
}

export function FileEntry(props: FileEntryProps) {
  const file = () => props.file;

  const viewsText = () => {
    const count = file().ViewsCount || 0;
    return count === 1 ? '1 view' : `${count} views`;
  };

  const handleDelete = (e: Event) => {
    e.stopPropagation();
    if (confirm(`Are you sure you want to delete "${file().FileName}"?`)) {
      props.onDelete(file().FileName);
    }
  };

  return (
    <div class="file-entry">
      <div class="file-preview" onClick={() => openModal(file())}>
        <div class="file-thumbnail">
          <Show when={mimeIsImage(file().MimeType)}>
            <img
              class="preview-image"
              src={`/${file().FileName}`}
              alt="Uploaded image"
              style={{ display: 'block' }}
            />
          </Show>
          <Show when={mimeIsVideo(file().MimeType)}>
            <video
              class="preview-video"
              src={`/${file().FileName}`}
              style={{ display: 'block' }}
            />
          </Show>
          <Show when={mimeIsAudio(file().MimeType)}>
            <div class="preview-audio file-icon" style={{ display: 'flex' }}>
              <Icon name="music" />
            </div>
          </Show>
          <Show
            when={
              !mimeIsImage(file().MimeType) &&
              !mimeIsVideo(file().MimeType) &&
              !mimeIsAudio(file().MimeType)
            }
          >
            <div class="preview-generic file-icon" style={{ display: 'flex' }}>
              <Icon name="file" />
            </div>
          </Show>
        </div>
        <div class="file-name">{file().OriginalFileName || file().FileName}</div>
      </div>

      <div class="file-info">
        <Show when={hasExpiry(file().ExpiryDate)}>
          <div class="expires-info" title={formatTimeDate(file().ExpiryDate)}>
            <Icon name="trash-2" />
            <span class="expiry-text">{relativeTime(file().ExpiryDate)}</span>
          </div>
        </Show>

        <div class="views">
          <Icon name="eye" />
          <span class="views-text">{viewsText()}</span>
        </div>

        <div class="visibility-status">
          <Icon name={file().Public ? 'lock-open' : 'lock'} />
          <span class="visibility-text">{file().Public ? 'Public' : 'Private'}</span>
        </div>
      </div>

      <button class="delete-button-form delete-button" onClick={handleDelete}>
        Delete
      </button>
    </div>
  );
}

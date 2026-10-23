import { Show, onCleanup } from "solid-js";
import "./FileEntry.css";
import type { FileData } from "../types";
import {
  mimeIsImage,
  mimeIsVideo,
  mimeIsAudio,
  formatTimeDate,
  relativeTime,
  hasExpiry,
  fileUrl,
} from "../utils";
import { closeFileMenu, fileMenu, openFileMenu, openModal } from "../store";
import {
  Ellipsis as EllipsisIcon,
  Eye as EyeIcon,
  File as FileIcon,
  Lock as LockIcon,
  LockOpen as LockOpenIcon,
  Music as MusicIcon,
  Trash2 as Trash2Icon,
} from "lucide-solid";
import { Dynamic } from "solid-js/web";

const LONG_PRESS_MS = 500;
const LONG_PRESS_MOVE_TOLERANCE = 10;

export function FileEntry(props: { file: FileData }) {
  const file = () => props.file;

  let menuButton: HTMLButtonElement | undefined;
  let longPressTimer: ReturnType<typeof setTimeout> | undefined;
  let longPressFired = false;
  let touching = false;
  let touchStartX = 0;
  let touchStartY = 0;

  onCleanup(() => clearTimeout(longPressTimer));

  const viewsText = () => {
    const count = file().ViewsCount || 0;
    return count === 1 ? "1 view" : `${count} views`;
  };

  const isMenuOpen = () => fileMenu()?.file.FileName === file().FileName;

  const openMenuAt = (x: number, y: number) => {
    // Android fires both our long-press timer and a native contextmenu event.
    if (isMenuOpen()) return;
    openFileMenu(file(), x, y, menuButton);
  };

  const handleMenuButton = () => {
    if (isMenuOpen()) {
      closeFileMenu();
      return;
    }
    const rect = menuButton!.getBoundingClientRect();
    openFileMenu(file(), rect.left, rect.bottom + 4, menuButton);
  };

  const handleContextMenu = (e: MouseEvent) => {
    e.preventDefault();
    clearTimeout(longPressTimer);
    if (touching) longPressFired = true;
    openMenuAt(e.clientX, e.clientY);
  };

  // iOS Safari never fires contextmenu, so long-press is detected manually.
  const handleTouchStart = (e: TouchEvent) => {
    longPressFired = false;
    if ((e.target as HTMLElement).closest(".file-entry-menu-button")) return;
    touching = true;
    touchStartX = e.touches[0].clientX;
    touchStartY = e.touches[0].clientY;
    clearTimeout(longPressTimer);
    longPressTimer = setTimeout(() => {
      longPressFired = true;
      openMenuAt(touchStartX, touchStartY);
    }, LONG_PRESS_MS);
  };
  const handleTouchMove = (e: TouchEvent) => {
    const dx = e.touches[0].clientX - touchStartX;
    const dy = e.touches[0].clientY - touchStartY;
    if (Math.hypot(dx, dy) > LONG_PRESS_MOVE_TOLERANCE)
      clearTimeout(longPressTimer);
  };
  const handleTouchEnd = (e: TouchEvent) => {
    clearTimeout(longPressTimer);
    touching = false;
    // Stop the tap that ended a long-press from also opening the file.
    if (longPressFired) e.preventDefault();
  };

  const openCurrent = () => openModal(file());
  // The whole card is clickable; keyboard users get the focusable preview below.
  const handleCardClick = (e: MouseEvent) => {
    if ((e.target as HTMLElement).closest(".file-entry-menu-button")) return;
    openCurrent();
  };
  const handlePreviewKey = (e: KeyboardEvent) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      openCurrent();
    }
  };
  const previewAlt = () => file().OriginalFileName || file().FileName;

  return (
    <div
      class="file-entry"
      onClick={handleCardClick}
      classList={{ "menu-open": isMenuOpen() }}
      onContextMenu={handleContextMenu}
      onTouchStart={handleTouchStart}
      onTouchMove={handleTouchMove}
      onTouchEnd={handleTouchEnd}
      onTouchCancel={() => {
        clearTimeout(longPressTimer);
        touching = false;
      }}
    >
      <button
        type="button"
        ref={menuButton}
        class="file-entry-menu-button"
        aria-label="File actions"
        aria-haspopup="menu"
        aria-expanded={isMenuOpen()}
        onClick={handleMenuButton}
      >
        <EllipsisIcon class="lucide-icon" aria-hidden="true" />
      </button>

      <div
        class="file-preview"
        role="button"
        tabindex="0"
        onKeyDown={handlePreviewKey}
      >
        <div class="file-thumbnail">
          <Show when={mimeIsImage(file().MimeType)}>
            <img
              class="preview-image"
              src={fileUrl(file().FileName)}
              alt={previewAlt()}
              loading="lazy"
            />
          </Show>
          <Show when={mimeIsVideo(file().MimeType)}>
            <video
              class="preview-video"
              src={fileUrl(file().FileName)}
              preload="metadata"
              muted
            />
          </Show>
          <Show when={mimeIsAudio(file().MimeType)}>
            <div class="preview-audio file-icon">
              <MusicIcon class="lucide-icon" aria-hidden="true" />
            </div>
          </Show>
          <Show
            when={
              !mimeIsImage(file().MimeType) &&
              !mimeIsVideo(file().MimeType) &&
              !mimeIsAudio(file().MimeType)
            }
          >
            <div class="preview-generic file-icon">
              <FileIcon class="lucide-icon" aria-hidden="true" />
            </div>
          </Show>
        </div>
        <div class="file-name">
          {file().OriginalFileName || file().FileName}
        </div>
      </div>

      <div class="file-info">
        <div class="views" title={viewsText()}>
          <EyeIcon class="lucide-icon" aria-hidden="true" />
          <span class="views-text">{file().ViewsCount || 0}</span>
        </div>

        <div
          class="visibility-status"
          title={file().Public ? "Public" : "Private"}
        >
          <Dynamic
            component={file().Public ? LockOpenIcon : LockIcon}
            class="lucide-icon"
            aria-hidden="true"
          />
        </div>

        <Show when={hasExpiry(file().ExpiryDate)}>
          <div
            class="expires-info"
            title={`Expires ${relativeTime(file().ExpiryDate)} (${formatTimeDate(file().ExpiryDate)})`}
          >
            <Trash2Icon class="lucide-icon" aria-hidden="true" />
            <span class="expiry-text">
              {relativeTime(file().ExpiryDate).replace(/^in /, "")}
            </span>
          </div>
        </Show>
      </div>
    </div>
  );
}

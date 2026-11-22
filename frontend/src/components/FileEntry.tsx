import { Match, Show, Switch } from "solid-js";
import "./FileEntry.css";
import type { FileData } from "../types";
import {
  mimeIsImage,
  mimeIsVideo,
  mimeIsAudio,
  formatTimeDate,
  relativeTime,
  hasDate,
  fileUrl,
  countLabel,
  cx,
  displayName,
} from "../utils";
import { closeFileMenu, fileMenu, openFileMenu, openModal } from "../store";
import EllipsisIcon from "lucide-solid/icons/ellipsis";
import EyeIcon from "lucide-solid/icons/eye";
import FileIcon from "lucide-solid/icons/file";
import MusicIcon from "lucide-solid/icons/eighth-notes";
import Trash2Icon from "lucide-solid/icons/trash";
import { VisibilityIcon } from "./VisibilityIcon";
import { createContextMenuTrigger } from "../contextMenuTrigger";

export function FileEntry(props: { file: FileData }) {
  const file = () => props.file;

  let menuButton: HTMLButtonElement | undefined;
  const views = () => file().ViewsCount || 0;

  const isMenuOpen = () => fileMenu()?.file.FileName === file().FileName;

  const menuTrigger = createContextMenuTrigger((x, y) => {
    if (!isMenuOpen()) openFileMenu(file(), x, y, menuButton);
  }, ".file-entry-menu-button");

  const handleMenuButton = () => {
    if (isMenuOpen()) {
      closeFileMenu();
      return;
    }
    const rect = menuButton!.getBoundingClientRect();
    openFileMenu(file(), rect.left, rect.bottom + 4, menuButton);
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
  const name = () => displayName(file());

  return (
    <li
      class={cx("file-entry", { "menu-open": isMenuOpen() })}
      onClick={handleCardClick}
      onContextMenu={menuTrigger.onContextMenu}
      onTouchStart={menuTrigger.onTouchStart}
      onTouchMove={menuTrigger.onTouchMove}
      onTouchEnd={menuTrigger.onTouchEnd}
      onTouchCancel={menuTrigger.onTouchCancel}
    >
      <button
        type="button"
        ref={menuButton}
        class="file-entry-menu-button"
        aria-label={`Actions for ${name()}`}
        aria-haspopup="menu"
        aria-expanded={isMenuOpen()}
        onClick={handleMenuButton}
      >
        <EllipsisIcon class="lucide-icon" aria-hidden="true" />
      </button>

      <div
        class="file-preview"
        role="button"
        aria-label={`Preview ${name()}`}
        tabindex="0"
        onKeyDown={handlePreviewKey}
      >
        <div class="file-thumbnail">
          <Switch
            fallback={
              <div class="preview-generic file-icon">
                <FileIcon class="lucide-icon" aria-hidden="true" />
              </div>
            }
          >
            <Match when={mimeIsImage(file().MimeType)}>
              <img
                class="preview-image"
                src={fileUrl(file().FileName)}
                alt={name()}
                loading="lazy"
              />
            </Match>
            <Match when={mimeIsVideo(file().MimeType)}>
              <video
                class="preview-video"
                src={fileUrl(file().FileName)}
                preload="metadata"
                muted
              />
            </Match>
            <Match when={mimeIsAudio(file().MimeType)}>
              <div class="preview-audio file-icon">
                <MusicIcon class="lucide-icon" aria-hidden="true" />
              </div>
            </Match>
          </Switch>
        </div>
        <div class="file-name">{name()}</div>
      </div>

      <div class="file-info">
        <div class="views" title={countLabel(views(), "view")}>
          <EyeIcon class="lucide-icon" aria-hidden="true" />
          <span class="views-text">{views()}</span>
        </div>

        <div
          class="visibility-status"
          title={file().Public ? "Public" : "Private"}
        >
          <VisibilityIcon public={file().Public} />
        </div>

        <Show when={hasDate(file().ExpiryDate)}>
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
    </li>
  );
}

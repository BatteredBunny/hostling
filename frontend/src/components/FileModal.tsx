import {
  Match,
  Show,
  Switch,
  createEffect,
  createMemo,
  createSignal,
  onCleanup,
  untrack,
} from "solid-js";
import { NativeDialog } from "./NativeDialog";
import "./FileModal.css";
import {
  closeModal,
  closeFileMenu,
  fileMenu,
  files,
  openFileMenu,
  updateFileInList,
  tagFilters,
  setCurrentPage,
  statsTags,
} from "../store";
import {
  mimeIsImage,
  mimeIsVideo,
  mimeIsAudio,
  formatTimeDate,
  relativeTime,
  humanizeBytes,
  hasDate,
  fileUrl,
  countLabel,
  displayName,
  ariaBool,
} from "../utils";
import { addFileTag, removeFileTag } from "../generated";
import { apiData } from "../client";
import { copyFileLink, toggleVisibility } from "../fileActions";
import { loadStats } from "./FileStats";
import CaseSensitiveIcon from "@lucide/solid/icons/case-sensitive";
import CheckIcon from "@lucide/solid/icons/check";
import ChevronLeftIcon from "@lucide/solid/icons/chevron-left";
import ChevronRightIcon from "@lucide/solid/icons/chevron-right";
import ClockIcon from "@lucide/solid/icons/clock";
import EllipsisIcon from "@lucide/solid/icons/ellipsis";
import EyeIcon from "@lucide/solid/icons/eye";
import FileIcon from "@lucide/solid/icons/file";
import FileTypeIcon from "@lucide/solid/icons/file-type";
import HardDriveIcon from "@lucide/solid/icons/hard-drive";
import HashIcon from "@lucide/solid/icons/hash";
import LinkIcon from "@lucide/solid/icons/link";
import TagIcon from "@lucide/solid/icons/tag";
import Trash2Icon from "@lucide/solid/icons/trash";
import XIcon from "@lucide/solid/icons/x";
import { VisibilityIcon } from "./VisibilityIcon";
import { TagEditor } from "./TagEditor";
import { loadFiles, showAdjacentFile, hasAdjacentFile } from "./FileGrid";
import type { FileData } from "../types";

export function FileModal(props: { file: FileData }) {
  let menuButton: HTMLButtonElement | undefined;
  const onKey = (e: KeyboardEvent) => {
    if (e.key === "ArrowLeft" || e.key === "ArrowRight") {
      // Leave arrows alone where they already mean something:
      // text cursor movement and media seeking.
      if (e.altKey || e.ctrlKey || e.metaKey || e.shiftKey) return;
      const target = e.target as HTMLElement | null;
      if (target?.closest("input, textarea, video, audio")) return;
      e.preventDefault();
      showAdjacentFile(e.key === "ArrowLeft" ? -1 : 1);
      return;
    }
  };

  const file = () => props.file;
  const fileName = createMemo(() => file().FileName);
  const name = () => displayName(file());
  // updateFileInList also updates the open file, so this follows tag edits.
  const tagNames = () => file().Tags.map((t) => t.Name);
  // Reassigning a media element's src restarts playback, even for the same URL.
  // Metadata updates must only notify the media binding when the file changes.
  const mediaSource = createMemo(() => fileUrl(fileName()));

  // Warm the cache for neighbouring images so stepping through feels instant.
  createEffect(
    () => {
      const current = fileName();
      // Only a new open file prompts this, not list updates.
      return untrack(() => {
        const list = files();
        const index = list.findIndex((f) => f.FileName === current);
        if (index === -1) return [];
        return [list[index - 1], list[index + 1]]
          .filter((f) => f && mimeIsImage(f.MimeType))
          .map((f) => fileUrl(f.FileName));
      });
    },
    (urls) => {
      for (const url of urls) new Image().src = url;
    },
  );

  // Tags of a file as currently known, which may no longer be the open one
  // if the user moved on while a tag request was in flight.
  const currentTags = (target: string) =>
    fileName() === target
      ? tagNames()
      : (files()
          .find((f) => f.FileName === target)
          ?.Tags.map((t) => t.Name) ?? []);

  // Remembers which file was copied, so stepping to another file resets it.
  const [copiedFile, setCopiedFile] = createSignal<string | null>(null);
  const copied = () => copiedFile() === fileName();
  let copiedTimeoutId: ReturnType<typeof setTimeout> | undefined;
  onCleanup(() => clearTimeout(copiedTimeoutId));

  const handleCopyLink = async () => {
    const copiedName = fileName();
    if (!(await copyFileLink(file()))) return;
    clearTimeout(copiedTimeoutId);
    setCopiedFile(copiedName);
    copiedTimeoutId = setTimeout(() => setCopiedFile(null), 1500);
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
    try {
      await apiData(addFileTag({ body: { file_name: f.FileName, tag } }));
    } catch (error) {
      return error instanceof Error ? error.message : "Failed to add tag";
    }

    const newTags = [...currentTags(f.FileName), tag];
    updateFileInList(f.FileName, {
      Tags: newTags.map((Name) => ({ Name })),
    });
    loadStats();
    return null;
  };

  const handleRemoveTag = async (tagName: string): Promise<string | null> => {
    const f = file();
    try {
      await apiData(
        removeFileTag({ body: { file_name: f.FileName, tag: tagName } }),
      );
    } catch (error) {
      return error instanceof Error ? error.message : "Failed to remove tag";
    }

    const newTags = currentTags(f.FileName).filter((t) => t !== tagName);
    updateFileInList(f.FileName, {
      Tags: newTags.map((Name) => ({ Name })),
    });
    loadStats();

    // The file no longer matches the filter, so refill from the first page.
    if (tagFilters().includes(tagName)) {
      setCurrentPage(0);
      loadFiles(0);
    }
    return null;
  };

  let touchStartX = 0;
  let touchStartY = 0;
  let touchIgnored = false;
  const onWindowTouchStart = (e: TouchEvent) => {
    touchStartX = e.changedTouches[0].clientX;
    touchStartY = e.changedTouches[0].clientY;
    // Swipes over inputs are text selection, over audio likely scrubbing.
    touchIgnored = !!(e.target as HTMLElement).closest(
      "input, textarea, audio",
    );
  };
  const onWindowTouchEnd = (e: TouchEvent) => {
    if (touchIgnored) return;
    const dx = e.changedTouches[0].clientX - touchStartX;
    const dy = e.changedTouches[0].clientY - touchStartY;
    if (Math.abs(dx) < 50 || Math.abs(dx) < Math.abs(dy) * 1.5) return;
    showAdjacentFile(dx > 0 ? -1 : 1);
  };

  return (
    <NativeDialog
      open
      id="file-modal"
      label={name()}
      onDismiss={closeModal}
      onKeyDown={onKey}
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
              class={[
                "file-modal-titlebar-button file-modal-copy",
                {
                  copied: copied(),
                },
              ]}
              aria-label={copied() ? "Link copied" : "Copy link"}
              title={copied() ? "Link copied" : "Copy link"}
              onClick={handleCopyLink}
            >
              <Show
                when={copied()}
                fallback={<LinkIcon class="lucide-icon" aria-hidden="true" />}
              >
                <CheckIcon class="lucide-icon" aria-hidden="true" />
              </Show>
            </button>
            <button
              type="button"
              ref={menuButton}
              class="file-modal-titlebar-button file-modal-more"
              aria-label="File actions"
              title="More actions"
              aria-haspopup="menu"
              aria-expanded={ariaBool(isMenuOpen())}
              onClick={handleMenuButton}
            >
              <EllipsisIcon class="lucide-icon" aria-hidden="true" />
            </button>
            <button
              type="button"
              autofocus
              class="file-modal-titlebar-button file-modal-close"
              aria-label="Close"
              onClick={closeModal}
            >
              <XIcon class="lucide-icon" aria-hidden="true" />
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
                <ChevronLeftIcon class="lucide-icon" aria-hidden="true" />
              </button>
            </Show>
            <Show when={hasAdjacentFile(1)}>
              <button
                type="button"
                class="file-modal-nav file-modal-nav-next"
                aria-label="Next file"
                onClick={() => showAdjacentFile(1)}
              >
                <ChevronRightIcon class="lucide-icon" aria-hidden="true" />
              </button>
            </Show>
            <Switch
              fallback={
                <a
                  id="file-preview-generic"
                  aria-label={`Download ${name()}`}
                  href={mediaSource()}
                >
                  <div class="file-icon">
                    <FileIcon class="lucide-icon" aria-hidden="true" />
                  </div>
                </a>
              }
            >
              <Match when={mimeIsImage(file().MimeType)}>
                <img id="file-preview-image" src={mediaSource()} alt={name()} />
              </Match>
              <Match when={mimeIsVideo(file().MimeType)}>
                <video
                  id="file-preview-video"
                  aria-label={`Video preview: ${name()}`}
                  src={mediaSource()}
                  controls
                  preload="metadata"
                />
              </Match>
              <Match when={mimeIsAudio(file().MimeType)}>
                <audio
                  id="file-preview-audio"
                  aria-label={`Audio preview: ${name()}`}
                  src={mediaSource()}
                  controls
                  preload="metadata"
                />
              </Match>
            </Switch>
          </div>

          <div class="file-modal-info">
            <div class="file-properties">
              <Show when={file().OriginalFileName}>
                <div class="original-file-name" title="Original filename">
                  <CaseSensitiveIcon class="lucide-icon" aria-hidden="true" />
                  <code id="file-modal-original-filename">
                    {file().OriginalFileName}
                  </code>
                </div>
              </Show>

              <div class="file-id" title="File ID">
                <HashIcon class="lucide-icon" aria-hidden="true" />
                <code id="file-modal-filename">{file().FileName}</code>
              </div>

              <div class="views">
                <EyeIcon class="lucide-icon" aria-hidden="true" />
                <span id="file-modal-views">
                  {countLabel(file().ViewsCount ?? 0, "view")}
                </span>
              </div>

              <div
                id="file-modal-filesize-wrapper"
                class="file-size"
                title={`${file().FileSize} bytes`}
              >
                <HardDriveIcon class="lucide-icon" aria-hidden="true" />
                <span id="file-modal-filesize">
                  {humanizeBytes(file().FileSize)}
                </span>
              </div>

              <Show when={file().MimeType}>
                <div class="mime-type" title="MIME type">
                  <FileTypeIcon class="lucide-icon" aria-hidden="true" />
                  <span id="file-modal-mimetype">{file().MimeType}</span>
                </div>
              </Show>

              <div
                id="file-modal-createdat-wrapper"
                title={`Uploaded ${relativeTime(file().CreatedAt)}`}
              >
                <ClockIcon class="lucide-icon" aria-hidden="true" />
                <span id="file-modal-createdat">
                  {formatTimeDate(file().CreatedAt)}
                </span>
              </div>

              <Show when={hasDate(file().ExpiryDate)}>
                <div
                  id="file-modal-expirydate-wrapper"
                  class="expires-info"
                  title={formatTimeDate(file().ExpiryDate)}
                >
                  <Trash2Icon class="lucide-icon" aria-hidden="true" />
                  <span id="file-modal-expirydate">
                    Expires {relativeTime(file().ExpiryDate)}
                  </span>
                </div>
              </Show>

              <button
                type="button"
                class="visibility-status"
                id="file-modal-toggle-public-button"
                title={
                  file().Public
                    ? "This file can be viewed by anyone with the link. Click to make it private."
                    : "This file can only be viewed by you. Click to make it public."
                }
                onClick={() => toggleVisibility(file())}
              >
                <VisibilityIcon public={file().Public} />
                <span id="file-modal-visibility">
                  {file().Public ? "Public" : "Private"}
                </span>
              </button>
            </div>

            <div class="file-modal-tags">
              <div class="file-modal-tags-header">
                <TagIcon class="lucide-icon" aria-hidden="true" />
                <span>Tags</span>
              </div>
              {/* Keyed so switching files starts the editor from scratch. */}
              <Show when={fileName()} keyed>
                <TagEditor
                  tags={tagNames()}
                  allTags={statsTags()}
                  onAdd={handleAddTag}
                  onRemove={handleRemoveTag}
                />
              </Show>
            </div>
          </div>
        </div>
      </div>
    </NativeDialog>
  );
}

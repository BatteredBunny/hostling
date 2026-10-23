import {
  Show,
  createSignal,
  createEffect,
  createMemo,
  on,
  onCleanup,
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
  statsTags,
} from "../store";
import {
  mimeIsImage,
  mimeIsVideo,
  mimeIsAudio,
  formatTimeDate,
  relativeTime,
  humanizeBytes,
  hasExpiry,
  fileUrl,
} from "../utils";
import { addFileTag, removeFileTag } from "../generated";
import { apiData } from "../client";
import { copyFileLink, toggleVisibility } from "../fileActions";
import { loadStats } from "./FileStats";
import {
  CaseSensitive as CaseSensitiveIcon,
  Check as CheckIcon,
  ChevronLeft as ChevronLeftIcon,
  ChevronRight as ChevronRightIcon,
  Clock as ClockIcon,
  Ellipsis as EllipsisIcon,
  Eye as EyeIcon,
  File as FileIcon,
  FileType as FileTypeIcon,
  HardDrive as HardDriveIcon,
  Hash as HashIcon,
  Link as LinkIcon,
  Lock as LockIcon,
  LockOpen as LockOpenIcon,
  Tag as TagIcon,
  Trash2 as Trash2Icon,
  X as XIcon,
} from "lucide-solid";
import { Dynamic } from "solid-js/web";
import { TagEditor } from "./TagEditor";
import { loadFiles, showAdjacentFile, hasAdjacentFile } from "./FileGrid";
import type { FileData } from "../types";

export function FileModal(props: { file: FileData }) {
  const [localTags, setLocalTags] = createSignal<string[]>([]);

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

  // Keep localTags in sync whenever the underlying file's tag set changes,
  // including external updates while the same file stays open.
  createEffect(
    on(
      () => props.file.Tags,
      (tags) => {
        setLocalTags(tags?.map((t) => t.Name) || []);
      },
    ),
  );

  const file = () => props.file;
  // Reassigning a media element's src restarts playback, even for the same URL.
  // Metadata updates must only notify the media binding when the file changes.
  const mediaSource = createMemo(() => fileUrl(file().FileName));

  // Warm the cache for neighbouring images so stepping through feels instant.
  createEffect(
    on(
      () => props.file.FileName,
      (name) => {
        const list = files();
        const index = list.findIndex((f) => f.FileName === name);
        if (index === -1) return;
        for (const neighbour of [list[index - 1], list[index + 1]]) {
          if (neighbour && mimeIsImage(neighbour.MimeType)) {
            new Image().src = fileUrl(neighbour.FileName);
          }
        }
      },
    ),
  );

  // Tags of a file as currently known, which may no longer be the open one
  // if the user moved on while a tag request was in flight.
  const currentTags = (fileName: string) =>
    file().FileName === fileName
      ? localTags()
      : (files()
          .find((f) => f.FileName === fileName)
          ?.Tags?.map((t) => t.Name) ?? []);

  const [copied, setCopied] = createSignal(false);
  let copiedTimeoutId: ReturnType<typeof setTimeout> | undefined;
  onCleanup(() => clearTimeout(copiedTimeoutId));
  createEffect(
    on(
      () => props.file.FileName,
      () => {
        clearTimeout(copiedTimeoutId);
        setCopied(false);
      },
      { defer: true },
    ),
  );

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
    try {
      await apiData(addFileTag({ body: { file_name: f.FileName, tag } }));
    } catch (error) {
      return error instanceof Error ? error.message : "Failed to add tag";
    }

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
    try {
      await apiData(
        removeFileTag({ body: { file_name: f.FileName, tag: tagName } }),
      );
    } catch (error) {
      return error instanceof Error ? error.message : "Failed to remove tag";
    }

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
      label={file().OriginalFileName || file().FileName}
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
              class="file-modal-titlebar-button file-modal-copy"
              classList={{ copied: copied() }}
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
              aria-expanded={isMenuOpen()}
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
            <Show when={mimeIsImage(file().MimeType)}>
              <img
                id="file-preview-image"
                src={mediaSource()}
                alt={file().OriginalFileName || file().FileName}
              />
            </Show>
            <Show when={mimeIsVideo(file().MimeType)}>
              <video
                id="file-preview-video"
                src={mediaSource()}
                controls
                preload="metadata"
              />
            </Show>
            <Show when={mimeIsAudio(file().MimeType)}>
              <audio
                id="file-preview-audio"
                src={mediaSource()}
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
              <a id="file-preview-generic" href={fileUrl(file().FileName)}>
                <div class="file-icon">
                  <FileIcon class="lucide-icon" aria-hidden="true" />
                </div>
              </a>
            </Show>
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
                  {(file().ViewsCount ?? 0) === 1
                    ? "1 view"
                    : `${file().ViewsCount ?? 0} views`}
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

              <Show when={hasExpiry(file().ExpiryDate)}>
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
                <Dynamic
                  component={file().Public ? LockOpenIcon : LockIcon}
                  class="lucide-icon"
                  aria-hidden="true"
                />
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
    </NativeDialog>
  );
}

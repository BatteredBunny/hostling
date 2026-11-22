import "./UploadPage.css";
import { LoadError } from "./LoadError";
import { SectionPanel } from "./SectionPanel";
import { For, Match, Show, Switch, createSignal, onCleanup } from "solid-js";
import { createStore } from "solid-js/store";
import CheckIcon from "lucide-solid/icons/check";
import FileIcon from "lucide-solid/icons/file";
import LinkIcon from "lucide-solid/icons/link";
import XIcon from "lucide-solid/icons/x";
import { useFrontendContext } from "../context";
import { uploadFile } from "../generated";
import type { UploadFileData } from "../generated";
import { formDataBodySerializer } from "../generated/client";
import { ApiError, apiData } from "../client";
import { Input } from "./Input";
import { UploadCommand } from "./UploadCommand";
import { showToast } from "../toast";
import { copyText, countLabel, cx, fileUrl, humanizeBytes } from "../utils";

// XHR preserves upload progress without buffering the file into JavaScript memory.
// The generated SDK still owns the URL, request contract, and response parsing.
function uploadTransport(
  body: NonNullable<UploadFileData["body"]>,
  progress: (value: number) => void,
): typeof fetch {
  return async (input) =>
    new Promise<Response>((resolve, reject) => {
      const request = input as Request;
      const xhr = new XMLHttpRequest();
      const abort = () => xhr.abort();
      const cleanup = () => request.signal.removeEventListener("abort", abort);
      if (request.signal.aborted) {
        reject(new DOMException("Upload cancelled.", "AbortError"));
        return;
      }
      xhr.open(request.method, request.url);
      request.headers.forEach((value, name) => {
        if (name.toLowerCase() !== "content-type")
          xhr.setRequestHeader(name, value);
      });
      xhr.upload.onprogress = (event) => {
        if (event.lengthComputable)
          progress(Math.round((event.loaded / event.total) * 100));
      };
      xhr.onerror = () =>
        reject(
          new Error("Upload failed. Check your connection and try again."),
        );
      xhr.onabort = () =>
        reject(new DOMException("Upload cancelled.", "AbortError"));
      xhr.onloadend = cleanup;
      xhr.onload = () =>
        resolve(
          new Response(xhr.responseText, {
            status: xhr.status,
            headers: {
              "Content-Type":
                xhr.getResponseHeader("Content-Type") || "text/plain",
            },
          }),
        );
      request.signal.addEventListener("abort", abort, { once: true });
      try {
        xhr.send(formDataBodySerializer.bodySerializer(body) as FormData);
      } catch (error) {
        cleanup();
        reject(error);
      }
    });
}

interface QueuedFile {
  id: number;
  file: File;
  preview?: string;
  status: "pending" | "uploading" | "done" | "failed";
  progress: number;
  error?: string;
  // The server's name for the file once uploaded.
  name?: string;
}

let nextQueueId = 0;

export function UploadPage() {
  const { context, error: contextError, retry } = useFrontendContext();
  const [queue, setQueue] = createStore<QueuedFile[]>([]);
  const [token, setToken] = createSignal("");
  const [expiry, setExpiry] = createSignal("");
  const [busy, setBusy] = createSignal(false);
  const [dragOver, setDragOver] = createSignal(false);
  let input!: HTMLInputElement;
  let uploadController: AbortController | undefined;

  const loggedIn = () => !!context()?.logged_in;
  const toUpload = () =>
    queue.filter(
      (item) => item.status === "pending" || item.status === "failed",
    );
  const update = (id: number, changes: Partial<QueuedFile>) =>
    setQueue((item) => item.id === id, changes);

  const release = (items: readonly QueuedFile[]) => {
    for (const item of items)
      if (item.preview) URL.revokeObjectURL(item.preview);
  };
  onCleanup(() => {
    uploadController?.abort();
    release(queue);
  });

  function addFiles(files?: FileList | null) {
    if (busy() || !files?.length) return;
    // A new selection starts a new batch; finished uploads make way.
    const finished = queue.filter((item) => item.status === "done");
    release(finished);
    setQueue((items) => [
      ...items.filter((item) => item.status !== "done"),
      ...Array.from(files, (file) => ({
        id: nextQueueId++,
        file,
        preview: /^(image|video)\//.test(file.type)
          ? URL.createObjectURL(file)
          : undefined,
        status: "pending" as const,
        progress: 0,
      })),
    ]);
  }

  function remove(id: number) {
    release(queue.filter((item) => item.id === id));
    setQueue((items) => items.filter((item) => item.id !== id));
  }

  const viewUrl = (name: string) =>
    loggedIn() ? `/gallery?file=${encodeURIComponent(name)}` : fileUrl(name);

  async function copyLink(name: string) {
    try {
      await copyText(new URL(fileUrl(name), window.location.origin).href);
      showToast("Link copied.");
    } catch {
      showToast("Could not copy the link.", "error");
    }
  }

  async function uploadOne(item: QueuedFile, signal: AbortSignal) {
    const body = {
      file: item.file,
      plain: true,
      expiry_date: expiry() || undefined,
      upload_token: loggedIn() ? undefined : token(),
    };
    const progress = (value: number) => update(item.id, { progress: value });
    const name = await apiData(
      uploadFile({
        body,
        signal,
        fetch: import.meta.env.PUBLIC_DEMO
          ? (await import("../demo/transport")).demoUploadTransport(progress)
          : uploadTransport(body, progress),
      }),
    );
    return name.trim().replace(/^\//, "");
  }

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    if (busy()) return;
    const items = toUpload();
    if (!items.length) {
      showToast("Choose files to upload.", "error");
      return;
    }
    setBusy(true);
    const controller = new AbortController();
    uploadController = controller;
    let failed = 0;
    try {
      for (const item of items) {
        if (controller.signal.aborted) break;
        update(item.id, { status: "uploading", progress: 0, error: undefined });
        try {
          // One at a time, so each file gets the full bandwidth in turn.
          // oxlint-disable-next-line no-await-in-loop
          const name = await uploadOne(item, controller.signal);
          update(item.id, { status: "done", progress: 100, name });
        } catch (failure) {
          if (
            failure instanceof ApiError &&
            failure.status === 401 &&
            !loggedIn()
          ) {
            update(item.id, { status: "pending", progress: 0 });
            setToken("");
            showToast("Invalid upload token.", "error");
            return;
          }
          failed++;
          update(item.id, {
            status: "failed",
            error:
              failure instanceof Error ? failure.message : "Upload failed.",
          });
        }
      }
      if (failed)
        showToast(`${countLabel(failed, "file")} failed to upload.`, "error");
      else if (!controller.signal.aborted)
        showToast(
          items.length === 1
            ? "File uploaded."
            : `${items.length} files uploaded.`,
        );
    } finally {
      uploadController = undefined;
      setBusy(false);
    }
  }

  return (
    <>
      <div class="branding-title">
        <div class="branding-main">{context()?.branding ?? "…"}</div>
        <div class="tagline">{context()?.tagline}</div>
      </div>
      <UploadCommand />
      <SectionPanel>
        <Show when={contextError()}>
          <LoadError message={contextError()} onRetry={retry} />
        </Show>
        <form aria-label="Upload files" class="upload-form" onSubmit={submit}>
          <Show when={!loggedIn()}>
            <Input
              label="Upload token"
              id="upload_token"
              name="upload_token"
              required
              value={token()}
              disabled={busy()}
              autocomplete="current-password"
              onInput={(event) => setToken(event.currentTarget.value)}
            />
          </Show>
          <input
            ref={input}
            type="file"
            multiple
            aria-label="Files to upload"
            id="file"
            class="file-input"
            disabled={busy()}
            onChange={(event) => {
              addFiles(event.currentTarget.files);
              event.currentTarget.value = "";
            }}
          />
          <div
            class={cx("file-drop-zone", {
              disabled: busy(),
              "file-selected": queue.length > 0,
              "drag-over": dragOver() && !busy(),
            })}
            onClick={(event) => {
              // Clicks on a file's own controls stay with them.
              if (!busy() && !(event.target as Element).closest("a, li button"))
                input.click();
            }}
            onDragOver={(event) => {
              event.preventDefault();
              setDragOver(true);
            }}
            onDragLeave={(event) => {
              if (!event.currentTarget.contains(event.relatedTarget as Node))
                setDragOver(false);
            }}
            onDrop={(event) => {
              event.preventDefault();
              setDragOver(false);
              addFiles(event.dataTransfer?.files);
            }}
          >
            <button type="button" class="drop-text" disabled={busy()}>
              {queue.length
                ? "Drop more files or click to add"
                : "Drop files here or click to select"}
            </button>
            <Show when={queue.length > 0}>
              <ul class="upload-tiles" aria-label="Selected files">
                <For each={queue}>
                  {(item) => (
                    <li class={cx("upload-tile", item.status)}>
                      <div class="upload-tile-preview">
                        <Show
                          when={item.preview}
                          fallback={
                            <FileIcon class="lucide-icon" aria-hidden="true" />
                          }
                        >
                          {(preview) =>
                            item.file.type.startsWith("video/") ? (
                              <video src={preview()} muted preload="metadata" />
                            ) : (
                              <img src={preview()} alt={item.file.name} />
                            )
                          }
                        </Show>
                        <Show when={item.status === "done"}>
                          <span class="upload-tile-badge" aria-hidden="true">
                            <CheckIcon class="lucide-icon" />
                          </span>
                        </Show>
                        <Show
                          when={
                            !busy() &&
                            (item.status === "pending" ||
                              item.status === "failed")
                          }
                        >
                          <button
                            type="button"
                            class="upload-tile-remove"
                            aria-label={`Remove ${item.file.name}`}
                            title="Remove"
                            onClick={() => remove(item.id)}
                          >
                            <XIcon class="lucide-icon" aria-hidden="true" />
                          </button>
                        </Show>
                      </div>
                      <span class="upload-tile-name" title={item.file.name}>
                        {item.file.name}
                      </span>
                      <Switch
                        fallback={
                          <span class="upload-tile-meta">
                            {humanizeBytes(item.file.size)}
                          </span>
                        }
                      >
                        <Match when={item.status === "uploading"}>
                          <div
                            class="progress-bar"
                            role="progressbar"
                            aria-label={`Upload progress for ${item.file.name}`}
                            aria-valuenow={item.progress}
                            aria-valuemin={0}
                            aria-valuemax={100}
                          >
                            <div
                              class="progress-fill"
                              style={{ width: `${item.progress}%` }}
                            />
                          </div>
                        </Match>
                        <Match when={item.status === "failed"}>
                          <span class="upload-tile-error">{item.error}</span>
                        </Match>
                        <Match when={item.status === "done" && item.name}>
                          {(name) => (
                            <span class="upload-tile-links">
                              <a href={viewUrl(name())}>View</a>
                              <button
                                type="button"
                                aria-label={`Copy link to ${item.file.name}`}
                                title="Copy link"
                                onClick={() => copyLink(name())}
                              >
                                <LinkIcon
                                  class="lucide-icon"
                                  aria-hidden="true"
                                />
                              </button>
                            </span>
                          )}
                        </Match>
                      </Switch>
                    </li>
                  )}
                </For>
              </ul>
            </Show>
          </div>

          <Input
            label="Expires (optional)"
            id="expiry_date"
            type="date"
            value={expiry()}
            disabled={busy()}
            onInput={(event) => setExpiry(event.currentTarget.value)}
          />
          <button
            type="submit"
            class="upload-button bold"
            disabled={busy() || !context() || toUpload().length === 0}
          >
            {busy()
              ? "Uploading…"
              : toUpload().length
                ? `Upload ${countLabel(toUpload().length, "file")}`
                : "Upload"}
          </button>
        </form>
      </SectionPanel>
    </>
  );
}

import "./UploadPage.css";
import { SectionPanel } from "./SectionPanel";
import { Show, createSignal, onCleanup, onMount } from "solid-js";
import { useFrontendContext } from "../context";
import { uploadFile } from "../generated";
import type { UploadFileData } from "../generated";
import { formDataBodySerializer } from "../generated/client";
import { apiData } from "../client";
import { Input } from "./Input";

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

export function UploadPage() {
  const { context, error: contextError, retry } = useFrontendContext();
  const [host, setHost] = createSignal("");
  const [file, setFile] = createSignal<File>();
  const [preview, setPreview] = createSignal("");
  const [token, setToken] = createSignal("");
  const [expiry, setExpiry] = createSignal("");
  const [busy, setBusy] = createSignal(false);
  const [progress, setProgress] = createSignal(0);
  const [error, setError] = createSignal("");
  const [result, setResult] = createSignal("");
  let input!: HTMLInputElement;
  let uploadController: AbortController | undefined;
  onMount(() => setHost(window.location.host));
  onCleanup(() => {
    uploadController?.abort();
    if (preview()) URL.revokeObjectURL(preview());
  });
  function selectFile(selected?: File) {
    if (busy() || !selected) return;
    if (preview()) URL.revokeObjectURL(preview());
    setFile(selected);
    setPreview(URL.createObjectURL(selected));
    setResult("");
    setError("");
  }
  async function submit(event: SubmitEvent) {
    event.preventDefault();
    const selected = file();
    if (busy() || !selected) {
      if (!selected) setError("Choose a file to upload.");
      return;
    }
    setBusy(true);
    setProgress(0);
    setError("");
    setResult("");
    uploadController = new AbortController();
    try {
      const body = {
        file: selected,
        plain: true,
        expiry_date: expiry() || undefined,
        upload_token: context()?.logged_in ? undefined : token(),
      };
      const name = await apiData(
        uploadFile({
          body,
          signal: uploadController.signal,
          fetch: uploadTransport(body, setProgress),
        }),
      );
      const fileName = name.trim().replace(/^\//, "");
      setResult(
        context()?.logged_in
          ? `/gallery?file=${encodeURIComponent(fileName)}`
          : `/${encodeURIComponent(fileName)}`,
      );
    } catch (failure) {
      setError(failure instanceof Error ? failure.message : "Upload failed.");
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
      <code>
        curl -X POST -F 'upload_token=YOUR_TOKEN' -F 'file=@yourfile.png'{" "}
        {host() ? window.location.protocol : "https:"}//{host()}/api/file/upload
      </code>
      <SectionPanel>
        <Show when={contextError()}>
          <p role="alert">
            {contextError()} <button onClick={retry}>Retry</button>
          </p>
        </Show>
        <form class="upload-form" onSubmit={submit}>
          <Show when={!context()?.logged_in}>
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
          <div class="form-group">
            <input
              ref={input}
              type="file"
              id="file"
              class="file-input"
              disabled={busy()}
              onChange={(event) => selectFile(event.currentTarget.files?.[0])}
            />
            <button
              type="button"
              class="file-drop-zone"
              classList={{ disabled: busy(), "file-selected": !!file() }}
              disabled={busy()}
              onClick={() => input.click()}
              onDragOver={(event) => event.preventDefault()}
              onDrop={(event) => {
                event.preventDefault();
                selectFile(event.dataTransfer?.files[0]);
              }}
            >
              <div class="drop-content">
                <span class="drop-text">Drop file here or click to select</span>
                <Show when={file()}>
                  {(selected) => (
                    <div class="file-preview" style={{ display: "flex" }}>
                      <div class="preview-content">
                        <Show when={selected().type.startsWith("image/")}>
                          <img src={preview()} alt={selected().name} />
                        </Show>
                        <Show when={selected().type.startsWith("video/")}>
                          <video src={preview()} />
                        </Show>
                        <Show when={selected().type.startsWith("audio/")}>
                          <span>Audio file</span>
                        </Show>
                      </div>
                      <div class="file-name bold">{selected().name}</div>
                    </div>
                  )}
                </Show>
              </div>
            </button>
          </div>
          <Input
            label="Expires (optional)"
            id="expiry_date"
            type="date"
            value={expiry()}
            disabled={busy()}
            onInput={(event) => setExpiry(event.currentTarget.value)}
          />
          <Show when={busy()}>
            <div
              class="upload-progress"
              style={{ display: "flex" }}
              role="progressbar"
              aria-label="Upload progress"
              aria-valuenow={progress()}
              aria-valuemin={0}
              aria-valuemax={100}
            >
              <div class="progress-bar">
                <div
                  class="progress-fill"
                  style={{ width: `${progress()}%` }}
                />
              </div>
              <span class="progress-text">{progress()}%</span>
            </div>
          </Show>
          <Show when={error()}>
            <p role="alert">{error()}</p>
          </Show>
          <Show when={result()}>
            <div class="upload-result" style={{ display: "flex" }}>
              <span>Upload successful</span>
              <a href={result()}>View file</a>
            </div>
          </Show>
          <button
            type="submit"
            class="upload-button bold"
            disabled={busy() || !context()}
          >
            Upload File
          </button>
        </form>
      </SectionPanel>
    </>
  );
}

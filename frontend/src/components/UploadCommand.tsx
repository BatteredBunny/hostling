import { For, Show, createSignal, onCleanup } from "solid-js";
import CheckIcon from "@lucide/solid/icons/check";
import CopyIcon from "@lucide/solid/icons/copy";
import { showToast } from "../toast";
import { copyText } from "../utils";
import "./UploadCommand.css";

const PLACEHOLDERS = new Set(["YOUR_TOKEN", "yourfile.png"]);

export function UploadCommand() {
  // plain=true makes the server answer with the file's link instead of a
  // redirect, which is what a terminal wants to print.
  const lines = [
    ["curl -X POST -F 'upload_token=", "YOUR_TOKEN", "' \\"],
    ["  -F 'plain=true' -F 'file=@", "yourfile.png", "' \\"],
    [`  ${window.location.origin}/api/file/upload`],
  ];
  const command = lines.map((parts) => parts.join("")).join("\n");

  const [copied, setCopied] = createSignal(false);
  let copiedTimeout: ReturnType<typeof setTimeout> | undefined;
  onCleanup(() => clearTimeout(copiedTimeout));
  const copy = async () => {
    try {
      await copyText(command);
    } catch {
      showToast("Could not copy the command.", "error");
      return;
    }
    showToast("Command copied.");
    setCopied(true);
    clearTimeout(copiedTimeout);
    copiedTimeout = setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div class="upload-command">
      <pre>
        <code>
          <For each={lines}>
            {(parts, line) => (
              <>
                <For each={parts}>
                  {(part) =>
                    PLACEHOLDERS.has(part) ? (
                      <span class="upload-command-placeholder">{part}</span>
                    ) : (
                      part
                    )
                  }
                </For>
                {line() < lines.length - 1 ? "\n" : ""}
              </>
            )}
          </For>
        </code>
      </pre>
      <button
        type="button"
        class="upload-command-copy"
        aria-label="Copy command"
        title="Copy command"
        onClick={copy}
      >
        <Show
          when={copied()}
          fallback={<CopyIcon class="lucide-icon" aria-hidden="true" />}
        >
          <CheckIcon class="lucide-icon" aria-hidden="true" />
        </Show>
      </button>
    </div>
  );
}

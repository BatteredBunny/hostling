import {
  For,
  Show,
  createEffect,
  createMemo,
  createSignal,
  onCleanup,
} from "solid-js";
import "./TagEditor.css";
import { TagNameSchema, UploadTagsSchema } from "../generated/schemas.gen";
import PlusIcon from "lucide-solid/icons/plus";
import TagIcon from "lucide-solid/icons/tag";
import XIcon from "lucide-solid/icons/x";
import { TagColorDot } from "./TagColorDot";
import { NativePopover } from "./NativePopover";
import { cx } from "../utils";

// Show the "n/max" counter only once a file gets close to the limit.
const TAG_MAX_LENGTH = TagNameSchema.maxLength;
const MAX_TAGS_PER_FILE = UploadTagsSchema.maxItems;
const COUNTER_THRESHOLD = MAX_TAGS_PER_FILE - 5;

interface TagEditorProps {
  tags: string[];
  // Every tag the account has, used for suggestions.
  allTags: string[];
  // Both resolve to an error message, or null on success.
  onAdd: (tag: string) => Promise<string | null>;
  onRemove: (tag: string) => Promise<string | null>;
}

type Option = { kind: "create" | "existing"; tag: string };

// Mirrors the server, which trims and lowercases tags.
const normalizeTag = (value: string) => value.trim().toLowerCase();

// Holds the state of one file's editing session; remount it for another file.
export function TagEditor(props: TagEditorProps) {
  const [input, setInputValue] = createSignal("");
  const [open, setOpen] = createSignal(false);
  const [highlighted, setHighlighted] = createSignal(0);
  const [error, setError] = createSignal<string | null>(null);
  // Set by the first Backspace on an empty input; the second one removes it.
  const [pendingRemoval, setPendingRemoval] = createSignal<string | null>(null);
  const [busy, setBusy] = createSignal(false);

  let inputEl: HTMLInputElement | undefined;
  let listEl: HTMLDivElement | undefined;
  let fieldEl: HTMLDivElement | undefined;
  // Requests may finish after the user has moved on to another file.
  let disposed = false;
  onCleanup(() => {
    disposed = true;
  });

  // New text means new options, so the highlight starts over.
  const setInput = (value: string) => {
    setInputValue(value);
    setHighlighted(0);
  };

  const atLimit = () => props.tags.length >= MAX_TAGS_PER_FILE;

  // Typed text first (exact match or "Create"), then prefix matches,
  // then substring matches, alphabetical within each group.
  const options = createMemo<Option[]>(() => {
    const query = normalizeTag(input());
    const available = props.allTags.filter((t) => !props.tags.includes(t));

    if (!query) {
      return [...available].sort().map((tag) => ({ kind: "existing", tag }));
    }

    const result: Option[] = [];
    if (available.includes(query)) {
      result.push({ kind: "existing", tag: query });
    } else if (!props.tags.includes(query)) {
      result.push({ kind: "create", tag: query });
    }
    const prefix = available
      .filter((t) => t !== query && t.startsWith(query))
      .sort();
    const contains = available
      .filter((t) => !t.startsWith(query) && t.includes(query))
      .sort();
    for (const tag of [...prefix, ...contains])
      result.push({ kind: "existing", tag });
    return result;
  });

  const showList = () => open() && !atLimit() && options().length > 0;
  // Options can also shrink when tags change, so keep the highlight in range.
  const activeIndex = () =>
    Math.max(0, Math.min(highlighted(), options().length - 1));

  // Keep the keyboard-highlighted option scrolled into view.
  createEffect(() => {
    if (!showList()) return;
    listEl?.children[activeIndex()]?.scrollIntoView({ block: "nearest" });
  });

  const add = async (tag: string) => {
    if (busy() || !tag) return;
    if (Array.from(tag).length > TAG_MAX_LENGTH) {
      setError(`Tags can be at most ${TAG_MAX_LENGTH} characters.`);
      return;
    }
    if (props.tags.includes(tag)) {
      setInput("");
      return;
    }
    setBusy(true);
    try {
      const err = await props.onAdd(tag);
      if (disposed) return;
      setError(err);
      if (!err) setInput("");
    } finally {
      setBusy(false);
      if (!disposed) inputEl?.focus();
    }
  };

  const remove = async (tag: string) => {
    setPendingRemoval(null);
    const err = await props.onRemove(tag);
    if (!disposed) setError(err);
  };

  const onKeyDown = (e: KeyboardEvent) => {
    const list = options();

    if (e.key === "Backspace" && input() === "" && props.tags.length > 0) {
      e.preventDefault();
      const last = props.tags[props.tags.length - 1];
      if (pendingRemoval() === last) remove(last);
      else setPendingRemoval(last);
      return;
    }
    setPendingRemoval(null);

    if (e.key === "ArrowDown") {
      e.preventDefault();
      if (!open()) setOpen(true);
      else if (list.length) setHighlighted((activeIndex() + 1) % list.length);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      if (list.length)
        setHighlighted((activeIndex() - 1 + list.length) % list.length);
    } else if (e.key === "Enter") {
      e.preventDefault();
      const option = showList() ? list[activeIndex()] : undefined;
      add(option?.tag ?? normalizeTag(input()));
    } else if (e.key === "Escape" && showList()) {
      // Escape's default action asks the modal dialog to close, so cancel it;
      // the native listener also stops the modal's own key handling.
      e.preventDefault();
      e.stopPropagation();
      setOpen(false);
    }
  };

  const counterVisible = () => props.tags.length >= COUNTER_THRESHOLD;

  return (
    <div class="tag-editor">
      <div class="tag-editor-selected" aria-label="File tags">
        <For each={props.tags}>
          {(tag) => (
            <div
              class={cx("tag-editor-selected-row", {
                pending: pendingRemoval() === tag,
              })}
            >
              <TagIcon class="lucide-icon" aria-hidden="true" />
              <TagColorDot name={tag} />
              <span class="tag-editor-name">{tag}</span>
              <button
                type="button"
                class="tag-editor-remove"
                aria-label={`Remove tag ${tag}`}
                onClick={() => remove(tag)}
              >
                <XIcon class="lucide-icon" aria-hidden="true" />
              </button>
            </div>
          )}
        </For>
      </div>
      <div
        ref={fieldEl}
        class={cx("tag-editor-field", { disabled: atLimit() })}
        onMouseDown={(e) => {
          // Clicking the field's empty space focuses the input.
          if (e.target === e.currentTarget) {
            e.preventDefault();
            inputEl?.focus();
          }
        }}
      >
        <input
          ref={(el) => {
            inputEl = el;
            // Native rather than delegated, so stopPropagation runs before
            // the modal's own key handling sees the event.
            el.addEventListener("keydown", onKeyDown);
          }}
          type="text"
          id="file-modal-tag-input"
          role="combobox"
          aria-label="Add tag"
          aria-expanded={showList()}
          aria-controls="tag-editor-options"
          aria-autocomplete="list"
          placeholder={
            atLimit()
              ? "Tag limit reached"
              : props.tags.length
                ? "Add tag…"
                : "Add a tag…"
          }
          disabled={atLimit()}
          value={input()}
          onInput={(e) => {
            setInput(e.currentTarget.value);
            setError(null);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          onBlur={() => {
            setOpen(false);
            setPendingRemoval(null);
          }}
        />
      </div>

      {/* In the top layer, so the modal's scrolling panels can't clip it. */}
      <NativePopover
        id="tag-editor-options"
        open={showList()}
        manual
        anchor={() => fieldEl}
        onOpenChange={() => {}}
        ref={(element) => {
          listEl = element;
        }}
        class="tag-editor-options"
        role="listbox"
        label="Tag suggestions"
      >
        <For each={options()}>
          {(option, index) => (
            <div
              class={cx("tag-editor-option", {
                active: index() === activeIndex(),
                create: option.kind === "create",
              })}
              role="option"
              aria-selected={index() === activeIndex()}
              // mousedown + preventDefault keeps focus in the input.
              onMouseDown={(e) => {
                e.preventDefault();
                add(option.tag);
              }}
              onMouseEnter={() => setHighlighted(index())}
            >
              <Show
                when={option.kind === "create"}
                fallback={<TagIcon class="lucide-icon" aria-hidden="true" />}
              >
                <PlusIcon class="lucide-icon" aria-hidden="true" />
              </Show>
              <TagColorDot name={option.tag} />
              <span>
                <Show when={option.kind === "create"}>Create </Show>
                <Show when={option.kind === "create"} fallback={option.tag}>
                  <strong>"{option.tag}"</strong>
                </Show>
              </span>
            </div>
          )}
        </For>
      </NativePopover>

      <Show when={error() || counterVisible()}>
        <div class="tag-editor-footer">
          <span class="tag-editor-error" role="alert">
            {error()}
          </span>
          <Show when={counterVisible()}>
            <span class={cx("tag-editor-counter", { full: atLimit() })}>
              {props.tags.length}/{MAX_TAGS_PER_FILE}
            </span>
          </Show>
        </div>
      </Show>
    </div>
  );
}

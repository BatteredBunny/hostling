import { For, Show, createEffect, createMemo, createSignal, on } from 'solid-js';
import './TagEditor.css';
import { TAG_MAX_LENGTH, MAX_TAGS_PER_FILE } from '../api';
import { Icon } from './Icon';
import { TagColorDot } from './TagColorDot';

// Show the "n/max" counter only once a file gets close to the limit.
const COUNTER_THRESHOLD = MAX_TAGS_PER_FILE - 5;

interface TagEditorProps {
  tags: string[];
  // Every tag the account has, used for suggestions.
  allTags: string[];
  // Changing this (e.g. switching files) clears the input and any error.
  resetKey: string;
  // Both resolve to an error message, or null on success.
  onAdd: (tag: string) => Promise<string | null>;
  onRemove: (tag: string) => Promise<string | null>;
}

type Option = { kind: 'create' | 'existing'; tag: string };

// Mirrors the server, which trims and lowercases tags.
const normalizeTag = (value: string) => value.trim().toLowerCase();

export function TagEditor(props: TagEditorProps) {
  const [input, setInput] = createSignal('');
  const [open, setOpen] = createSignal(false);
  const [activeIndex, setActiveIndex] = createSignal(0);
  const [error, setError] = createSignal<string | null>(null);
  // Set by the first Backspace on an empty input; the second one removes it.
  const [pendingRemoval, setPendingRemoval] = createSignal<string | null>(null);
  const [busy, setBusy] = createSignal(false);

  let inputEl: HTMLInputElement | undefined;
  let listEl: HTMLDivElement | undefined;

  createEffect(on(() => props.resetKey, () => {
    setInput('');
    setOpen(false);
    setError(null);
    setPendingRemoval(null);
  }, { defer: true }));

  const atLimit = () => props.tags.length >= MAX_TAGS_PER_FILE;

  // Typed text first (exact match or "Create"), then prefix matches,
  // then substring matches, alphabetical within each group.
  const options = createMemo<Option[]>(() => {
    const query = normalizeTag(input());
    const available = props.allTags.filter((t) => !props.tags.includes(t));

    if (!query) {
      return [...available].sort().map((tag) => ({ kind: 'existing', tag }));
    }

    const result: Option[] = [];
    if (available.includes(query)) {
      result.push({ kind: 'existing', tag: query });
    } else if (!props.tags.includes(query)) {
      result.push({ kind: 'create', tag: query });
    }
    const prefix = available.filter((t) => t !== query && t.startsWith(query)).sort();
    const contains = available.filter((t) => !t.startsWith(query) && t.includes(query)).sort();
    for (const tag of [...prefix, ...contains]) result.push({ kind: 'existing', tag });
    return result;
  });

  const showList = () => open() && !atLimit() && options().length > 0;

  createEffect(on(options, () => setActiveIndex(0)));

  // Keep the keyboard-highlighted option scrolled into view.
  createEffect(() => {
    if (!showList()) return;
    listEl?.children[activeIndex()]?.scrollIntoView({ block: 'nearest' });
  });

  const add = async (tag: string) => {
    if (busy() || !tag) return;
    if (props.tags.includes(tag)) {
      setInput('');
      return;
    }
    setBusy(true);
    const key = props.resetKey;
    try {
      const err = await props.onAdd(tag);
      // The user may have moved to another file meanwhile.
      if (key !== props.resetKey) return;
      setError(err);
      if (!err) setInput('');
    } finally {
      setBusy(false);
      inputEl?.focus();
    }
  };

  const remove = async (tag: string) => {
    setPendingRemoval(null);
    const key = props.resetKey;
    const err = await props.onRemove(tag);
    if (key === props.resetKey) setError(err);
  };

  const onKeyDown = (e: KeyboardEvent) => {
    const list = options();

    if (e.key === 'Backspace' && input() === '' && props.tags.length > 0) {
      e.preventDefault();
      const last = props.tags[props.tags.length - 1];
      if (pendingRemoval() === last) remove(last);
      else setPendingRemoval(last);
      return;
    }
    setPendingRemoval(null);

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (!open()) setOpen(true);
      else if (list.length) setActiveIndex((i) => (i + 1) % list.length);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (list.length) setActiveIndex((i) => (i - 1 + list.length) % list.length);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      const option = showList() ? list[activeIndex()] : undefined;
      add(option?.tag ?? normalizeTag(input()));
    } else if (e.key === 'Escape' && showList()) {
      // Native listener on the input, so this keeps the modal from closing.
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
            <div class="tag-editor-selected-row" classList={{ pending: pendingRemoval() === tag }}>
              <Icon name="tag" />
              <TagColorDot name={tag} />
              <span class="tag-editor-name">{tag}</span>
              <button type="button" class="tag-editor-remove" aria-label={`Remove tag ${tag}`} onClick={() => remove(tag)}>
                <Icon name="x" />
              </button>
            </div>
          )}
        </For>
      </div>
      <div
        class="tag-editor-field"
        classList={{ disabled: atLimit() }}
        onMouseDown={(e) => {
          // Clicking the field's empty space focuses the input.
          if (e.target === e.currentTarget) {
            e.preventDefault();
            inputEl?.focus();
          }
        }}
      >
        <input
          ref={inputEl}
          type="text"
          id="file-modal-tag-input"
          role="combobox"
          aria-label="Add tag"
          aria-expanded={showList()}
          aria-controls="tag-editor-options"
          aria-autocomplete="list"
          placeholder={atLimit() ? 'Tag limit reached' : props.tags.length ? 'Add tag…' : 'Add a tag…'}
          maxLength={TAG_MAX_LENGTH}
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
          on:keydown={onKeyDown}
        />
      </div>

      <Show when={showList()}>
        <div class="tag-editor-options" id="tag-editor-options" role="listbox" ref={listEl}>
          <For each={options()}>
            {(option, index) => (
              <div
                class="tag-editor-option"
                classList={{ active: index() === activeIndex(), create: option.kind === 'create' }}
                role="option"
                aria-selected={index() === activeIndex()}
                // mousedown + preventDefault keeps focus in the input.
                onMouseDown={(e) => {
                  e.preventDefault();
                  add(option.tag);
                }}
                onMouseEnter={() => setActiveIndex(index())}
              >
                <Show when={option.kind === 'create'} fallback={<Icon name="tag" />}>
                  <Icon name="plus" />
                </Show>
                <TagColorDot name={option.tag} />
                <span>
                  <Show when={option.kind === 'create'}>Create </Show>
                  <Show when={option.kind === 'create'} fallback={option.tag}>
                    <strong>"{option.tag}"</strong>
                  </Show>
                </span>
              </div>
            )}
          </For>
        </div>
      </Show>

      <Show when={error() || counterVisible()}>
        <div class="tag-editor-footer">
          <span class="tag-editor-error" role="alert">{error()}</span>
          <Show when={counterVisible()}>
            <span class="tag-editor-counter" classList={{ full: atLimit() }}>
              {props.tags.length}/{MAX_TAGS_PER_FILE}
            </span>
          </Show>
        </div>
      </Show>
    </div>
  );
}

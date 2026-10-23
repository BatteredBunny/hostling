import { For, Show, batch, createEffect, createMemo, createResource, createSignal, onCleanup, onMount } from 'solid-js';
import { Portal } from 'solid-js/web';
import type { JSX } from 'solid-js';
import './FileFilters.css';
import {
  sortField,
  setSortField,
  sortDesc,
  setSortDesc,
  tagFilters,
  setTagFilters,
  visibilityFilter,
  setVisibilityFilter,
  untaggedFilter,
  setUntaggedFilter,
  setCurrentPage,
  activeFilters,
} from '../store';
import { loadFiles } from './FileGrid';
import { Icon } from './Icon';
import { SelectBox } from './SelectBox';
import { fetchFilterTags } from '../api';
import { TagColorDot } from './TagColorDot';
import type { SortField } from '../types';
import type { Visibility } from '../url';

// Keep in sync with the bottom-sheet media query in FileFilters.css.
const SHEET_QUERY = '(max-width: 640px)';

const VISIBILITY_OPTIONS: { value: Visibility | null; label: string }[] = [
  { value: null, label: 'All' },
  { value: 'public', label: 'Public' },
  { value: 'private', label: 'Private' },
];

function reload() {
  setCurrentPage(0);
  loadFiles(0);
}

function setVisibility(value: Visibility | null) {
  if (visibilityFilter() === value) return;
  setVisibilityFilter(value);
  reload();
}

// Untagged and a specific tag are the only mutually exclusive pair:
// picking one clears the other. Visibility combines with either.
function toggleUntagged() {
  batch(() => {
    const next = !untaggedFilter();
    setUntaggedFilter(next);
    if (next) setTagFilters([]);
  });
  reload();
}

function toggleTag(tag: string) {
  batch(() => {
    const selected = tagFilters().includes(tag);
    setTagFilters((tags) => selected ? tags.filter((t) => t !== tag) : [...tags, tag]);
    if (!selected) setUntaggedFilter(false);
  });
  reload();
}

function clearFilters() {
  batch(() => {
    setTagFilters([]);
    setVisibilityFilter(null);
    setUntaggedFilter(false);
  });
  reload();
}

const activeFilterCount = () =>
  Number(!!visibilityFilter()) + Number(untaggedFilter()) + tagFilters().length;

export function FileFilters() {
  const [open, setOpen] = createSignal(false);
  const [position, setPosition] = createSignal({ top: 0, left: 0 });
  let button: HTMLButtonElement | undefined;
  let popover: HTMLDivElement | undefined;

  const openPopover = () => {
    // Rendered in a portal: the setting-group's overflow/backdrop-filter
    // would otherwise clip it, so it's placed against the viewport.
    const rect = button!.getBoundingClientRect();
    const margin = 8;
    const width = 300;
    setPosition({
      top: rect.bottom + 6,
      // Right-aligned with the button, which sits at the toolbar's right edge.
      left: Math.max(margin, Math.min(rect.right - width, window.innerWidth - width - margin)),
    });
    setOpen(true);
  };

  const close = (refocus = true) => {
    setOpen(false);
    if (refocus) button?.focus();
  };

  createEffect(() => {
    if (!open()) return;
    const onPointerDown = (e: PointerEvent) => {
      const target = e.target as Node;
      if (!popover?.contains(target) && !button?.contains(target)) close(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close();
    };
    // A viewport-anchored popover would drift from its button.
    const onScroll = (e: Event) => {
      if (!popover?.contains(e.target as Node)) close(false);
    };
    // On phones it's a bottom sheet, and the on-screen keyboard resizes the viewport.
    const onResize = () => {
      if (!window.matchMedia(SHEET_QUERY).matches) close(false);
    };
    document.addEventListener('pointerdown', onPointerDown);
    document.addEventListener('keydown', onKey);
    window.addEventListener('scroll', onScroll, true);
    window.addEventListener('resize', onResize);
    onCleanup(() => {
      document.removeEventListener('pointerdown', onPointerDown);
      document.removeEventListener('keydown', onKey);
      window.removeEventListener('scroll', onScroll, true);
      window.removeEventListener('resize', onResize);
    });
  });

  const handleSortChange = (value: string) => {
    const [field, order] = value.split(':') as [SortField, string];
    setSortField(field);
    setSortDesc(order === 'desc');
    reload();
  };

  return (
    <div class="file-filters">
      <div class="file-filters-toolbar">
        <div class="file-filters-anchor">
          <button
            type="button"
            ref={button}
            class="file-filters-button"
            classList={{ open: open() }}
            aria-haspopup="dialog"
            aria-expanded={open()}
            onClick={() => (open() ? close() : openPopover())}
          >
            <Icon name="sliders-horizontal" />
            <span>Filters</span>
            <Show when={activeFilterCount() > 0}>
              <span class="file-filters-badge">{activeFilterCount()}</span>
            </Show>
          </button>

          <Show when={open()}>
            <Portal>
              <div class="file-filters-backdrop" onClick={() => close(false)} />
              <FiltersPopover
                ref={(el) => (popover = el)}
                style={{ top: `${position().top}px`, left: `${position().left}px` }}
                onClose={close}
              />
            </Portal>
          </Show>
        </div>

        <SelectBox
          id="sort-dropdown"
          aria-label="Sort files"
          value={`${sortField()}:${sortDesc() ? 'desc' : 'asc'}`}
          onChange={handleSortChange}
          options={[
            { value: 'created_at:desc', label: 'Newest First' },
            { value: 'created_at:asc', label: 'Oldest First' },
            { value: 'views:desc', label: 'Most Views' },
            { value: 'views:asc', label: 'Least Views' },
            { value: 'file_size:desc', label: 'Largest First' },
            { value: 'file_size:asc', label: 'Smallest First' },
          ]}
        />
      </div>
    </div>
  );
}

function FiltersPopover(props: {
  ref: (el: HTMLDivElement) => void;
  style: JSX.CSSProperties;
  onClose: (refocus?: boolean) => void;
}) {
  const [query, setQuery] = createSignal('');
  let firstOption: HTMLButtonElement | undefined;

  const [availableTags] = createResource(
    () => ({ tags: [...tagFilters()], filters: activeFilters() }),
    async (selection) => fetchFilterTags(selection.tags, selection.filters),
  );
  const selectedTags = createMemo(() => [...tagFilters()].sort());
  const matches = createMemo(() => {
    const q = query().trim().toLowerCase();
    // Hide the previous selection's results while its replacement loads.
    const available = availableTags.loading || availableTags.error ? [] : (availableTags() ?? []);
    return available.filter((tag) => !tagFilters().includes(tag) && tag.includes(q)).sort();
  });

  onMount(() => firstOption?.focus());

  return (
    <div class="file-filters-popover" role="dialog" aria-label="Filters" ref={props.ref} style={props.style}>
      <div class="file-filters-popover-header">
        <span>Filters</span>
        <button type="button" class="file-filters-close" aria-label="Close filters" onClick={() => props.onClose()}>
          <Icon name="x" />
        </button>
      </div>

      <section>
        <h4 id="file-filters-visibility-label">Visibility</h4>
        <div class="file-filters-segmented" role="radiogroup" aria-labelledby="file-filters-visibility-label">
          <For each={VISIBILITY_OPTIONS}>
            {(option, index) => (
              <button
                type="button"
                ref={(el) => {
                  if (index() === 0) firstOption = el;
                }}
                role="radio"
                aria-checked={visibilityFilter() === option.value}
                onClick={() => setVisibility(option.value)}
              >
                {option.label}
              </button>
            )}
          </For>
        </div>

        <button
          type="button"
          role="checkbox"
          aria-checked={untaggedFilter()}
          class="file-filters-checkbox"
          onClick={toggleUntagged}
        >
          <span class="file-filters-checkbox-box">
            <Icon name="check" />
          </span>
          Untagged only
        </button>
      </section>

      <section>
        <h4>Tags</h4>
        <input
          type="text"
          class="file-filters-search"
          placeholder="Search tags…"
          aria-label="Search tags"
          value={query()}
          onInput={(e) => setQuery(e.currentTarget.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && matches().length > 0) {
              e.preventDefault();
              toggleTag(matches()[0]);
            }
          }}
        />
        <div class="file-filters-tag-list" role="listbox" aria-label="Tags" aria-multiselectable="true" aria-busy={availableTags.loading}>
          <For each={selectedTags()}>
            {(tag) => (
              <button type="button" role="option" class="file-filters-tag-option" aria-selected="true" onClick={() => toggleTag(tag)}>
                <Icon name="check" />
                <TagColorDot name={tag} />
                <span>{tag}</span>
              </button>
            )}
          </For>
          <For each={matches()} fallback={<div class="file-filters-empty" role="status">{availableTags.loading ? 'Loading tags…' : availableTags.error ? 'Could not load tags' : 'No matching tags'}</div>}>
            {(tag) => (
              <button
                type="button"
                role="option"
                class="file-filters-tag-option"
                aria-selected={tagFilters().includes(tag)}
                onClick={() => toggleTag(tag)}
              >
                <Icon name="tag" />
                <TagColorDot name={tag} />
                <span>{tag}</span>
              </button>
            )}
          </For>
        </div>
      </section>

      <div class="file-filters-popover-footer">
        <button
          type="button"
          class="file-filters-clear"
          disabled={activeFilterCount() === 0}
          onClick={clearFilters}
        >
          Clear all filters
        </button>
      </div>
    </div>
  );
}

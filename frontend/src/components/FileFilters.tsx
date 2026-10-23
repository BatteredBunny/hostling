import {
  For,
  Show,
  batch,
  createMemo,
  createResource,
  createSignal,
  createUniqueId,
} from "solid-js";
import { NativePopover } from "./NativePopover";
import "./FileFilters.css";
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
} from "../store";
import { loadFiles } from "./FileGrid";
import {
  Check as CheckIcon,
  SlidersHorizontal as SlidersHorizontalIcon,
  Tag as TagIcon,
  X as XIcon,
} from "lucide-solid";
import { SelectBox } from "./SelectBox";
import { getFilterTags } from "../generated";
import { apiData } from "../client";
import { TagColorDot } from "./TagColorDot";
import type { SortField } from "../types";
import type { Visibility } from "../url";

const VISIBILITY_OPTIONS: { value: Visibility | null; label: string }[] = [
  { value: null, label: "All" },
  { value: "public", label: "Public" },
  { value: "private", label: "Private" },
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
    setTagFilters((tags) =>
      selected ? tags.filter((t) => t !== tag) : [...tags, tag],
    );
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
  const popoverId = createUniqueId();
  let button: HTMLButtonElement | undefined;
  let popover: HTMLDivElement | undefined;

  const close = (refocus = true) => {
    setOpen(false);
    if (refocus) button?.focus();
  };

  const handleSortChange = (value: string) => {
    const [field, order] = value.split(":") as [SortField, string];
    setSortField(field);
    setSortDesc(order === "desc");
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
            aria-controls={popoverId}
            onClick={() => setOpen(!open())}
          >
            <SlidersHorizontalIcon class="lucide-icon" aria-hidden="true" />
            <span>Filters</span>
            <Show when={activeFilterCount() > 0}>
              <span class="file-filters-badge">{activeFilterCount()}</span>
            </Show>
          </button>

          <NativePopover
            id={popoverId}
            open={open()}
            anchor={() => button}
            onOpenChange={setOpen}
            role="dialog"
            label="Filters"
            class="file-filters-popover"
            ref={(el) => (popover = el)}
            onOpen={() =>
              popover
                ?.querySelector<HTMLButtonElement>("button[role=radio]")
                ?.focus()
            }
          >
            <Show when={open()}>
              <FiltersPopover onClose={close} />
            </Show>
          </NativePopover>
        </div>

        <SelectBox
          id="sort-dropdown"
          aria-label="Sort files"
          value={`${sortField()}:${sortDesc() ? "desc" : "asc"}`}
          onChange={handleSortChange}
          options={[
            { value: "created_at:desc", label: "Newest First" },
            { value: "created_at:asc", label: "Oldest First" },
            { value: "views:desc", label: "Most Views" },
            { value: "views:asc", label: "Least Views" },
            { value: "file_size:desc", label: "Largest First" },
            { value: "file_size:asc", label: "Smallest First" },
          ]}
        />
      </div>
    </div>
  );
}

function FiltersPopover(props: { onClose: (refocus?: boolean) => void }) {
  const [query, setQuery] = createSignal("");

  const [availableTags] = createResource(
    () => ({ tags: [...tagFilters()], filters: activeFilters() }),
    async (selection) =>
      (
        await apiData(
          getFilterTags({
            query: { tag: selection.tags, filter: selection.filters },
          }),
        )
      ).tags ?? [],
  );
  const selectedTags = createMemo(() => [...tagFilters()].sort());
  const matches = createMemo(() => {
    const q = query().trim().toLowerCase();
    // Hide the previous selection's results while its replacement loads.
    const available =
      availableTags.loading || availableTags.error
        ? []
        : (availableTags() ?? []);
    return available
      .filter((tag) => !tagFilters().includes(tag) && tag.includes(q))
      .sort();
  });

  return (
    <>
      <div class="file-filters-popover-header">
        <span>Filters</span>
        <button
          type="button"
          class="file-filters-close"
          aria-label="Close filters"
          onClick={() => props.onClose()}
        >
          <XIcon class="lucide-icon" aria-hidden="true" />
        </button>
      </div>

      <section>
        <h4 id="file-filters-visibility-label">Visibility</h4>
        <div
          class="file-filters-segmented"
          role="radiogroup"
          aria-labelledby="file-filters-visibility-label"
        >
          <For each={VISIBILITY_OPTIONS}>
            {(option) => (
              <button
                type="button"
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
            <CheckIcon class="lucide-icon" aria-hidden="true" />
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
            if (e.key === "Enter" && matches().length > 0) {
              e.preventDefault();
              toggleTag(matches()[0]);
            }
          }}
        />
        <div
          class="file-filters-tag-list"
          role="listbox"
          aria-label="Tags"
          aria-multiselectable="true"
          aria-busy={availableTags.loading}
        >
          <For each={selectedTags()}>
            {(tag) => (
              <button
                type="button"
                role="option"
                class="file-filters-tag-option"
                aria-selected="true"
                onClick={() => toggleTag(tag)}
              >
                <CheckIcon class="lucide-icon" aria-hidden="true" />
                <TagColorDot name={tag} />
                <span>{tag}</span>
              </button>
            )}
          </For>
          <For
            each={matches()}
            fallback={
              <div class="file-filters-empty" role="status">
                {availableTags.loading
                  ? "Loading tags…"
                  : availableTags.error
                    ? "Could not load tags"
                    : "No matching tags"}
              </div>
            }
          >
            {(tag) => (
              <button
                type="button"
                role="option"
                class="file-filters-tag-option"
                aria-selected={tagFilters().includes(tag)}
                onClick={() => toggleTag(tag)}
              >
                <TagIcon class="lucide-icon" aria-hidden="true" />
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
    </>
  );
}

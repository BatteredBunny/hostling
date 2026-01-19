import { For, onMount, Show } from 'solid-js';
import './FileStats.css';
import { statsCount, statsSizeTotal, statsTags, setStatsCount, setStatsSizeTotal, setStatsTags, tagFilter, setTagFilter, setFileFilter, setCurrentPage, fileFilter } from '../store';
import { fetchFileStats } from '../api';
import { humanizeBytes } from '../utils';
import { Tag } from './Tag';
import { loadFiles } from './FileGrid';

export function FileStats() {
  onMount(async () => {
    await loadStats();
  });

  return (
    <>
      <div>
        <h3>Statistics</h3>
        <p id="files-stats">
          {statsCount() === 1 ? '1 file' : `${statsCount()} files`} • {humanizeBytes(statsSizeTotal())}
        </p>
      </div>
      <Show when={statsTags().length > 0}>
        <div>
          <h3>Tags</h3>
          <div id="files-stats-tags">
            <For each={statsTags()}>{(tag) =>
              <Tag
                name={tag}
                enabled={fileFilter() !== 'untagged'}
                onClicked={() => {
                  const newTag = tagFilter() === tag ? null : tag;
                  setTagFilter(newTag);
                  if (newTag) {
                    setFileFilter(null); // Clear untagged filter when selecting a tag
                  }
                  setCurrentPage(0);
                  loadFiles(0);
                }}
                selected={tagFilter() === tag}
              />
            }</For>
          </div>
        </div>
      </Show>
    </>
  );
}

export async function loadStats() {
  try {
    const data = await fetchFileStats();
    setStatsCount(data.count || 0);
    setStatsSizeTotal(data.size_total || 0);
    setStatsTags(data.tags || []);
  } catch {
    console.error('Failed to load stats');
  }
}

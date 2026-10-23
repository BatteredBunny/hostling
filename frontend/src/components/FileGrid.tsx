import { For, Show, batch, onMount } from 'solid-js';
import './FileGrid.css';
import {
  files,
  setFiles,
  totalFiles,
  setTotalFiles,
  currentPage,
  setCurrentPage,
  isLoading,
  setIsLoading,
  loadingText,
  setLoadingText,
  sortField,
  sortDesc,
  tagFilters,
  setTagFilters,
  activeFilters,
  pendingModalFile,
  setPendingModalFile,
  pendingModalEdge,
  setPendingModalEdge,
  modalFile,
  setModalFile,
  openModal,
  markReplaceNextUrlSync,
} from '../store';
import { fetchFiles, FILES_PER_PAGE } from '../api';
import { FileEntry } from './FileEntry';
import { Icon } from './Icon';
import { FileFilters } from './FileFilters';
import { FileStats } from './FileStats';

export function FileGrid() {
  onMount(() => {
    loadFiles(currentPage() * FILES_PER_PAGE);
  });

  const totalPages = () => Math.ceil(totalFiles() / FILES_PER_PAGE);

  const pageInfo = () => {
    const skip = currentPage() * FILES_PER_PAGE;
    const start = skip + 1;
    const end = Math.min(skip + FILES_PER_PAGE, totalFiles());
    return `${start}-${end} of ${totalFiles()}`;
  };

  const handlePrev = () => {
    if (currentPage() > 0) {
      const newPage = currentPage() - 1;
      setCurrentPage(newPage);
      loadFiles(newPage * FILES_PER_PAGE);
    }
  };

  const handleNext = () => {
    if (currentPage() + 1 < totalPages()) {
      const newPage = currentPage() + 1;
      setCurrentPage(newPage);
      loadFiles(newPage * FILES_PER_PAGE);
    }
  };

  return (
    <>
      <div class="setting-group-header">
        <div class="files-top-row">
          <h2>Files</h2>
          <FileStats />
        </div>
      </div>

      <div class="setting-group-body">
        <FileFilters />
        <div class="file-grid" classList={{ loading: isLoading() && files().length > 0 }}>
          <Show when={loadingText()}>
            <p id="file-grid-loading-text">{loadingText()}</p>
          </Show>

          <For each={files()}>
            {(file) => <FileEntry file={file} />}
          </For>
        </div>

        <Show when={isLoading() && files().length > 0}>
          <div id="file-grid-loading-overlay" class="visible">
            <div class="spinner" />
          </div>
        </Show>

        <Show when={totalPages() > 1}>
          <div id="pagination-controls" style={{ display: 'flex' }}>
            <button
              id="prev-page-btn"
              class="pagination-button"
              onClick={handlePrev}
              disabled={currentPage() === 0}
            >
              <Icon name="chevron-left" />
              <span>Previous</span>
            </button>
            <span id="page-info">{pageInfo()}</span>
            <button
              id="next-page-btn"
              class="pagination-button"
              onClick={handleNext}
              disabled={currentPage() + 1 >= totalPages()}
            >
              <span>Next</span>
              <Icon name="chevron-right" />
            </button>
          </div>
        </Show>
      </div>
    </>
  );
}

let inflight: AbortController | null = null;

export async function loadFiles(skip: number) {
  // Abort any in-flight request so rapid filter/sort changes don't race:
  // older responses must not clobber newer ones.
  if (inflight) inflight.abort();
  const controller = new AbortController();
  inflight = controller;

  setIsLoading(true);

  if (files().length === 0) {
    setLoadingText('Loading...');
  }

  try {
    let effectiveSkip = skip;
    let data: Awaited<ReturnType<typeof fetchFiles>> | undefined;

    for (let attempt = 0; attempt < 3; attempt++) {
      data = await fetchFiles(effectiveSkip, sortField(), sortDesc(), tagFilters(), activeFilters(), controller.signal);

      // Tags only exist while some file has them, so an empty result for a
      // single tag alone most likely means it was removed recently. With more
      // tags or other filters, an empty result can be legitimate.
      if (data.files.length === 0 && tagFilters().length === 1 && activeFilters().length === 0) {
        if (controller.signal.aborted) return;
        batch(() => {
          markReplaceNextUrlSync();
          setTagFilters([]);
        });
        data = await fetchFiles(effectiveSkip, sortField(), sortDesc(), [], activeFilters(), controller.signal);
      }

      const count = data.count || 0;
      setTotalFiles(count);

      if (count > 0 && effectiveSkip >= count) {
        const lastPage = Math.max(0, Math.ceil(count / FILES_PER_PAGE) - 1);
        setCurrentPage(lastPage);
        effectiveSkip = lastPage * FILES_PER_PAGE;
        continue;
      }
      break;
    }

    if (!data) return;

    setFiles(data.files || []);

    const pending = pendingModalFile();
    if (pending && data.files) {
      const match = data.files.find((f) => f.FileName === pending);
      if (match) {
        openModal(match);
        setPendingModalFile(null);
      }
      // If not matched, keep pendingModalFile so the URL ?file=... is preserved
      // until the user navigates to a page/filter where the file is visible.
    }

    const edge = pendingModalEdge();
    if (edge && data.files && data.files.length > 0) {
      setModalFile(edge === 'first' ? data.files[0] : data.files[data.files.length - 1]);
    }
    setPendingModalEdge(null);

    if (data.files && data.files.length > 0) {
      setLoadingText('');
    } else if (tagFilters().length > 0 || activeFilters().length > 0) {
      setLoadingText('No files match these filters.');
    } else {
      setLoadingText('No files uploaded yet.');
    }
  } catch (err) {
    if ((err as Error)?.name === 'AbortError') return;
    setLoadingText('Failed to load files.');
  } finally {
    if (inflight === controller) {
      inflight = null;
      setIsLoading(false);
    }
  }
}
function modalIndex() {
  const name = modalFile()?.FileName;
  return files().findIndex((f) => f.FileName === name);
}

export function hasAdjacentFile(delta: -1 | 1): boolean {
  const index = modalIndex();
  if (index === -1) return false;
  const target = currentPage() * FILES_PER_PAGE + index + delta;
  return target >= 0 && target < totalFiles();
}

// Moves the modal to the previous/next file, loading the adjacent page
// when stepping past either end of the current one.
export function showAdjacentFile(delta: -1 | 1) {
  if (isLoading() || !hasAdjacentFile(delta)) return;

  const target = modalIndex() + delta;
  if (target >= 0 && target < files().length) {
    setModalFile(files()[target]);
    return;
  }

  const newPage = currentPage() + delta;
  setPendingModalEdge(delta === 1 ? 'first' : 'last');
  setCurrentPage(newPage);
  loadFiles(newPage * FILES_PER_PAGE);
}

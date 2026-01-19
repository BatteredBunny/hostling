import { For, Show, onMount } from 'solid-js';
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
  setSortField,
  sortDesc,
  setSortDesc,
  removeFileFromList,
  tagFilter,
  setTagFilter,
  fileFilter,
  setFileFilter,
} from '../store';
import { fetchFiles, deleteFile, FILES_PER_PAGE } from '../api';
import { loadStats } from './FileStats';
import { FileStats } from './FileStats';
import { FileEntry } from './FileEntry';
import { FileModal } from './FileModal';
import { Icon } from './Icon';
import type { SortField } from '../types';

export function FileGrid() {
  onMount(() => {
    loadFiles(0);
  });

  const totalPages = () => Math.ceil(totalFiles() / FILES_PER_PAGE);

  const pageInfo = () => {
    const skip = currentPage() * FILES_PER_PAGE;
    const start = skip + 1;
    const end = Math.min(skip + FILES_PER_PAGE, totalFiles());
    return `${start}-${end} of ${totalFiles()}`;
  };

  const handleSortChange = (e: Event) => {
    const value = (e.target as HTMLSelectElement).value;
    const [field, order] = value.split(':') as [SortField, string];
    setSortField(field);
    setSortDesc(order === 'desc');
    setCurrentPage(0);
    loadFiles(0);
  };

  const handleFilterChange = (e: Event) => {
    const value = (e.target as HTMLSelectElement).value;

    if (value === 'all') {
      setTagFilter(null);
      setFileFilter(null);
    } else if (value === 'untagged') {
      setTagFilter(null);
      setFileFilter('untagged');
    }

    setCurrentPage(0);
    loadFiles(0);
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

  const handleDelete = async (fileName: string) => {
    const success = await deleteFile(fileName);
    if (success) {
      removeFileFromList(fileName);
      loadStats();

      if (files().length === 0 && currentPage() > 0) {
        const newPage = currentPage() - 1;
        setCurrentPage(newPage);
        loadFiles(newPage * FILES_PER_PAGE);
      }
    } else {
      alert('Failed to delete file');
    }
  };

  const currentFilterValue = () => {
    if (fileFilter() === 'untagged') return 'untagged';
    return 'all';
  };

  return (
    <>
      <div class="setting-group-header">
        <div class="files-top-row">
          <h2>Files</h2>
          <div class="options">
            <select
              id="filter-dropdown"
              onChange={handleFilterChange}
              value={currentFilterValue()}
            >
              <option value="all">All Files</option>
              <option value="untagged">Untagged</option>
            </select>
            <select
              id="sort-dropdown"
              onChange={handleSortChange}
              value={`${sortField()}:${sortDesc() ? 'desc' : 'asc'}`}
            >
              <option value="created_at:desc">Newest First</option>
              <option value="created_at:asc">Oldest First</option>
              <option value="views:desc">Most Views</option>
              <option value="views:asc">Least Views</option>
              <option value="file_size:desc">Largest First</option>
              <option value="file_size:asc">Smallest First</option>
            </select>
          </div>
        </div>
      </div>

      <div class="setting-group-body">
        <FileStats />

        <div class="file-grid" classList={{ loading: isLoading() && files().length > 0 }}>
          <Show when={loadingText()}>
            <p id="file-grid-loading-text">{loadingText()}</p>
          </Show>

          <For each={files()}>
            {(file) => <FileEntry file={file} onDelete={handleDelete} />}
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

        <FileModal />
      </div>
    </>
  );
}

export async function loadFiles(skip: number) {
  if (isLoading()) return;

  setIsLoading(true);

  if (files().length === 0) {
    setLoadingText('Loading...');
  }

  try {
    let data = await fetchFiles(skip, sortField(), sortDesc(), tagFilter(), fileFilter());

    // Tags can only be filtered if there are files with that tag
    // If it returns nothing it most likely means the tag was removed recently
    if (data.files.length === 0 && tagFilter()) {
      setTagFilter(null);
      data = await fetchFiles(skip, sortField(), sortDesc(), null, fileFilter());
    }

    // If untagged filter returns nothing, clear it
    if (data.files.length === 0 && fileFilter() === 'untagged') {
      setFileFilter(null);
      data = await fetchFiles(skip, sortField(), sortDesc(), tagFilter(), null);
    }

    setTotalFiles(data.count || 0);
    setFiles(data.files || []);

    if (data.files && data.files.length > 0) {
      setLoadingText('');
    } else {
      setLoadingText('No files uploaded yet.');
    }
  } catch {
    setLoadingText('Failed to load files.');
  } finally {
    setIsLoading(false);
  }
}
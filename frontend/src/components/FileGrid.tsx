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
  pendingModalFile,
  setPendingModalFile,
  openModal,
} from '../store';
import { fetchFiles, deleteFile, FILES_PER_PAGE } from '../api';
import { loadStats } from './FileStats';
import { FileEntry } from './FileEntry';
import { Icon } from './Icon';
import { SelectBox } from './SelectBox';
import type { SortField } from '../types';

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

  const handleSortChange = (value: string) => {
    const [field, order] = value.split(':') as [SortField, string];
    setSortField(field);
    setSortDesc(order === 'desc');
    setCurrentPage(0);
    loadFiles(0);
  };

  const handleFilterChange = (value: string) => {
    if (value === 'all') {
      setTagFilter(null);
      setFileFilter(null);
    } else if (value === 'untagged') {
      setTagFilter(null);
      setFileFilter('untagged');
    } else if (value === 'public') {
      setTagFilter(null);
      setFileFilter('public');
    } else if (value === 'private') {
      setTagFilter(null);
      setFileFilter('private');
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
      } else if (tagFilter()) {
        loadFiles(currentPage() * FILES_PER_PAGE);
      }
    } else {
      alert('Failed to delete file');
    }
  };

  const currentFilterValue = () => {
    if (fileFilter() === 'untagged') return 'untagged';
    if (fileFilter() === 'public') return 'public';
    if (fileFilter() === 'private') return 'private';
    return 'all';
  };

  return (
    <>
      <div class="setting-group-header">
        <div class="files-top-row">
          <h2>Files</h2>
          <div class="options">
            <SelectBox
              id="filter-dropdown"
              value={currentFilterValue()}
              onChange={handleFilterChange}
              options={[
                { value: 'all', label: 'All Files' },
                { value: 'untagged', label: 'Untagged' },
                { value: 'public', label: 'Public' },
                { value: 'private', label: 'Private' },
              ]}
            />
            <SelectBox
              id="sort-dropdown"
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
      </div>

      <div class="setting-group-body">
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

    const count = data.count || 0;
    setTotalFiles(count);

    // Clamp page if URL/state references a page beyond what's available.
    if (count > 0 && skip >= count) {
      const lastPage = Math.max(0, Math.ceil(count / FILES_PER_PAGE) - 1);
      setCurrentPage(lastPage);
      setIsLoading(false);
      await loadFiles(lastPage * FILES_PER_PAGE);
      return;
    }

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

    if (data.files && data.files.length > 0) {
      setLoadingText('');
    } else {
      if (tagFilter() || fileFilter()) {
        setLoadingText('No files found.');
      } else {
        setLoadingText('No files uploaded yet.');
      }
    }
  } catch {
    setLoadingText('Failed to load files.');
  } finally {
    setIsLoading(false);
  }
}
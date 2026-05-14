import { createEffect, onCleanup, onMount, Show } from 'solid-js';
import { FileGrid, loadFiles } from './FileGrid';
import { FileStats } from './FileStats';
import { FileModal } from './FileModal';
import {
  currentPage,
  setCurrentPage,
  sortField,
  setSortField,
  sortDesc,
  setSortDesc,
  tagFilter,
  setTagFilter,
  fileFilter,
  setFileFilter,
  modalFile,
  pendingModalFile,
  setPendingModalFile,
  closeModal,
  consumeReplaceNextUrlSync,
} from '../store';
import { updateUrl, parseUrlParams } from '../url';
import { FILES_PER_PAGE } from '../api';

export function FileLibrary() {
  let lastPage = currentPage();
  let lastTag = tagFilter();

  createEffect(() => {
    const page = currentPage();
    const tag = tagFilter();
    // Only push a new history entry on discrete navigation (page / tag).
    // Sort toggles, filter changes, and modal open/close mutate in place.
    const forceReplace = consumeReplaceNextUrlSync();
    const isNavigation = !forceReplace && (page !== lastPage || tag !== lastTag);
    lastPage = page;
    lastTag = tag;

    updateUrl(
      {
        page,
        sort: sortField(),
        desc: sortDesc(),
        tag,
        filter: fileFilter(),
        file: modalFile()?.FileName ?? pendingModalFile() ?? null,
      },
      { replace: !isNavigation },
    );
  });

  onMount(() => {
    const handlePopState = () => {
      const next = parseUrlParams();
      setSortField(next.sort);
      setSortDesc(next.order === 'desc');
      setTagFilter(next.tag);
      setFileFilter(next.filter);
      setCurrentPage(next.page);
      if (next.file) {
        setPendingModalFile(next.file);
      } else {
        closeModal();
        setPendingModalFile(null);
      }
      loadFiles(next.page * FILES_PER_PAGE);
    };

    window.addEventListener('popstate', handlePopState);
    onCleanup(() => window.removeEventListener('popstate', handlePopState));
  });

  return (
    <>
      <setting-group>
        <div class="setting-group-header">
          <h2>Overview</h2>
        </div>
        <div class="setting-group-body">
          <FileStats />
        </div>
      </setting-group>
      <setting-group>
        <FileGrid />
      </setting-group>
      <Show when={modalFile()}>
        {(file) => <FileModal file={file()} />}
      </Show>
    </>
  );
}

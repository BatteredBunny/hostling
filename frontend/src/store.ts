import { createSignal, untrack } from 'solid-js';
import type { FileData, SortField } from './types';
import { parseUrlParams } from './url';

const initial = parseUrlParams();

// Files state
export const [files, setFiles] = createSignal<FileData[]>([]);
export const [totalFiles, setTotalFiles] = createSignal(0);
export const [currentPage, setCurrentPage] = createSignal(initial.page);
export const [isLoading, setIsLoading] = createSignal(false);
export const [loadingText, setLoadingText] = createSignal('');

// Sort/Filter state
export const [sortField, setSortField] = createSignal<SortField>(initial.sort);
export const [sortDesc, setSortDesc] = createSignal(initial.order === 'desc');
export const [tagFilter, setTagFilter] = createSignal<string | null>(initial.tag);
export const [fileFilter, setFileFilter] = createSignal<string | null>(initial.filter);

// Stats state
export const [statsCount, setStatsCount] = createSignal(0);
export const [statsSizeTotal, setStatsSizeTotal] = createSignal(0);
export const [statsTags, setStatsTags] = createSignal<string[]>([]);

// Modal state
export const [modalFile, setModalFile] = createSignal<FileData | null>(null);
export const [isModalOpen, setIsModalOpen] = createSignal(false);
export const [pendingModalFile, setPendingModalFile] = createSignal<string | null>(initial.file);
// Which end of the next loaded page to open in the modal, used when
// stepping through files across a page boundary.
export const [pendingModalEdge, setPendingModalEdge] = createSignal<'first' | 'last' | null>(null);

export function openModal(file: FileData) {
  setModalFile(file);
  setIsModalOpen(true);
}

export function closeModal() {
  setIsModalOpen(false);
  setModalFile(null);
  setPendingModalEdge(null);
}

export function updateFileInList(fileName: string, updates: Partial<FileData>) {
  setFiles((prev) =>
    prev.map((f) => (f.FileName === fileName ? { ...f, ...updates } : f))
  );

  const current = modalFile();
  if (current?.FileName === fileName) {
    setModalFile({ ...current, ...updates });
  }
}

const [replaceNextUrlSync, setReplaceNextUrlSync] = createSignal(false);
export function markReplaceNextUrlSync() {
  setReplaceNextUrlSync(true);
}
export function consumeReplaceNextUrlSync(): boolean {
  const v = replaceNextUrlSync();
  if (v) untrack(() => setReplaceNextUrlSync(false));
  return v;
}

export function removeFileFromList(fileName: string) {
  setFiles((prev) => prev.filter((f) => f.FileName !== fileName));
  setTotalFiles((prev) => Math.max(0, prev - 1));
}

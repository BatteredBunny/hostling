import { createSignal } from 'solid-js';
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

export function openModal(file: FileData) {
  setModalFile(file);
  setIsModalOpen(true);
  document.body.classList.add('no-scroll');
}

export function closeModal() {
  setIsModalOpen(false);
  setModalFile(null);
  document.body.classList.remove('no-scroll');
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

export function removeFileFromList(fileName: string) {
  setFiles((prev) => prev.filter((f) => f.FileName !== fileName));
  setTotalFiles((prev) => prev - 1);
}

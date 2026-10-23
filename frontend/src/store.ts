import { createSignal, untrack } from "solid-js";
import { createStore, reconcile } from "solid-js/store";
import type { FileData, SortField } from "./types";
import { parseUrlParams, type Visibility } from "./url";

const initial = parseUrlParams();

// Files state
const [fileState, setFileState] = createStore<{ files: FileData[] }>({
  files: [],
});
export const files = () => fileState.files;
export function setFiles(
  next: FileData[] | ((previous: FileData[]) => FileData[]),
) {
  const updated = typeof next === "function" ? next(fileState.files) : next;
  // Keep cards and media mounted when metadata changes or a list is refreshed.
  setFileState("files", reconcile(updated, { key: "FileName" }));
}
export const [totalFiles, setTotalFiles] = createSignal(0);
export const [currentPage, setCurrentPage] = createSignal(initial.page);
export const [isLoading, setIsLoading] = createSignal(false);
export const [loadingText, setLoadingText] = createSignal("");

// Sort/Filter state
export const [sortField, setSortField] = createSignal<SortField>(initial.sort);
export const [sortDesc, setSortDesc] = createSignal(initial.order === "desc");
export const [tagFilters, setTagFilters] = createSignal<string[]>(initial.tags);
export const [visibilityFilter, setVisibilityFilter] =
  createSignal<Visibility | null>(initial.visibility);
export const [untaggedFilter, setUntaggedFilter] = createSignal(
  initial.untagged,
);

// Values for the API's repeatable `filter` parameter.
export function activeFilters(): string[] {
  const filters: string[] = [];
  const visibility = visibilityFilter();
  if (visibility) filters.push(visibility);
  if (untaggedFilter()) filters.push("untagged");
  return filters;
}

// Stats state
export const [statsCount, setStatsCount] = createSignal(0);
export const [statsSizeTotal, setStatsSizeTotal] = createSignal(0);
export const [statsTags, setStatsTags] = createSignal<string[]>([]);

// Modal state
export const [modalFile, setModalFile] = createSignal<FileData | null>(null);
export const [isModalOpen, setIsModalOpen] = createSignal(false);
export const [pendingModalFile, setPendingModalFile] = createSignal<
  string | null
>(initial.file);
// Which end of the next loaded page to open in the modal, used when
// stepping through files across a page boundary.
export const [pendingModalEdge, setPendingModalEdge] = createSignal<
  "first" | "last" | null
>(null);

export function openModal(file: FileData) {
  setModalFile(file);
  setIsModalOpen(true);
}

export function closeModal() {
  setIsModalOpen(false);
  setModalFile(null);
  setPendingModalEdge(null);
}

// Context menu state
export interface FileMenuState {
  file: FileData;
  x: number;
  y: number;
  // Element to return focus to once the menu closes.
  returnFocus?: HTMLElement | null;
}
export const [fileMenu, setFileMenu] = createSignal<FileMenuState | null>(null);

export function openFileMenu(
  file: FileData,
  x: number,
  y: number,
  returnFocus?: HTMLElement | null,
) {
  setFileMenu({ file, x, y, returnFocus });
}

export function closeFileMenu(restoreFocus = true) {
  const menu = fileMenu();
  setFileMenu(null);
  if (restoreFocus) menu?.returnFocus?.focus?.();
}

// Confirm dialog state
export interface ConfirmRequest {
  title: string;
  message: string;
  confirmLabel: string;
  danger?: boolean;
  resolve: (confirmed: boolean) => void;
}
export const [confirmRequest, setConfirmRequest] =
  createSignal<ConfirmRequest | null>(null);

export function cancelConfirmation() {
  const request = confirmRequest();
  setConfirmRequest(null);
  request?.resolve(false);
}

export function confirmDialog(
  options: Omit<ConfirmRequest, "resolve">,
): Promise<boolean> {
  confirmRequest()?.resolve(false);
  return new Promise((resolve) => setConfirmRequest({ ...options, resolve }));
}

export { showToast } from "./toast";

export function updateFileInList(fileName: string, updates: Partial<FileData>) {
  setFileState("files", (file) => file.FileName === fileName, updates);

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

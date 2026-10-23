import {
  closeModal,
  confirmDialog,
  currentPage,
  modalFile,
  removeFileFromList,
  showToast,
  updateFileInList,
} from "./store";
import { deleteFile, toggleFileVisibility } from "./generated";
import { FILES_PER_PAGE } from "./utils";
import { apiData } from "./client";
import { fileUrl, copyText } from "./utils";
import { loadFiles } from "./components/FileGrid";
import { loadStats } from "./components/FileStats";
import type { FileData } from "./types";

export function absoluteFileUrl(file: FileData): string {
  return new URL(fileUrl(file.FileName), window.location.origin).href;
}

// Resolves to whether the copy succeeded; failures are reported via toast.
export async function copyFileLink(file: FileData): Promise<boolean> {
  try {
    await copyText(absoluteFileUrl(file));
    return true;
  } catch {
    showToast("Failed to copy link", "error");
    return false;
  }
}

const pendingVisibility = new Set<string>();

export async function toggleVisibility(file: FileData) {
  if (pendingVisibility.has(file.FileName)) return;
  pendingVisibility.add(file.FileName);
  try {
    await apiData(toggleFileVisibility({ body: { file_name: file.FileName } }));
    updateFileInList(file.FileName, { Public: !file.Public });
  } catch (error) {
    showToast(
      error instanceof Error ? error.message : "Failed to change visibility",
      "error",
    );
  } finally {
    pendingVisibility.delete(file.FileName);
  }
}

const pendingDeletes = new Set<string>();

export async function deleteFileWithConfirm(file: FileData) {
  if (pendingDeletes.has(file.FileName)) return;
  const confirmed = await confirmDialog({
    title: "Delete file?",
    message: `"${file.OriginalFileName || file.FileName}" will be permanently deleted.`,
    confirmLabel: "Delete",
    danger: true,
  });
  if (!confirmed) return;

  pendingDeletes.add(file.FileName);
  try {
    await apiData(deleteFile({ body: { file_name: file.FileName } }));
    if (modalFile()?.FileName === file.FileName) closeModal();
    removeFileFromList(file.FileName);
    loadStats();
    // Refill the page from the next one; loadFiles steps back a page
    // if this was the last file on it.
    loadFiles(currentPage() * FILES_PER_PAGE);
  } catch (error) {
    showToast(
      error instanceof Error ? error.message : "Failed to delete file",
      "error",
    );
  } finally {
    pendingDeletes.delete(file.FileName);
  }
}

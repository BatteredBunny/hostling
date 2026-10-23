import {
  closeModal,
  confirmDialog,
  currentPage,
  modalFile,
  removeFileFromList,
  showToast,
  updateFileInList,
} from './store';
import { deleteFile, toggleFileVisibility, FILES_PER_PAGE } from './api';
import { fileUrl } from './utils';
import { loadFiles } from './components/FileGrid';
import { loadStats } from './components/FileStats';
import type { FileData } from './types';

export function absoluteFileUrl(file: FileData): string {
  return new URL(fileUrl(file.FileName), window.location.origin).href;
}

// Resolves to whether the copy succeeded; failures are reported via toast.
export async function copyFileLink(file: FileData): Promise<boolean> {
  const url = absoluteFileUrl(file);
  try {
    await navigator.clipboard.writeText(url);
  } catch {
    // The clipboard API is unavailable outside secure contexts (e.g. plain http on LAN).
    const textarea = document.createElement('textarea');
    textarea.value = url;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    const copied = document.execCommand('copy');
    textarea.remove();
    if (!copied) {
      showToast('Failed to copy link');
      return false;
    }
  }
  return true;
}

const pendingVisibility = new Set<string>();

export async function toggleVisibility(file: FileData) {
  if (pendingVisibility.has(file.FileName)) return;
  pendingVisibility.add(file.FileName);
  try {
    const result = await toggleFileVisibility(file.FileName);
    if (result.ok) {
      updateFileInList(file.FileName, { Public: !file.Public });
    } else {
      alert(`Failed to change visibility: ${result.error || 'unknown error'}`);
    }
  } finally {
    pendingVisibility.delete(file.FileName);
  }
}

const pendingDeletes = new Set<string>();

export async function deleteFileWithConfirm(file: FileData) {
  if (pendingDeletes.has(file.FileName)) return;
  const confirmed = await confirmDialog({
    title: 'Delete file?',
    message: `"${file.OriginalFileName || file.FileName}" will be permanently deleted.`,
    confirmLabel: 'Delete',
    danger: true,
  });
  if (!confirmed) return;

  pendingDeletes.add(file.FileName);
  try {
    const result = await deleteFile(file.FileName);
    if (!result.ok) {
      alert(`Failed to delete file: ${result.error || 'unknown error'}`);
      return;
    }
    if (modalFile()?.FileName === file.FileName) closeModal();
    removeFileFromList(file.FileName);
    loadStats();
    // Refill the page from the next one; loadFiles steps back a page
    // if this was the last file on it.
    loadFiles(currentPage() * FILES_PER_PAGE);
  } finally {
    pendingDeletes.delete(file.FileName);
  }
}

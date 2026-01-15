import type { FilesResponse, FileStatsResponse, SortField, SortOrder } from './types';

const FILES_PER_PAGE = 8;

export async function fetchFiles(
  skip: number,
  sort: SortField,
  desc: boolean
): Promise<FilesResponse> {
  const response = await fetch(
    `/api/account/files?skip=${skip}&sort=${sort}&desc=${desc}`,
    { method: 'GET' }
  );

  if (!response.ok) {
    throw new Error('Failed to fetch files');
  }

  return response.json();
}

export async function fetchFileStats(): Promise<FileStatsResponse> {
  const response = await fetch('/api/account/files/stats', { method: 'GET' });

  if (!response.ok) {
    throw new Error('Failed to fetch stats');
  }

  return response.json();
}

export async function deleteFile(fileName: string): Promise<boolean> {
  const formData = new FormData();
  formData.append('file_name', fileName);

  const response = await fetch('/api/account/file', {
    method: 'DELETE',
    body: formData,
  });

  return response.ok;
}

export async function toggleFileVisibility(fileName: string): Promise<boolean> {
  const formData = new FormData();
  formData.append('file_name', fileName);

  const response = await fetch('/api/account/file/public', {
    method: 'POST',
    body: formData,
  });

  return response.ok;
}

export async function addFileTag(fileName: string, tag: string): Promise<boolean> {
  const formData = new FormData();
  formData.append('file_name', fileName);
  formData.append('tag', tag);

  const response = await fetch('/api/account/file/tag', {
    method: 'POST',
    body: formData,
  });

  return response.ok;
}

export async function removeFileTag(fileName: string, tag: string): Promise<boolean> {
  const formData = new FormData();
  formData.append('file_name', fileName);
  formData.append('tag', tag);

  const response = await fetch('/api/account/file/tag', {
    method: 'DELETE',
    body: formData,
  });

  return response.ok;
}

export { FILES_PER_PAGE };

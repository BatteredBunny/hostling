import type { FilesResponse, FileStatsResponse, SortField } from './types';

const FILES_PER_PAGE = 8;
const TAG_MAX_LENGTH = 25;
const MAX_TAGS_PER_FILE = 50;

export async function fetchFiles(
  skip: number,
  sort: SortField,
  desc: boolean,
  tagFilter?: string | null,
  fileFilter?: string | null,
  signal?: AbortSignal
): Promise<FilesResponse> {
  const params = new URLSearchParams({
    skip: skip.toString(),
    sort,
    desc: desc.toString(),
  });
  if (tagFilter) params.set('tag', tagFilter);
  if (fileFilter) params.set('filter', fileFilter);

  const response = await fetch(`/api/account/files?${params}`, { method: 'GET', signal });

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

export interface MutationResult {
  ok: boolean;
  status?: number;
  error?: string;
}

async function mutate(url: string, method: string, fields: Record<string, string>): Promise<MutationResult> {
  const formData = new FormData();
  for (const [k, v] of Object.entries(fields)) formData.append(k, v);

  let response: Response;
  try {
    response = await fetch(url, { method, body: formData });
  } catch (err) {
    return { ok: false, error: (err as Error)?.message || 'Network error' };
  }

  if (response.ok) return { ok: true, status: response.status };

  const body = await response.text().catch(() => '');
  const contentType = response.headers.get('content-type') ?? '';
  const message = contentType.startsWith('text/plain') && body
    ? body.slice(0, 300)
    : response.statusText;

  return { ok: false, status: response.status, error: message };
}

export function deleteFile(fileName: string): Promise<MutationResult> {
  return mutate('/api/account/file', 'DELETE', { file_name: fileName });
}

export function toggleFileVisibility(fileName: string): Promise<MutationResult> {
  return mutate('/api/account/file/public', 'POST', { file_name: fileName });
}

export function addFileTag(fileName: string, tag: string): Promise<MutationResult> {
  return mutate('/api/account/file/tag', 'POST', { file_name: fileName, tag });
}

export function removeFileTag(fileName: string, tag: string): Promise<MutationResult> {
  return mutate('/api/account/file/tag', 'DELETE', { file_name: fileName, tag });
}

export { FILES_PER_PAGE, TAG_MAX_LENGTH, MAX_TAGS_PER_FILE };

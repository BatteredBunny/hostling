export interface Tag {
  ID: number;
  Name: string;
}

export interface FileData {
  FileName: string;
  OriginalFileName: string;
  FileSize: number;
  MimeType: string;
  Public: boolean;
  ViewsCount: number;
  ExpiryDate: string;
  CreatedAt: string;
  Tags: Tag[];
}

export interface FilesResponse {
  files: FileData[];
  count: number;
}

export interface FileStatsResponse {
  count: number;
  size_total: number;
  tags: string[];
}

export type SortField = 'created_at' | 'views' | 'file_size';
export type SortOrder = 'asc' | 'desc';

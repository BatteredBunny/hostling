import type { Files, Tag as ApiTag, ListFilesData } from "./generated";

// Normalize nullable collection fields once when data enters the UI.
export type FileData = Omit<Files, "Tags"> & { Tags: ApiTag[] };
export type SortField = NonNullable<
  NonNullable<ListFilesData["query"]>["sort"]
>;

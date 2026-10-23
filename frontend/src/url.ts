import type { SortField } from "./types";

const VALID_SORT_FIELDS: SortField[] = ["created_at", "views", "file_size"];
const VALID_ORDERS = ["asc", "desc"] as const;
const VALID_VISIBILITIES = ["public", "private"] as const;
export type Visibility = (typeof VALID_VISIBILITIES)[number];

export interface UrlParams {
  page: number;
  sort: SortField;
  order: "asc" | "desc";
  tags: string[];
  visibility: Visibility | null;
  untagged: boolean;
  file: string | null;
}

export function parseUrlParams(): UrlParams {
  const params = new URLSearchParams(window.location.search);

  const pageStr = params.get("page");
  let page = 0;
  if (pageStr) {
    const parsed = parseInt(pageStr, 10);
    if (!isNaN(parsed) && parsed >= 1) {
      page = parsed - 1; // URL is 1-based, internal is 0-based
    }
  }

  const sortStr = params.get("sort") as SortField | null;
  const sort: SortField =
    sortStr && VALID_SORT_FIELDS.includes(sortStr) ? sortStr : "created_at";

  const orderStr = params.get("order");
  const order: "asc" | "desc" =
    orderStr && (VALID_ORDERS as readonly string[]).includes(orderStr)
      ? (orderStr as "asc" | "desc")
      : "desc";

  // `tag` is repeatable too; files must have every tag.
  const tags = [
    ...new Set(
      params
        .getAll("tag")
        .map((t) => t.trim().toLowerCase())
        .filter(Boolean),
    ),
  ];

  // `filter` is repeatable, mirroring the API: e.g. ?filter=private&filter=untagged
  const filters = params.getAll("filter");
  const untagged = filters.includes("untagged");
  const visibility =
    VALID_VISIBILITIES.find((v) => filters.includes(v)) ?? null;

  const fileStr = params.get("file");
  const file = fileStr && fileStr.trim() ? fileStr.trim() : null;

  return { page, sort, order, tags, visibility, untagged, file };
}

interface UrlState {
  page: number;
  sort: SortField;
  desc: boolean;
  tags: string[];
  visibility: Visibility | null;
  untagged: boolean;
  file: string | null;
}

export function updateUrl(
  state: UrlState,
  options: { replace?: boolean } = {},
) {
  const params = new URLSearchParams();

  if (state.page > 0) {
    params.set("page", String(state.page + 1)); // internal 0-based → URL 1-based
  }

  if (state.sort !== "created_at") {
    params.set("sort", state.sort);
  }

  if (!state.desc) {
    params.set("order", "asc");
  }

  for (const tag of state.tags) {
    params.append("tag", tag);
  }

  if (state.visibility) {
    params.append("filter", state.visibility);
  }

  if (state.untagged) {
    params.append("filter", "untagged");
  }

  if (state.file) {
    params.set("file", state.file);
  }

  const search = params.toString();
  const url = search
    ? `${window.location.pathname}?${search}`
    : window.location.pathname;

  const current = window.location.pathname + window.location.search;
  if (url === current) return;

  if (options.replace) {
    history.replaceState(null, "", url);
  } else {
    history.pushState(null, "", url);
  }
}

import type { SortField } from './types';

const VALID_SORT_FIELDS: SortField[] = ['created_at', 'views', 'file_size'];
const VALID_ORDERS = ['asc', 'desc'] as const;
const VALID_FILTERS = ['untagged', 'public', 'private'] as const;

export interface UrlParams {
  page: number;
  sort: SortField;
  order: 'asc' | 'desc';
  tag: string | null;
  filter: string | null;
  file: string | null;
}

export function parseUrlParams(): UrlParams {
  const params = new URLSearchParams(window.location.search);

  const pageStr = params.get('page');
  let page = 0;
  if (pageStr) {
    const parsed = parseInt(pageStr, 10);
    if (!isNaN(parsed) && parsed >= 1) {
      page = parsed - 1; // URL is 1-based, internal is 0-based
    }
  }

  const sortStr = params.get('sort') as SortField | null;
  const sort: SortField = sortStr && VALID_SORT_FIELDS.includes(sortStr) ? sortStr : 'created_at';

  const orderStr = params.get('order');
  const order: 'asc' | 'desc' = orderStr && (VALID_ORDERS as readonly string[]).includes(orderStr)
    ? (orderStr as 'asc' | 'desc')
    : 'desc';

  const tagStr = params.get('tag');
  const tag = tagStr && tagStr.trim() ? tagStr.trim() : null;

  const filterStr = params.get('filter');
  const filter = filterStr && (VALID_FILTERS as readonly string[]).includes(filterStr) ? filterStr : null;

  const fileStr = params.get('file');
  const file = fileStr && fileStr.trim() ? fileStr.trim() : null;

  return { page, sort, order, tag, filter, file };
}

interface UrlState {
  page: number;
  sort: SortField;
  desc: boolean;
  tag: string | null;
  filter: string | null;
  file: string | null;
}

export function updateUrl(state: UrlState) {
  const params = new URLSearchParams();

  if (state.page > 0) {
    params.set('page', String(state.page + 1)); // internal 0-based → URL 1-based
  }

  if (state.sort !== 'created_at') {
    params.set('sort', state.sort);
  }

  if (!state.desc) {
    params.set('order', 'asc');
  }

  if (state.tag) {
    params.set('tag', state.tag);
  }

  if (state.filter) {
    params.set('filter', state.filter);
  }

  if (state.file) {
    params.set('file', state.file);
  }

  const search = params.toString();
  const url = search ? `${window.location.pathname}?${search}` : window.location.pathname;
  history.replaceState(null, '', url);
}

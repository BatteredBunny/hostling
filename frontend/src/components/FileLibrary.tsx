import { createEffect } from 'solid-js';
import { FileGrid } from './FileGrid';
import { currentPage, sortField, sortDesc, tagFilter, fileFilter, modalFile } from '../store';
import { updateUrl } from '../url';

export function FileLibrary() {
  createEffect(() => {
    updateUrl({
      page: currentPage(),
      sort: sortField(),
      desc: sortDesc(),
      tag: tagFilter(),
      filter: fileFilter(),
      file: modalFile()?.FileName ?? null,
    });
  });

  return <FileGrid />;
}

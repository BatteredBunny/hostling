import { createEffect, flush, onCleanup, onSettled, Show } from "solid-js";
import { FileGrid, loadFiles, cancelFileLoad } from "./FileGrid";
import { FileModal } from "./FileModal";
import { FileMenu } from "./FileMenu";
import {
  applyUrlParams,
  currentPage,
  sortField,
  sortDesc,
  tagFilters,
  visibilityFilter,
  untaggedFilter,
  modalFile,
  fileMenu,
  pendingModalFile,
  setPendingModalFile,
  setPendingModalEdge,
  closeModal,
  closeFileMenu,
  cancelConfirmation,
  consumeReplaceNextUrlSync,
} from "../store";
import { updateUrl, parseUrlParams } from "../url";
import { FILES_PER_PAGE } from "../utils";

export function FileLibrary() {
  onCleanup(() => {
    cancelFileLoad();
    closeModal();
    closeFileMenu(false);
    cancelConfirmation();
  });
  createEffect(
    () => ({
      page: currentPage(),
      tags: tagFilters(),
      sort: sortField(),
      desc: sortDesc(),
      visibility: visibilityFilter(),
      untagged: untaggedFilter(),
      file: modalFile()?.FileName ?? pendingModalFile() ?? null,
    }),
    (state, previous) => {
      // Only push a new history entry on discrete navigation (page / tag).
      // Sort toggles, filter changes, and modal open/close mutate in place.
      const forceReplace = consumeReplaceNextUrlSync();
      const isNavigation =
        !forceReplace &&
        !!previous &&
        (state.page !== previous.page ||
          state.tags.join(",") !== previous.tags.join(","));
      updateUrl(state, { replace: !isNavigation });
    },
  );

  onSettled(() => {
    const handlePopState = () => {
      if (window.location.pathname !== "/gallery") return;
      const next = parseUrlParams();
      applyUrlParams(next);
      setPendingModalEdge(null);
      // An open modal stays until loadFiles finds the requested file.
      if (!next.file) closeModal();
      setPendingModalFile(next.file);
      flush();
      loadFiles(next.page * FILES_PER_PAGE);
    };

    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  });

  return (
    <>
      <FileGrid />
      <Show when={modalFile()}>{(file) => <FileModal file={file()} />}</Show>
      <Show when={fileMenu()} keyed>
        {(menu) => <FileMenu menu={menu} />}
      </Show>
    </>
  );
}

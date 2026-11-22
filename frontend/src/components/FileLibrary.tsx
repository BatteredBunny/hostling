import { batch, createEffect, onCleanup, onMount, Show } from "solid-js";
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
  // Astro navigation retains modules, so a new visit must read its own URL
  // rather than reuse the selection from the module's first import.
  onCleanup(() => {
    cancelFileLoad();
    closeModal();
    closeFileMenu(false);
    cancelConfirmation();
  });
  const initial = parseUrlParams();
  batch(() => {
    applyUrlParams(initial);
    closeModal();
    setPendingModalFile(initial.file);
  });
  let lastPage = currentPage();
  let lastTags = tagFilters().join(",");

  createEffect(() => {
    const page = currentPage();
    const tags = tagFilters();
    const tagsKey = tags.join(",");
    // Only push a new history entry on discrete navigation (page / tag).
    // Sort toggles, filter changes, and modal open/close mutate in place.
    const forceReplace = consumeReplaceNextUrlSync();
    const isNavigation =
      !forceReplace && (page !== lastPage || tagsKey !== lastTags);
    lastPage = page;
    lastTags = tagsKey;

    updateUrl(
      {
        page,
        sort: sortField(),
        desc: sortDesc(),
        tags,
        visibility: visibilityFilter(),
        untagged: untaggedFilter(),
        file: modalFile()?.FileName ?? pendingModalFile() ?? null,
      },
      { replace: !isNavigation },
    );
  });

  onMount(() => {
    const handlePopState = () => {
      if (window.location.pathname !== "/gallery") return;
      const next = parseUrlParams();
      // Batched so the URL sync effect sees the whole history entry at once
      // instead of pushing entries for each partially applied field.
      batch(() => {
        applyUrlParams(next);
        setPendingModalEdge(null);
        // An open modal stays until loadFiles finds the requested file.
        if (!next.file) closeModal();
        setPendingModalFile(next.file);
      });
      loadFiles(next.page * FILES_PER_PAGE);
    };

    window.addEventListener("popstate", handlePopState);
    onCleanup(() => window.removeEventListener("popstate", handlePopState));
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

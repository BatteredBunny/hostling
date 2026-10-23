import { batch, createEffect, onCleanup, onMount, Show } from "solid-js";
import { FileGrid, loadFiles, cancelFileLoad } from "./FileGrid";
import { FileModal } from "./FileModal";
import { FileMenu } from "./FileMenu";
import { ConfirmDialog } from "./ConfirmDialog";
import { HostlingToast } from "./HostlingToast";
import {
  currentPage,
  setCurrentPage,
  sortField,
  setSortField,
  sortDesc,
  setSortDesc,
  tagFilters,
  setTagFilters,
  visibilityFilter,
  untaggedFilter,
  setVisibilityFilter,
  setUntaggedFilter,
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
    setSortField(initial.sort);
    setSortDesc(initial.order === "desc");
    setTagFilters(initial.tags);
    setVisibilityFilter(initial.visibility);
    setUntaggedFilter(initial.untagged);
    setCurrentPage(initial.page);
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
      const next = parseUrlParams();
      setSortField(next.sort);
      setSortDesc(next.order === "desc");
      setTagFilters(next.tags);
      setVisibilityFilter(next.visibility);
      setUntaggedFilter(next.untagged);
      setCurrentPage(next.page);
      setPendingModalEdge(null);
      if (next.file) {
        setPendingModalFile(next.file);
      } else {
        closeModal();
        setPendingModalFile(null);
      }
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
      <ConfirmDialog />
      <HostlingToast />
    </>
  );
}

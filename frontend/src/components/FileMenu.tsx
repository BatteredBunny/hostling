import { closeFileMenu, showToast, type FileMenuState } from "../store";
import {
  absoluteFileUrl,
  copyFileLink,
  deleteFileWithConfirm,
  toggleVisibility,
} from "../fileActions";
import DownloadIcon from "lucide-solid/icons/download";
import ExternalLinkIcon from "lucide-solid/icons/external-link";
import LinkIcon from "lucide-solid/icons/link";
import LockIcon from "lucide-solid/icons/lock";
import LockOpenIcon from "lucide-solid/icons/lock-open";
import Trash2Icon from "lucide-solid/icons/trash";
import { displayName } from "../utils";
import { ContextMenu, type ContextMenuItem } from "./ContextMenu";

function menuItems(menu: FileMenuState): ContextMenuItem[] {
  const file = menu.file;
  return [
    {
      label: "Copy link",
      icon: LinkIcon,
      action: async () => {
        // The menu is gone by now, so confirm via toast instead.
        if (await copyFileLink(file)) showToast("Link copied");
      },
    },
    {
      label: "Open in new tab",
      icon: ExternalLinkIcon,
      href: absoluteFileUrl(file),
    },
    {
      label: "Download",
      icon: DownloadIcon,
      href: absoluteFileUrl(file),
      download: displayName(file),
    },
    {
      label: file.Public ? "Make private" : "Make public",
      icon: file.Public ? LockIcon : LockOpenIcon,
      action: () => toggleVisibility(file),
    },
    {
      label: "Delete",
      icon: Trash2Icon,
      danger: true,
      separatorBefore: true,
      action: () => deleteFileWithConfirm(file),
    },
  ];
}

export function FileMenu(props: { menu: FileMenuState }) {
  return (
    <ContextMenu
      x={props.menu.x}
      y={props.menu.y}
      label={displayName(props.menu.file)}
      items={menuItems(props.menu)}
      onClose={closeFileMenu}
    />
  );
}

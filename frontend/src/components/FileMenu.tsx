import { For, Show, createSignal, onCleanup, onMount } from 'solid-js';
import { Portal } from 'solid-js/web';
import './FileMenu.css';
import { closeFileMenu, showToast, type FileMenuState } from '../store';
import { absoluteFileUrl, copyFileLink, deleteFileWithConfirm, toggleVisibility } from '../fileActions';
import { Icon } from './Icon';

interface MenuItem {
  label: string;
  icon: string;
  danger?: boolean;
  separatorBefore?: boolean;
  href?: string;
  download?: string;
  action?: () => void;
}

function menuItems(menu: FileMenuState): MenuItem[] {
  const file = menu.file;
  return [
    {
      label: 'Copy link',
      icon: 'link',
      action: async () => {
        // The menu is gone by now, so confirm via toast instead.
        if (await copyFileLink(file)) showToast('Link copied');
      },
    },
    { label: 'Open in new tab', icon: 'external-link', href: absoluteFileUrl(file) },
    {
      label: 'Download',
      icon: 'download',
      href: absoluteFileUrl(file),
      download: file.OriginalFileName || file.FileName,
    },
    {
      label: file.Public ? 'Make private' : 'Make public',
      icon: file.Public ? 'lock' : 'lock-open',
      action: () => toggleVisibility(file),
    },
    {
      label: 'Delete',
      icon: 'trash-2',
      danger: true,
      separatorBefore: true,
      action: () => deleteFileWithConfirm(file),
    },
  ];
}

export function FileMenu(props: { menu: FileMenuState }) {
  let menuEl: HTMLDivElement | undefined;
  // Placed in onMount once the menu's size is known, before the first paint.
  const [position, setPosition] = createSignal({ left: 0, top: 0 });

  const menuButtons = () =>
    Array.from(menuEl?.querySelectorAll<HTMLElement>('.file-menu-item') ?? []);

  const moveFocus = (delta: number) => {
    const buttons = menuButtons();
    const index = buttons.indexOf(document.activeElement as HTMLElement);
    const next = (index + delta + buttons.length) % buttons.length;
    buttons[next]?.focus();
  };

  onMount(() => {
    // Keep the menu on screen, flipping it to the other side of the
    // cursor when it would overflow.
    const rect = menuEl!.getBoundingClientRect();
    const margin = 8;
    let left = props.menu.x;
    let top = props.menu.y;
    if (left + rect.width > window.innerWidth - margin) left = Math.max(margin, left - rect.width);
    if (top + rect.height > window.innerHeight - margin) top = Math.max(margin, top - rect.height);
    setPosition({ left, top });
    menuButtons()[0]?.focus();

    // Capture phase so the file modal's own key handling (Escape, arrows)
    // doesn't also react while the menu is open.
    const onKey = (e: KeyboardEvent) => {
      e.stopPropagation();
      if (e.key === 'Escape') {
        e.preventDefault();
        closeFileMenu();
      } else if (e.key === 'ArrowDown') {
        e.preventDefault();
        moveFocus(1);
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        moveFocus(-1);
      } else if (e.key === 'Tab') {
        closeFileMenu();
      }
    };
    const onDismiss = () => closeFileMenu(false);

    window.addEventListener('keydown', onKey, true);
    window.addEventListener('scroll', onDismiss, true);
    window.addEventListener('resize', onDismiss);
    onCleanup(() => {
      window.removeEventListener('keydown', onKey, true);
      window.removeEventListener('scroll', onDismiss, true);
      window.removeEventListener('resize', onDismiss);
    });
  });

  const runItem = (item: MenuItem) => {
    // Return focus before the action runs, so dialogs it opens can take
    // focus and later hand it back to the trigger.
    closeFileMenu();
    item.action?.();
  };

  return (
    <Portal>
      {/* Swallows the dismissing click so it doesn't also hit what's underneath. */}
      <div
        class="file-menu-overlay"
        onClick={() => closeFileMenu(false)}
        onContextMenu={(e) => {
          e.preventDefault();
          closeFileMenu(false);
        }}
      />
      <div
        ref={menuEl}
        class="file-menu"
        role="menu"
        aria-label={props.menu.file.OriginalFileName || props.menu.file.FileName}
        style={{ left: `${position().left}px`, top: `${position().top}px` }}
        onContextMenu={(e) => e.preventDefault()}
      >
        <For each={menuItems(props.menu)}>
          {(item) => (
            <>
              <Show when={item.separatorBefore}>
                <div class="file-menu-separator" role="separator" />
              </Show>
              <Show
                when={item.href}
                fallback={
                  <button
                    type="button"
                    role="menuitem"
                    class="file-menu-item"
                    classList={{ danger: item.danger }}
                    onClick={() => runItem(item)}
                  >
                    <Icon name={item.icon} />
                    <span>{item.label}</span>
                  </button>
                }
              >
                <a
                  role="menuitem"
                  class="file-menu-item"
                  href={item.href}
                  target={item.download ? undefined : '_blank'}
                  rel="noopener noreferrer"
                  download={item.download}
                  onClick={() => closeFileMenu()}
                >
                  <Icon name={item.icon} />
                  <span>{item.label}</span>
                </a>
              </Show>
            </>
          )}
        </For>
      </div>
    </Portal>
  );
}

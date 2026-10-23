import { For, Show, createEffect, createSignal, createUniqueId, onCleanup } from 'solid-js';
import { Portal } from 'solid-js/web';
import './SelectBox.css';
import { Icon } from './Icon';

export interface SelectBoxOption {
  value: string;
  label: string;
}

interface SelectBoxProps {
  id?: string;
  value: string;
  options: SelectBoxOption[];
  onChange: (value: string) => void;
  'aria-label'?: string;
}

// A styled replacement for <select>, whose option list can't be themed.
// Follows the WAI-ARIA select-only combobox pattern.
export function SelectBox(props: SelectBoxProps) {
  const [open, setOpen] = createSignal(false);
  const [activeIndex, setActiveIndex] = createSignal(0);
  const listId = createUniqueId();

  let root: HTMLDivElement | undefined;
  let button: HTMLButtonElement | undefined;
  let list: HTMLDivElement | undefined;

  const selectedIndex = () => props.options.findIndex((o) => o.value === props.value);
  const selectedLabel = () => props.options[selectedIndex()]?.label ?? '';
  const optionId = (index: number) => `${listId}-${index}`;

  const [position, setPosition] = createSignal({ top: 0, right: 0, minWidth: 0 });

  const openList = () => {
    // Rendered in a portal so containers with overflow: hidden can't clip it;
    // right-aligned with the button.
    const rect = button!.getBoundingClientRect();
    setPosition({ top: rect.bottom + 6, right: window.innerWidth - rect.right, minWidth: rect.width });
    setActiveIndex(Math.max(0, selectedIndex()));
    setOpen(true);
  };

  const close = (refocus = true) => {
    setOpen(false);
    if (refocus) button?.focus();
  };

  const choose = (index: number) => {
    const option = props.options[index];
    close();
    if (option && option.value !== props.value) props.onChange(option.value);
  };

  createEffect(() => {
    if (!open()) return;
    list?.focus();

    const onPointerDown = (e: PointerEvent) => {
      const target = e.target as Node;
      if (!root?.contains(target) && !list?.contains(target)) close(false);
    };
    // The list is placed against the viewport, so it would drift on scroll.
    const onScroll = (e: Event) => {
      if (!list?.contains(e.target as Node)) close(false);
    };
    const onResize = () => close(false);
    document.addEventListener('pointerdown', onPointerDown);
    window.addEventListener('scroll', onScroll, true);
    window.addEventListener('resize', onResize);
    onCleanup(() => {
      document.removeEventListener('pointerdown', onPointerDown);
      window.removeEventListener('scroll', onScroll, true);
      window.removeEventListener('resize', onResize);
    });
  });

  createEffect(() => {
    if (!open()) return;
    list?.children[activeIndex()]?.scrollIntoView({ block: 'nearest' });
  });

  const onButtonKeyDown = (e: KeyboardEvent) => {
    if (['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(e.key)) {
      e.preventDefault();
      openList();
    }
  };

  const onListKeyDown = (e: KeyboardEvent) => {
    const last = props.options.length - 1;
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setActiveIndex((i) => Math.min(last, i + 1));
        break;
      case 'ArrowUp':
        e.preventDefault();
        setActiveIndex((i) => Math.max(0, i - 1));
        break;
      case 'Home':
        e.preventDefault();
        setActiveIndex(0);
        break;
      case 'End':
        e.preventDefault();
        setActiveIndex(last);
        break;
      case 'Enter':
      case ' ':
        e.preventDefault();
        choose(activeIndex());
        break;
      case 'Escape':
        e.preventDefault();
        e.stopPropagation();
        close();
        break;
      case 'Tab':
        close(false);
        break;
    }
  };

  return (
    <div class="select-box" classList={{ open: open() }} ref={root}>
      <button
        type="button"
        ref={button}
        id={props.id}
        class="select-box-button"
        aria-haspopup="listbox"
        aria-expanded={open()}
        aria-controls={listId}
        aria-label={props['aria-label']}
        onClick={() => (open() ? close() : openList())}
        onKeyDown={onButtonKeyDown}
      >
        <span>{selectedLabel()}</span>
        <Icon name="chevron-down" />
      </button>

      <Show when={open()}>
        <Portal>
          <div
            ref={list}
            id={listId}
            class="select-box-list"
            role="listbox"
            tabindex="-1"
            aria-activedescendant={optionId(activeIndex())}
            style={{
              top: `${position().top}px`,
              right: `${position().right}px`,
              'min-width': `${position().minWidth}px`,
            }}
            onKeyDown={onListKeyDown}
          >
            <For each={props.options}>
              {(option, index) => (
                <div
                  id={optionId(index())}
                  role="option"
                  class="select-box-option"
                  classList={{ active: index() === activeIndex() }}
                  aria-selected={option.value === props.value}
                  onClick={() => choose(index())}
                  onMouseEnter={() => setActiveIndex(index())}
                >
                  <span>{option.label}</span>
                  <Show when={option.value === props.value}>
                    <Icon name="check" />
                  </Show>
                </div>
              )}
            </For>
          </div>
        </Portal>
      </Show>
    </div>
  );
}

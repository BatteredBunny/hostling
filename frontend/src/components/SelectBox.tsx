import {
  For,
  Show,
  createEffect,
  createSignal,
  createUniqueId,
} from "solid-js";
import { NativePopover } from "./NativePopover";
import "./SelectBox.css";
import {
  Check as CheckIcon,
  ChevronDown as ChevronDownIcon,
} from "lucide-solid";

export interface SelectBoxOption {
  value: string;
  label: string;
}

interface SelectBoxProps {
  id?: string;
  value: string;
  options: SelectBoxOption[];
  onChange: (value: string) => void;
  "aria-label"?: string;
}

// A styled replacement for <select>, whose option list can't be themed.
// Follows the WAI-ARIA select-only combobox pattern.
export function SelectBox(props: SelectBoxProps) {
  const [open, setOpen] = createSignal(false);
  const [activeIndex, setActiveIndex] = createSignal(0);
  const listId = createUniqueId();

  let button: HTMLButtonElement | undefined;
  let list: HTMLDivElement | undefined;

  const selectedIndex = () =>
    props.options.findIndex((o) => o.value === props.value);
  const selectedLabel = () => props.options[selectedIndex()]?.label ?? "";
  const optionId = (index: number) => `${listId}-${index}`;

  const openList = () => {
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
    list?.children[activeIndex()]?.scrollIntoView({ block: "nearest" });
  });

  const onButtonKeyDown = (e: KeyboardEvent) => {
    if (["ArrowDown", "ArrowUp", "Enter", " "].includes(e.key)) {
      e.preventDefault();
      openList();
    }
  };

  const onListKeyDown = (e: KeyboardEvent) => {
    const last = props.options.length - 1;
    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        setActiveIndex((i) => Math.min(last, i + 1));
        break;
      case "ArrowUp":
        e.preventDefault();
        setActiveIndex((i) => Math.max(0, i - 1));
        break;
      case "Home":
        e.preventDefault();
        setActiveIndex(0);
        break;
      case "End":
        e.preventDefault();
        setActiveIndex(last);
        break;
      case "Enter":
      case " ":
        e.preventDefault();
        choose(activeIndex());
        break;
      case "Escape":
        e.preventDefault();
        e.stopPropagation();
        close();
        break;
      case "Tab":
        close(false);
        break;
    }
  };

  return (
    <div class="select-box" classList={{ open: open() }}>
      <button
        type="button"
        ref={button}
        id={props.id}
        class="select-box-button"
        aria-haspopup="listbox"
        aria-expanded={open()}
        aria-controls={listId}
        aria-label={props["aria-label"]}
        onClick={() => (open() ? close() : openList())}
        onKeyDown={onButtonKeyDown}
      >
        <span>{selectedLabel()}</span>
        <ChevronDownIcon class="lucide-icon" aria-hidden="true" />
      </button>

      <NativePopover
        open={open()}
        anchor={() => button}
        onOpenChange={setOpen}
        onOpen={() => list?.focus()}
        ref={(element) => {
          list = element;
        }}
        id={listId}
        class="select-box-list"
        role="listbox"
        label={props["aria-label"]}
        activeDescendant={optionId(activeIndex())}
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
                <CheckIcon class="lucide-icon" aria-hidden="true" />
              </Show>
            </div>
          )}
        </For>
      </NativePopover>
    </div>
  );
}

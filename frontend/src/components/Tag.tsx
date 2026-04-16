import type { JSX } from 'solid-js';
import { Show, createMemo } from 'solid-js';
import './Tag.css';
import { hashStringToHSL } from '../utils';
import { Icon } from './Icon';

interface TagProps {
  name: string;
  selected?: boolean;
  enabled?: boolean; // Disable selecting tag
  onRemove?: (name: string) => void;
  onClicked?: () => void;
}

export function Tag(props: TagProps): JSX.Element {
  const hsl = createMemo(() => hashStringToHSL(props.name));

  const style = createMemo(() => {
    const { h, s, l } = hsl();
    if (props.selected) {
      return {
        "background-color": `hsl(${h}, ${s}%, ${l - 30}%)`,
        "border-color": `hsl(${h}, ${s}%, ${l}%)`,
      };
    }
    return {
      "background-color": `hsl(${h}, ${s}%, ${l}%)`,
      "border-color": `hsl(${h}, ${s}%, ${l - 30}%)`,
    };
  });

  return (
    <span class="file-tag" style={{
      ...style(),
      "cursor": props.onClicked ? "pointer" : "default",
      "opacity": props.enabled === false ? "0.5" : "1"
    }} onClick={() => props.onClicked?.()}>
      <Show when={props.onRemove} fallback={props.name}>
        <span class="file-modal-tag-text">{props.name}</span>
        <button
          class="file-modal-tag-remove"
          onClick={() => props.onRemove!(props.name)}
        >
          <Icon name="x" />
        </button>
      </Show>
    </span>
  );
}
import type { JSX } from 'solid-js';
import { Show } from 'solid-js';
import './Tag.css';
import { hashStringToHSL } from '../utils';
import { Icon } from './Icon';

interface TagProps {
  name: string;
  selected?: boolean;
  onRemove?: (name: string) => void;
  onClicked?: () => void;
}

export function Tag(props: TagProps): JSX.Element {
  const { h, s, l } = hashStringToHSL(props.name);

  const normalStyle = {
    "background-color": `hsl(${h}, ${s}%, ${l}%)`,
    "border-color": `hsl(${h}, ${s}%, ${l - 30}%)`,
  };

  const selectedStyle = {
    "background-color": `hsl(${h}, ${s}%, ${l - 30}%)`,
    "border-color": `hsl(${h}, ${s}%, ${l}%)`,
  };

  return (
    <span class="file-tag" style={{
      ...(props.selected ? selectedStyle : normalStyle),
      "cursor": props.onClicked ? "pointer" : "default"
    }} onClick={props.onClicked}>
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
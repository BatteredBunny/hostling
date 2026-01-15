import type { JSX } from 'solid-js';
import { Show } from 'solid-js';
import './Tag.css';
import { hashStringToHSL } from '../utils';
import { Icon } from './Icon';

interface TagProps {
  name: string;
  onRemove?: (name: string) => void;
}

export function Tag(props: TagProps): JSX.Element {
  const style = () => {
    const { h, s, l } = hashStringToHSL(props.name);
    return `background-color: hsl(${h}, ${s}%, ${l}%); border-color: hsl(${h}, ${s}%, ${l - 30}%);`;
  };

  return (
    <span class="file-tag" style={style()}>
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

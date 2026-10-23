import { createMemo } from 'solid-js';
import { hashStringToHSL } from '../utils';
import './TagColorDot.css';

export function TagColorDot(props: { name: string }) {
  const color = createMemo(() => {
    const { h, s, l } = hashStringToHSL(props.name);
    return `hsl(${h}, ${s}%, ${l}%)`;
  });
  return <span class="tag-color-dot" aria-hidden="true" style={{ "background-color": color() }} />;
}

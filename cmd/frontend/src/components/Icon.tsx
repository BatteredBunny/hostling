import type { JSX } from 'solid-js';

interface IconProps {
  name: string;
  class?: string;
}

export function Icon(props: IconProps): JSX.Element {
  return (
    <svg class={`lucide-icon ${props.class ?? ''}`} viewBox="0 0 24 24">
      <use href={`/public/assets/lucide-sprite.svg#${props.name}`} />
    </svg>
  );
}

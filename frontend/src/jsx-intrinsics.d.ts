import 'solid-js';

declare module 'solid-js' {
  namespace JSX {
    interface IntrinsicElements {
      'setting-group': JSX.HTMLAttributes<HTMLElement>;
    }
  }
}

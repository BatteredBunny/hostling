import { Show, createUniqueId, splitProps } from 'solid-js';
import type { JSX } from 'solid-js';

type InputProps = JSX.InputHTMLAttributes<HTMLInputElement> & {
  label: string;
  hint?: string;
  error?: string;
  fieldClass?: string;
};

export function Input(props: InputProps) {
  const [local, input] = splitProps(props, ['id', 'label', 'hint', 'error', 'fieldClass', 'class', 'aria-describedby', 'aria-invalid']);
  const generatedId = createUniqueId();
  const id = () => local.id ?? `input-${generatedId}`;
  const describedBy = () => [
    local['aria-describedby'],
    local.hint ? `${id()}-hint` : undefined,
    local.error ? `${id()}-error` : undefined,
  ].filter(Boolean).join(' ') || undefined;

  return (
    <div class={`hostling-input-field ${local.fieldClass ?? ''}`}>
      <label for={id()}>{local.label}</label>
      <input
        {...input}
        id={id()}
        class={`hostling-input ${local.class ?? ''}`}
        aria-describedby={describedBy()}
        aria-invalid={local.error ? true : local['aria-invalid']}
      />
      <Show when={local.hint}><span class="hostling-input-hint" id={`${id()}-hint`}>{local.hint}</span></Show>
      <Show when={local.error}><span class="hostling-input-error" id={`${id()}-error`} role="alert">{local.error}</span></Show>
    </div>
  );
}

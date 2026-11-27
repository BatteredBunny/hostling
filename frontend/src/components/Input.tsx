import "./Input.css";
import { Show, createUniqueId, omit } from "solid-js";
import type { JSX } from "@solidjs/web";

type InputProps = JSX.InputHTMLAttributes<HTMLInputElement> & {
  label: string;
  hint?: string;
  error?: string;
  fieldClass?: string;
};

export function Input(props: InputProps) {
  const input = omit(
    props,
    "id",
    "label",
    "hint",
    "error",
    "fieldClass",
    "class",
    "aria-describedby",
    "aria-invalid",
  );
  const generatedId = createUniqueId();
  const id = () => props.id ?? `input-${generatedId}`;
  const describedBy = () =>
    [
      props["aria-describedby"],
      props.hint ? `${id()}-hint` : undefined,
      props.error ? `${id()}-error` : undefined,
    ]
      .filter(Boolean)
      .join(" ") || undefined;

  return (
    <div class={["hostling-input-field", props.fieldClass]}>
      <label for={id()}>{props.label}</label>
      <input
        {...input}
        id={id()}
        class={["hostling-input", props.class]}
        aria-describedby={describedBy()}
        aria-invalid={props.error ? "true" : props["aria-invalid"]}
      />
      <Show when={props.hint}>
        <span class="hostling-input-hint" id={`${id()}-hint`}>
          {props.hint}
        </span>
      </Show>
      <Show when={props.error}>
        <span class="hostling-input-error" id={`${id()}-error`} role="alert">
          {props.error}
        </span>
      </Show>
    </div>
  );
}

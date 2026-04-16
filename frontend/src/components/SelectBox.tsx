import { For } from 'solid-js';
import './SelectBox.css';

export interface SelectBoxOption {
  value: string;
  label: string;
}

interface SelectBoxProps {
  id?: string;
  value: string;
  options: SelectBoxOption[];
  onChange: (value: string) => void;
}

export function SelectBox(props: SelectBoxProps) {
  return (
    <select
      class="select-box"
      id={props.id}
      value={props.value}
      onChange={(e) => props.onChange(e.currentTarget.value)}
    >
      <For each={props.options}>
        {(opt) => <option value={opt.value}>{opt.label}</option>}
      </For>
    </select>
  );
}

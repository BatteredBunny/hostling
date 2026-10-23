import { For } from "solid-js";
import { SectionPanel } from "./SectionPanel";
import "./LoadingSections.css";

// The same shell is rendered in static HTML and while an island fetches its data.
export function LoadingSections(props: { sections: string[] }) {
  return (
    <div
      class="loading-sections"
      role="status"
      aria-label="Loading page"
      aria-busy="true"
    >
      <For each={props.sections}>
        {(title) => (
          <SectionPanel title={title} bodyClass="loading-section-body">
            <span>Loading…</span>
          </SectionPanel>
        )}
      </For>
    </div>
  );
}

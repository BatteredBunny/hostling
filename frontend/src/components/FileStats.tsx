import { onMount } from "solid-js";
import { trackPageLoad } from "../loading";
import "./FileStats.css";
import {
  statsCount,
  statsSizeTotal,
  setStatsCount,
  setStatsSizeTotal,
  setStatsTags,
} from "../store";
import { getFileStats } from "../generated";
import { apiData } from "../client";
import { countLabel, humanizeBytes } from "../utils";

export function FileStats() {
  onMount(() => void trackPageLoad(loadStats()));

  return (
    <span id="files-stats" role="status" aria-label="File statistics">
      {countLabel(statsCount(), "file")} · {humanizeBytes(statsSizeTotal())}
    </span>
  );
}

export async function loadStats() {
  try {
    const data = await apiData(getFileStats());
    setStatsCount(data.count || 0);
    setStatsSizeTotal(data.size_total || 0);
    setStatsTags(data.tags || []);
  } catch {
    console.error("Failed to load stats");
  }
}

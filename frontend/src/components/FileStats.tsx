import { onMount } from "solid-js";
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
import { humanizeBytes } from "../utils";

export function FileStats() {
  onMount(async () => {
    await loadStats();
  });

  return (
    <span id="files-stats">
      {statsCount() === 1 ? "1 file" : `${statsCount()} files`} ·{" "}
      {humanizeBytes(statsSizeTotal())}
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

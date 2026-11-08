// Explicit, shareable demo states. These switches are only imported by demo builds.
export const scenarios = [
  "default",
  "guest",
  "member",
  "empty",
  "no-providers",
  "failed-provider",
  "unlinked",
  "context-error",
  "context-loading",
  "gallery-error",
  "settings-error",
  "tokens-error",
  "admin-error",
  "action-error",
  "gallery-loading",
  "settings-loading",
  "tokens-loading",
  "admin-loading",
  "upload-progress",
  "upload-error",
] as const;
export type Scenario = (typeof scenarios)[number];
// In-app links drop `?demo=`, so the chosen scenario is kept for the tab's
// session; reloads and navigation stay in it until another `?demo=` is given.
function selectScenario(): string | null {
  if (typeof window === "undefined") return null;
  const fromUrl = new URLSearchParams(window.location.search).get("demo");
  try {
    if (fromUrl) sessionStorage.setItem("hostling-demo-scenario", fromUrl);
    else return sessionStorage.getItem("hostling-demo-scenario");
  } catch {
    // Storage can be unavailable (private mode, blocked site data).
  }
  return fromUrl;
}
const selected = selectScenario();
export function scenario(): Scenario {
  return scenarios.find((name) => name === selected) ?? "default";
}

export function pending(signal: AbortSignal): Promise<Response> {
  signal.throwIfAborted();
  return new Promise((_, reject) => {
    signal.addEventListener("abort", () => reject(signal.reason), {
      once: true,
    });
  });
}

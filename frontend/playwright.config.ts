import { defineConfig } from "@playwright/test";

// Without a baseline the suite captures into screenshots/. With
// SCREENSHOT_BASELINE set to an earlier capture, it compares against it instead
// and the changes reporter writes the differences to screenshot-changes/.
const baseline = process.env.SCREENSHOT_BASELINE;
// Rendering is only reproducible inside the Nix build sandbox; local runs show
// anti-aliasing noise, so comparisons are refused anywhere else.
if (baseline && !process.env.NIX_BUILD_TOP)
  throw new Error(
    "Screenshot comparisons only run in the Nix sandbox (nix/example-screenshots.nix with screenshotBaseline).",
  );

export default defineConfig({
  testDir: "./screenshots-tests",
  workers: 1,
  globalSetup: "./screenshots-tests/setup.ts",
  timeout: 30000,
  snapshotPathTemplate: `${baseline ?? "screenshots"}/{arg}{ext}`,
  updateSnapshots: baseline ? "none" : "all",
  expect: {
    toHaveScreenshot: {
      threshold: 0.1,
      maxDiffPixels: 0,
      animations: "disabled",
    },
  },
  reporter: [
    ["list"],
    ["html", { open: "never", outputFolder: "screenshot-report" }],
    ...(baseline
      ? [["./screenshots-tests/changes.ts", { baseline }] as [string, object]]
      : []),
  ],
  use: {
    baseURL: "http://127.0.0.1:4321",
    browserName: "chromium",
    viewport: { width: 1280, height: 900 },
    colorScheme: "dark",
    reducedMotion: "reduce",
    locale: "en-GB",
    timezoneId: "UTC",
    launchOptions: {
      executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
    },
  },
  webServer: {
    command:
      "HOSTLING_DEMO=1 node_modules/.bin/astro preview --ignore-lock --host 127.0.0.1 --port 4321",
    url: "http://127.0.0.1:4321",
    reuseExistingServer: false,
  },
});

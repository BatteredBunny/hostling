import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./screenshots-tests",
  workers: 1,
  use: {
    baseURL: "http://127.0.0.1:4321",
    browserName: "chromium",
    viewport: { width: 1280, height: 900 },
    colorScheme: "dark",
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

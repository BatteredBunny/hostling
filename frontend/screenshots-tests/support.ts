import { test as base, expect, type Locator, type Page } from "@playwright/test";
import { existsSync } from "node:fs";

const comparing = Boolean(process.env.SCREENSHOT_BASELINE);

export const test = base.extend<{ networkGuard: void }>({
  networkGuard: [
    async ({ page }, use) => {
      const errors: string[] = [];
      page.on("pageerror", (error) => errors.push(error.message));
      await page.route("**/*", async (route) => {
        const url = new URL(route.request().url());
        if (url.origin !== "http://127.0.0.1:4321" || url.pathname.startsWith("/api/")) {
          errors.push(`Unexpected network request: ${url}`);
          return route.abort();
        }
        return route.continue();
      });
      await page.clock.setFixedTime(new Date("2026-01-02T12:00:00Z"));
      await use();
      expect(errors, "The standalone demo must not need a backend or external assets").toEqual([]);
    },
    { auto: true },
  ],
});
export { expect };

export async function visit(page: Page, path: string, scenario = "default") {
  const url = new URL(path, "http://127.0.0.1:4321");
  url.searchParams.set("demo", scenario);
  await page.goto(url.href);
}
export const fileList = (page: Page) => page.getByRole("list", { name: "Files", exact: true });
export async function gallery(page: Page, path = "/gallery", scenario = "default") {
  await visit(page, path, scenario);
  await expect(fileList(page)).toHaveAttribute("aria-busy", "false");
  await expect(page.getByRole("status", { name: "File statistics" })).toContainText("12 files");
}
export async function preview(page: Page, file = "sample-1.png", name = "Mascot 1.png") {
  await gallery(page, `/gallery?file=${file}`);
  const dialog = page.getByRole("dialog", { name, exact: true });
  await expect(dialog).toBeVisible();
  return dialog;
}

// Captures into screenshots/, or compares against SCREENSHOT_BASELINE. Each shot
// is a soft assertion so one visual change doesn't stop the rest of the test, and
// the annotations tell the changes reporter which shots a test took.
export async function capture(page: Page, name: string, title: string, target?: Locator) {
  const info = test.info();
  expect(
    info.annotations.some((note) => note.type === "screenshot" && note.description === name),
    `Duplicate screenshot name: ${name}`,
  ).toBe(false);
  info.annotations.push({ type: "screenshot", description: name }, { type: "title", description: title });
  // Wait for decoded local assets, including lazy images below the viewport.
  await page.evaluate(async () => {
    await document.fonts.ready;
    await Promise.all(
      [...document.images].map(async (image) => {
        image.loading = "eager";
        await image.decode();
      }),
    );
  });
  await page.mouse.move(0, 0);
  const snapshot = `${name}.png`.split("/");
  // `kind: "screenshot"` can't take path segments; the config's single
  // snapshotPathTemplate gives every snapshot kind the same path anyway.
  const path = info.snapshotPath(...snapshot);
  const fullPage = target ? undefined : (await page.getByRole("dialog").count()) === 0;
  if (comparing && !existsSync(path)) {
    // New shot with no baseline: Playwright attaches nothing in this case, so
    // record the capture under the name the changes reporter looks for.
    const options = { animations: "disabled", fullPage } as const;
    const body = target ? await target.screenshot(options) : await page.screenshot(options);
    await info.attach(`${name}-actual.png`, { body, contentType: "image/png" });
    return;
  }
  if (target) await expect.soft(target).toHaveScreenshot(snapshot);
  else await expect.soft(page).toHaveScreenshot(snapshot, { fullPage });
  // Show fresh captures in the HTML report, which doubles as the screenshot gallery.
  if (!comparing) await info.attach(name, { path, contentType: "image/png" });
}

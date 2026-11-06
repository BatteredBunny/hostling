import { expect, test, type Page } from "@playwright/test";
import { mkdir } from "node:fs/promises";

async function capture(page: Page, name: string) {
  await page.evaluate(async () => {
    await document.fonts.ready;
    await Promise.all(
      [...document.images].map(async (image) => {
        // Include offscreen lazy thumbnails in readiness checks.
        image.loading = "eager";
        await image.decode();
      }),
    );
  });
  await page.screenshot({
    path: `screenshots/${name}.png`,
    animations: "disabled",
  });
}

test("standalone demo renders the four screenshot views", async ({ page }) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.route("**/*", async (route) => {
    const url = new URL(route.request().url());
    if (
      url.origin !== "http://127.0.0.1:4321" ||
      url.pathname.startsWith("/api/")
    ) {
      errors.push(`Unexpected network request: ${url}`);
      return route.abort();
    }
    return route.continue();
  });
  await page.clock.setFixedTime(new Date("2026-01-02T12:00:00Z"));
  await mkdir("screenshots", { recursive: true });
  await page.goto("/");
  await expect(
    page
      .getByRole("form", { name: "Upload file", exact: true })
      .getByRole("button", { name: "Upload File", exact: true }),
  ).toBeEnabled();
  await capture(page, "upload");
  await page.goto("/gallery");
  await expect(
    page
      .getByRole("list", { name: "Files", exact: true })
      .getByRole("listitem"),
  ).toHaveCount(8);
  await expect(
    page.getByRole("status", { name: "File statistics", exact: true }),
  ).toContainText("12 files");
  await capture(page, "gallery");
  await page
    .getByRole("button", { name: "Preview Mascot 1.png", exact: true })
    .press("Enter");
  await expect(
    page.getByRole("dialog", { name: "Mascot 1.png", exact: true }),
  ).toBeVisible();
  await capture(page, "modal");
  await page.goto("/admin");
  await expect(
    page
      .getByRole("region", { name: "Users", exact: true })
      .getByRole("article", { name: "User 1", exact: true }),
  ).toBeVisible();
  await capture(page, "admin");
  expect(errors).toEqual([]);
});

test("gallery paginates and reload restores fixture data", async ({ page }) => {
  const files = page
    .getByRole("list", { name: "Files", exact: true })
    .getByRole("listitem");
  const pagination = page.getByRole("status", { name: "Pagination" });
  await page.goto("/gallery");
  await expect(files).toHaveCount(8);
  await page.getByRole("button", { name: "Next", exact: true }).click();
  await expect(files).toHaveCount(4);
  await expect(pagination).toHaveText("9-12 of 12");
  await page.goto("/gallery");
  await page
    .getByRole("button", { name: "Actions for Mascot 1.png", exact: true })
    .press("Enter");
  await page.getByRole("menuitem", { name: "Delete", exact: true }).click();
  await page
    .getByRole("dialog", { name: "Delete file?", exact: true })
    .getByRole("button", { name: "Delete", exact: true })
    .click();
  await expect(pagination).toHaveText("1-8 of 11");
  await page.reload();
  await expect(files).toHaveCount(8);
  await expect(pagination).toHaveText("1-8 of 12");
});

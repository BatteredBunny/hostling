import {
  test,
  expect,
  capture,
  visit,
  gallery,
  preview,
  fileList,
} from "./support";
import { resolve } from "node:path";

const sample = resolve("demo-public/demo-assets/sample-1.png");
async function uploadReady(
  page: import("@playwright/test").Page,
  state = "default",
) {
  await visit(page, "/", state);
  await expect(
    page.getByRole("button", { name: "Upload File", exact: true }),
  ).toBeEnabled();
}

for (const state of [
  "default",
  "guest",
  "selected",
  "progress",
  "success",
  "error",
  "focus",
] as const) {
  test(`upload: ${state}`, async ({ page }) => {
    await uploadReady(
      page,
      state === "progress"
        ? "upload-progress"
        : state === "error"
          ? "upload-error"
          : state === "guest"
            ? "guest"
            : "default",
    );
    if (state === "guest")
      await expect(
        page.getByLabel("Upload token", { exact: true }),
      ).toBeVisible();
    if (["selected", "progress", "success", "error"].includes(state))
      await page.getByLabel("File to upload").setInputFiles(sample);
    if (["progress", "success", "error"].includes(state))
      await page
        .getByRole("button", { name: "Upload File", exact: true })
        .click();
    if (state === "progress")
      await expect(
        page.getByRole("progressbar", { name: "Upload progress" }),
      ).toHaveAttribute("aria-valuenow", "64");
    if (state === "success")
      await expect(
        page.getByText("Upload successful", { exact: true }),
      ).toBeVisible();
    if (state === "error")
      await expect(page.getByRole("alert")).toContainText(
        "Demo request failed",
      );
    if (state === "focus") {
      await page
        .getByRole("button", { name: "Drop file here or click to select" })
        .focus();
      await expect(
        page.getByRole("button", { name: "Drop file here or click to select" }),
      ).toBeFocused();
    }
    await capture(page, `upload/${state}`, `Upload: ${state}`);
  });
}

test("gallery: main states, menus, and filters", async ({ page }) => {
  await gallery(page);
  await expect(fileList(page).getByRole("listitem")).toHaveCount(8);
  await capture(page, "gallery/default", "Mixed media library");
  await page.getByRole("button", { name: "Filters", exact: true }).click();
  const filters = page.getByRole("dialog", { name: "Filters" });
  await expect(
    filters.getByRole("option", { name: "art", exact: true }),
  ).toBeVisible();
  await capture(page, "gallery/filters-open", "Filters popover");
  await filters.getByRole("radio", { name: "Private", exact: true }).click();
  await filters.getByRole("option", { name: "art", exact: true }).click();
  await expect(fileList(page).getByRole("listitem")).toHaveCount(2);
  await capture(
    page,
    "gallery/filters-selected",
    "Private files with the art tag",
  );
  await filters
    .getByRole("textbox", { name: "Search tags" })
    .fill("nonexistent");
  await expect(filters.getByRole("status")).toHaveText("No matching tags");
  await capture(page, "gallery/tag-search-empty", "Tag search with no matches");
  await filters.getByRole("button", { name: "Close filters" }).click();
  await capture(page, "gallery/filtered-results", "Filtered library");
  await gallery(page);
  await page.getByRole("button", { name: "Sort files", exact: true }).click();
  await expect(page.getByRole("listbox", { name: "Sort files" })).toBeVisible();
  await capture(page, "gallery/sort-open", "Sort options");
  await page.getByRole("option", { name: "Most Views", exact: true }).click();
  await expect(fileList(page).getByRole("listitem").first()).toContainText(
    "Meeting notes.txt",
  );
  await capture(page, "gallery/sorted-results", "Library sorted by views");
  await gallery(page);
  await page
    .getByRole("button", { name: "Actions for Mascot 1.png", exact: true })
    .press("Enter");
  const menu = page.getByRole("menu", { name: "Mascot 1.png", exact: true });
  await expect(menu).toBeVisible();
  await capture(page, "gallery/file-menu", "File actions", menu);
  await menu.getByRole("menuitem", { name: "Delete", exact: true }).click();
  await expect(
    page.getByRole("dialog", { name: "Delete file?", exact: true }),
  ).toBeVisible();
  await capture(
    page,
    "gallery/delete-confirmation",
    "Delete file confirmation",
  );
});

for (const state of [
  "empty",
  "no-results",
  "last-page",
  "error",
  "loading",
] as const) {
  test(`gallery: ${state}`, async ({ page }) => {
    const path =
      state === "no-results"
        ? "/gallery?tag=art&tag=travel"
        : state === "last-page"
          ? "/gallery?page=2"
          : "/gallery";
    await visit(
      page,
      path,
      state === "empty"
        ? "empty"
        : state === "error"
          ? "gallery-error"
          : state === "loading"
            ? "gallery-loading"
            : "default",
    );
    if (state === "loading")
      await expect(fileList(page)).toHaveAttribute("aria-busy", "true");
    else if (state === "last-page")
      await expect(page.getByRole("status", { name: "Pagination" })).toHaveText(
        "9-12 of 12",
      );
    else
      await expect(
        page.getByText(
          state === "empty"
            ? "No files uploaded yet."
            : state === "no-results"
              ? "No files match these filters."
              : "Failed to load files.",
          { exact: true },
        ),
      ).toBeVisible();
    await capture(page, `gallery/${state}`, `Gallery: ${state}`);
  });
}

for (const [state, file, name] of [
  ["image", "sample-1.png", "Mascot 1.png"],
  ["video", "clip.webm", "Short film.webm"],
  ["audio", "tone.wav", "Sound sample.wav"],
  ["document", "notes.txt", "Project notes.txt"],
  ["private-expiring", "copy-7-sample-1.png", "Private mascot.png"],
] as const) {
  test(`preview: ${state}`, async ({ page }) => {
    const dialog = await preview(page, file, name);
    if (state === "video" || state === "audio") {
      const media = dialog.getByLabel(
        `${state === "video" ? "Video" : "Audio"} preview: ${name}`,
      );
      await expect
        .poll(() =>
          media.evaluate((element) => (element as HTMLMediaElement).readyState),
        )
        .toBeGreaterThanOrEqual(2);
      if (state === "video") {
        // Decode a frame and settle native controls before capturing a paused video.
        await media.evaluate(async (element) => {
          const video = element as HTMLVideoElement;
          video.muted = true;
          await video.play();
          await new Promise<void>((resolve) =>
            video.requestVideoFrameCallback(() => resolve()),
          );
          video.pause();
          await new Promise<void>((resolve) => {
            video.addEventListener("seeked", () => resolve(), { once: true });
            video.currentTime = 0.5;
          });
        });
      }
    }
    if (state === "document")
      await expect(
        dialog.getByRole("link", { name: "Download Project notes.txt" }),
      ).toBeVisible();
    if (state === "private-expiring")
      await expect(dialog.getByText("Private", { exact: true })).toBeVisible();
    await capture(page, `preview/${state}`, `File preview: ${state}`);
  });
}

test("preview: tag suggestions", async ({ page }) => {
  const dialog = await preview(page);
  await dialog.getByRole("combobox", { name: "Add tag" }).fill("ar");
  await expect(
    dialog.getByRole("option", { name: "art", exact: true }),
  ).toBeVisible();
  await capture(page, "preview/tag-suggestions", "Tag autocomplete");
});

for (const state of [
  "default",
  "no-providers",
  "context-error",
  "context-loading",
] as const) {
  test(`login: ${state}`, async ({ page }) => {
    await visit(page, "/login", state);
    const panel = page.getByRole("region", { name: "Welcome back" });
    if (state === "default")
      await expect(
        panel.getByRole("link", { name: "Log in with GitHub" }),
      ).toBeVisible();
    else
      await expect(
        panel.getByText(
          state === "no-providers"
            ? /No authentication providers/
            : state === "context-error"
              ? /Could not load server information/
              : "Loading login providers…",
        ),
      ).toBeVisible();
    await capture(page, `login/${state}`, `Login: ${state}`);
  });
}

test("registration: invitation and validation", async ({ page }) => {
  await visit(page, "/register");
  await expect(
    page.getByText("Enter your invitation code to join Hostling."),
  ).toBeVisible();
  await capture(page, "registration/default", "Invitation registration");
  await page
    .getByLabel("Invitation code", { exact: true })
    .fill("EXPIRED-DEMO-CODE");
  await page
    .getByRole("button", { name: "Create account", exact: true })
    .click();
  await expect(page.getByRole("alert")).toContainText("invalid or expired");
  await capture(page, "registration/rejected", "Rejected invitation");
});

for (const state of [
  "default",
  "unlinked",
  "no-providers",
  "settings-error",
  "settings-loading",
] as const) {
  test(`settings: ${state}`, async ({ page }) => {
    await visit(page, "/settings", state);
    if (state === "settings-error")
      await expect(page.getByRole("alert")).toContainText(
        "Could not load settings",
      );
    else if (state === "settings-loading")
      await expect(
        page.getByRole("status", { name: "Loading page" }),
      ).toBeVisible();
    else {
      await expect(
        page.getByRole("region", { name: "Account", exact: true }),
      ).toBeVisible();
      if (state === "unlinked")
        await expect(
          page.getByText(
            "Link your account with at least one platform so you can log in again.",
            {
              exact: true,
            },
          ),
        ).toBeVisible();
    }
    await capture(page, `settings/${state}`, `Settings: ${state}`);
  });
}

for (const [state, action, title] of [
  ["unlink-confirmation", "Unlink GitHub", "Unlink provider?"],
  ["delete-account", "Delete my account", "Confirm deletion"],
  ["delete-files", "Delete all files", "Confirm deletion"],
] as const) {
  test(`settings: ${state}`, async ({ page }) => {
    await visit(page, "/settings");
    await page.getByRole("button", { name: action, exact: true }).click();
    await expect(
      page.getByRole("dialog", { name: title, exact: true }),
    ).toBeVisible();
    await capture(page, `settings/${state}`, action);
  });
}

for (const state of [
  "default",
  "empty",
  "tokens-error",
  "tokens-loading",
  "action-error",
] as const) {
  test(`tokens: ${state}`, async ({ page }) => {
    await visit(page, "/tokens", state);
    if (state === "tokens-error")
      await expect(page.getByRole("alert")).toContainText(
        "Could not load your tokens",
      );
    else if (state === "tokens-loading")
      await expect(page.getByRole("status")).toHaveText("Loading tokens…");
    else if (state === "empty")
      await expect(page.getByText("No upload tokens yet.")).toBeVisible();
    else {
      await expect(
        page.getByRole("article", { name: "Desktop uploader", exact: true }),
      ).toBeVisible();
      await expect(
        page.getByRole("region", { name: "Invite codes" }),
      ).toContainText("DEMO-INVITE-ONLY");
      if (state === "action-error") {
        await page.getByLabel("Token name", { exact: true }).fill("New token");
        await page
          .getByRole("button", { name: "Create upload token", exact: true })
          .click();
        await expect(page.getByRole("alert")).toContainText(
          "Demo request failed",
        );
      }
    }
    await capture(page, `tokens/${state}`, `Tokens: ${state}`);
  });
}

test("tokens: creation dialog and success feedback", async ({ page }) => {
  await visit(page, "/tokens");
  await expect(
    page.getByRole("button", { name: "Create upload token", exact: true }),
  ).toBeEnabled();
  await page
    .getByLabel("Token name", { exact: true })
    .fill("Screenshot uploader");
  await page
    .getByRole("button", { name: "Create upload token", exact: true })
    .click();
  const dialog = page.getByRole("dialog", {
    name: "Save your upload token",
    exact: true,
  });
  await expect(dialog).toContainText("DEMO-ONLY-NOT-A-REAL-UPLOAD-TOKEN");
  await capture(page, "tokens/created", "New upload token");
  await dialog.getByRole("button", { name: "Dismiss", exact: true }).click();
  await page
    .getByRole("article", { name: "Screenshot uploader", exact: true })
    .getByRole("button", { name: "Delete token", exact: true })
    .click();
  await expect(page.getByRole("status")).toContainText(
    "Deleted “Screenshot uploader”.",
  );
  await capture(page, "feedback/success-toast", "Successful token deletion");
});

for (const state of [
  "default",
  "no-providers",
  "failed-provider",
  "admin-error",
  "admin-loading",
] as const) {
  test(`admin: ${state}`, async ({ page }) => {
    await visit(page, "/admin", state);
    if (state === "admin-error")
      await expect(page.getByRole("alert")).toContainText(
        "Could not load admin data",
      );
    else if (state === "admin-loading")
      await expect(
        page.getByRole("status", { name: "Loading page" }),
      ).toBeVisible();
    else {
      await expect(
        page.getByRole("article", { name: "User 3", exact: true }),
      ).toBeVisible();
      if (state === "no-providers")
        await expect(
          page.getByText(/No authentication providers are configured/),
        ).toBeVisible();
      if (state === "failed-provider")
        await expect(
          page.getByText(/providers failed to initialize/),
        ).toBeVisible();
    }
    await capture(page, `admin/${state}`, `Admin: ${state}`);
  });
}

test("admin: destructive action confirmation", async ({ page }) => {
  await visit(page, "/admin");
  await page
    .getByRole("article", { name: "User 2", exact: true })
    .getByRole("button", { name: "Delete user", exact: true })
    .click();
  await expect(
    page.getByRole("dialog", { name: "Confirm action", exact: true }),
  ).toBeVisible();
  await capture(page, "admin/delete-user", "Delete user confirmation");
});

test("feedback: error toast", async ({ page }) => {
  await gallery(page, "/gallery", "action-error");
  await page
    .getByRole("button", { name: "Actions for Mascot 1.png", exact: true })
    .press("Enter");
  await page
    .getByRole("menuitem", { name: "Make private", exact: true })
    .click();
  await expect(page.getByRole("alert")).toContainText("Demo request failed");
  await capture(page, "feedback/error-toast", "Failed file action");
});

test("gallery paginates and reload restores fixture data", async ({ page }) => {
  await gallery(page);
  await page.getByRole("button", { name: "Next", exact: true }).click();
  await expect(fileList(page).getByRole("listitem")).toHaveCount(4);
  await gallery(page);
  await page
    .getByRole("button", { name: "Actions for Mascot 1.png", exact: true })
    .press("Enter");
  await page.getByRole("menuitem", { name: "Delete", exact: true }).click();
  await page
    .getByRole("dialog", { name: "Delete file?", exact: true })
    .getByRole("button", { name: "Delete", exact: true })
    .click();
  await expect(page.getByRole("status", { name: "Pagination" })).toHaveText(
    "1-8 of 11",
  );
  await page.reload();
  await expect(page.getByRole("status", { name: "Pagination" })).toHaveText(
    "1-8 of 12",
  );
});

test.describe("mobile", () => {
  test.use({
    viewport: { width: 390, height: 844 },
    isMobile: true,
    hasTouch: true,
  });
  for (const screen of [
    "upload",
    "gallery",
    "filters",
    "preview",
    "tokens",
    "admin",
  ] as const) {
    test(screen, async ({ page }) => {
      if (screen === "upload") await uploadReady(page);
      else if (screen === "preview") await preview(page);
      else if (screen === "gallery" || screen === "filters") {
        await gallery(page);
        if (screen === "filters") {
          await page
            .getByRole("button", { name: "Filters", exact: true })
            .click();
          await expect(
            page.getByRole("dialog", { name: "Filters" }),
          ).toBeVisible();
        }
      } else {
        await visit(page, `/${screen}`);
        await expect(
          page.getByRole("article", {
            name: screen === "tokens" ? "Desktop uploader" : "User 1",
            exact: true,
          }),
        ).toBeVisible();
      }
      await capture(page, `mobile/${screen}`, `Mobile: ${screen}`);
    });
  }
});

test.describe("light theme", () => {
  test.use({ colorScheme: "light" });
  for (const screen of ["upload", "gallery", "preview", "admin"] as const) {
    test(screen, async ({ page }) => {
      if (screen === "upload") await uploadReady(page);
      else if (screen === "gallery") await gallery(page);
      else if (screen === "preview") await preview(page);
      else {
        await visit(page, "/admin");
        await expect(
          page.getByRole("article", { name: "User 1", exact: true }),
        ).toBeVisible();
      }
      await capture(page, `light/${screen}`, `Light theme: ${screen}`);
    });
  }
});

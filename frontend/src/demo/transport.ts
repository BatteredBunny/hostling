import type {
  Files,
  FrontendContext,
  AdminOverviewOutput,
  FileStatsOutput,
  SettingsOutput,
} from "../generated";

const timestamp = "2026-01-01T12:00:00Z";
let files: Files[] = Array.from({ length: 12 }, (_, index) => ({
  CreatedAt: timestamp,
  ExpiryDate: "",
  FileName: `sample-${index + 1}.png`,
  OriginalFileName: `Mascot ${index + 1}.png`,
  FileSize: 204800 + index * 1024,
  MimeType: "image/png",
  Public: index % 2 === 0,
  Tags: [{ Name: index % 2 ? "art" : "favorites" }],
  ViewsCount: 12 + index * 7,
}));
const context: FrontendContext = {
  account_id: 1,
  branding: "Hostling",
  tagline: "Simple file hosting service",
  has_admin_warning: false,
  is_admin: true,
  logged_in: true,
  providers: ["github"],
};
const settings: SettingsOutput = {
  no_providers_configured: false,
  unlinked_account: false,
  providers: [
    { Name: "github", IsLinked: true, Username: "demo", ProfileURL: "" },
  ],
};
const json = (data: unknown, status = 200) => Response.json(data, { status });

// This transport never forwards requests to a server. State lasts for this document.
export const demoFetch: typeof fetch = async (input, init) => {
  const request = input instanceof Request ? input : new Request(input, init);
  request.signal.throwIfAborted();
  const url = new URL(request.url);
  const path = url.pathname;
  const method = request.method;
  if (method === "GET") {
    switch (path) {
      case "/api/context":
        return json(context);
      case "/api/account/settings":
        return json(settings);
      case "/api/account/upload_tokens":
        return json([]);
      case "/api/account/invite_code":
        return json({ code: "DEMO-INVITE", uses: 3, expiry_date: "" });
      case "/api/account/files/stats": {
        const stats: FileStatsOutput = {
          count: files.length,
          size_total: files.reduce((n, f) => n + f.FileSize, 0),
          tags: [
            ...new Set(files.flatMap((f) => f.Tags?.map((t) => t.Name) ?? [])),
          ],
        };
        return json(stats);
      }
      case "/api/account/files/tags":
        return json({ tags: ["art", "favorites"] });
      case "/api/account/files": {
        const tags = url.searchParams.getAll("tag");
        const filters = url.searchParams.getAll("filter");
        const filtered = files.filter(
          (f) =>
            tags.every((tag) => f.Tags?.some((t) => t.Name === tag)) &&
            filters.every((filter) =>
              filter === "public"
                ? f.Public
                : filter === "private"
                  ? !f.Public
                  : filter === "untagged"
                    ? !f.Tags?.length
                    : true,
            ),
        );
        const field = url.searchParams.get("sort");
        const direction = url.searchParams.get("desc") === "false" ? 1 : -1;
        filtered.sort(
          (a, b) =>
            direction *
            (field === "views"
              ? a.ViewsCount - b.ViewsCount
              : field === "file_size"
                ? a.FileSize - b.FileSize
                : a.CreatedAt.localeCompare(b.CreatedAt)),
        );
        const skip = Math.max(0, Number(url.searchParams.get("skip")) || 0);
        return json({
          count: filtered.length,
          files: filtered.slice(skip, skip + 8),
        });
      }
      case "/api/admin/overview": {
        const overview: AdminOverviewOutput = {
          accounts: ["demo", "artist", "guest"].map((name, i) => ({
            ID: i + 1,
            AccountType: i ? "user" : "admin",
            CreatedAt: timestamp,
            LastActivity: timestamp,
            GithubUsername: name,
            OIDCUsername: "",
            InvitedBy: i ? "demo" : "",
            FilesUploaded: i ? 0 : files.length,
            SpaceUsed: i ? 0 : files.reduce((n, f) => n + f.FileSize, 0),
            SessionsCount: 1,
            UploadTokensCount: 0,
            You: i === 0,
          })),
          failed_providers: [],
          file_storage_method: "local",
          login_providers: ["github"],
          max_upload_size: 100000000,
          no_providers_configured: false,
          version: "demo",
        };
        return json(overview);
      }
    }
  }
  if (path.startsWith("/api/account/file") && method !== "GET") {
    if (path === "/api/account/files" && method === "DELETE") {
      files = [];
      return json("Deleted");
    }
    const body = await request.formData();
    const file = files.find((f) => f.FileName === body.get("file_name"));
    if (!file) return json({ detail: "Sample file not found." }, 404);
    if (path === "/api/account/file" && method === "DELETE")
      files = files.filter((f) => f !== file);
    else if (path.endsWith("/public")) file.Public = !file.Public;
    else if (path.endsWith("/tag")) {
      const name = String(body.get("tag") ?? "");
      file.Tags =
        method === "DELETE"
          ? (file.Tags?.filter((t) => t.Name !== name) ?? [])
          : [...(file.Tags ?? []), { Name: name }];
    } else
      return json({ detail: "This action is unavailable in the demo." }, 403);
    return json("Updated");
  }
  return json({ detail: "This action is unavailable in the demo." }, 403);
};

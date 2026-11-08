import type {
  FrontendContext,
  AdminOverviewOutput,
  FileStatsOutput,
  SettingsOutput,
  CreatedUploadToken,
} from "../generated";
import {
  createFiles,
  createTokens,
  createInvites,
  timestamp,
} from "./fixtures";
import { scenario, pending } from "./scenarios";
import { rememberUpload } from "./media";

let files = scenario() === "empty" ? [] : createFiles();
let tokens = scenario() === "empty" ? [] : createTokens();
let invites = scenario() === "empty" ? [] : createInvites();
let nextTokenId = 3;
let nextUploadId = files.length + 1;
const json = (data: unknown, status = 200) => Response.json(data, { status });
const unavailable = () =>
  json({ detail: "This action is unavailable in the demo." }, 403);
const error = () =>
  json({ detail: "Demo request failed. Please try again." }, 503);
const endpointNames: Record<string, string> = {
  "/api/context": "context",
  "/api/account/files": "gallery",
  "/api/account/settings": "settings",
  "/api/account/upload_tokens": "tokens",
  "/api/admin/overview": "admin",
};
function matchingFiles(url: URL) {
  const tags = url.searchParams.getAll("tag");
  const filters = url.searchParams.getAll("filter");
  return files.filter(
    (file) =>
      tags.every((tag) => file.Tags?.some((t) => t.Name === tag)) &&
      filters.every((filter) =>
        filter === "public"
          ? file.Public
          : filter === "private"
            ? !file.Public
            : filter === "untagged"
              ? !file.Tags?.length
              : true,
      ),
  );
}

// Fake responses and local state only: there is deliberately no network fallback.
export const demoFetch: typeof fetch = async (input, init) => {
  const request = input instanceof Request ? input : new Request(input, init);
  request.signal.throwIfAborted();
  const url = new URL(request.url);
  const path = url.pathname;
  const method = request.method;
  const state = scenario();
  if (method === "GET") {
    if (state === `${endpointNames[path]}-error`) return error();
    if (state === `${endpointNames[path]}-loading`)
      return pending(request.signal);
    switch (path) {
      case "/api/context": {
        const guest =
          state === "guest" ||
          ["/login", "/register"].includes(
            window.location.pathname.replace(/\/$/, ""),
          );
        const context: FrontendContext = {
          account_id: guest ? 0 : 1,
          branding: "Hostling",
          tagline: "Simple file hosting service",
          has_admin_warning: ["no-providers", "failed-provider"].includes(
            state,
          ),
          is_admin: !guest && state !== "member",
          logged_in: !guest,
          providers:
            state === "no-providers" ? [] : ["github", "openid-connect"],
        };
        return json(context);
      }
      case "/api/account/settings": {
        const settings: SettingsOutput = {
          no_providers_configured: state === "no-providers",
          unlinked_account: state === "unlinked",
          providers:
            state === "no-providers"
              ? []
              : [
                  {
                    Name: "github",
                    IsLinked: state !== "unlinked",
                    Username: "demo",
                    ProfileURL: "",
                  },
                  {
                    Name: "openid-connect",
                    IsLinked: false,
                    Username: "",
                    ProfileURL: "",
                  },
                ],
        };
        return json(settings);
      }
      case "/api/account/upload_tokens":
        return json(tokens);
      case "/api/account/invite_code":
        return json(invites);
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
        return json({
          tags: [
            ...new Set(
              matchingFiles(url).flatMap(
                (f) => f.Tags?.map((t) => t.Name) ?? [],
              ),
            ),
          ],
        });
      case "/api/account/files": {
        const filtered = matchingFiles(url);
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
            LastActivity: i === 2 ? "0001-01-01T00:00:00Z" : timestamp,
            GithubUsername: i === 2 ? "" : name,
            OIDCUsername: i === 2 ? name : "",
            InvitedBy: i ? "demo" : "",
            FilesUploaded: i ? 0 : files.length,
            SpaceUsed: i ? 0 : files.reduce((n, f) => n + f.FileSize, 0),
            SessionsCount: i === 2 ? 0 : 1,
            UploadTokensCount: i ? 0 : tokens.length,
            You: i === 0,
          })),
          failed_providers:
            state === "failed-provider" ? ["openid-connect"] : [],
          file_storage_method: "local",
          login_providers:
            state === "no-providers" ? [] : ["github", "openid-connect"],
          max_upload_size: 100000000,
          no_providers_configured: state === "no-providers",
          version: "demo",
        };
        return json(overview);
      }
    }
  }
  if (state === "action-error") return error();
  if (path === "/api/auth/register")
    return json({ detail: "This invitation code is invalid or expired." }, 400);
  if (path === "/api/file/upload" && method === "POST") {
    if (state === "upload-error") return error();
    const body = await request.formData();
    const file = body.get("file");
    if (!(file instanceof File))
      return json({ detail: "Choose a file to upload." }, 400);
    const name = `upload-${nextUploadId++}-${file.name}`;
    rememberUpload(name, file);
    files.unshift({
      CreatedAt: timestamp,
      ExpiryDate: "",
      FileName: name,
      OriginalFileName: file.name,
      MimeType: file.type,
      FileSize: file.size,
      Public: false,
      Tags: [],
      ViewsCount: 0,
    });
    return json(name);
  }
  if (path === "/api/account/upload_tokens") {
    const body = await request.formData();
    if (method === "POST") {
      const token: CreatedUploadToken = {
        id: nextTokenId++,
        name: String(body.get("nickname")),
        created_at: timestamp,
        last_used: null,
        token: "DEMO-ONLY-NOT-A-REAL-UPLOAD-TOKEN",
      };
      tokens.unshift({
        id: token.id,
        name: token.name,
        created_at: token.created_at,
        last_used: null,
      });
      return json(token);
    }
    if (method === "DELETE") {
      tokens = tokens.filter((t) => t.id !== Number(body.get("id")));
      return json("Deleted");
    }
  }
  if (path === "/api/account/invite_code" && method === "DELETE") {
    const body = await request.formData();
    invites = invites.filter((i) => i.code !== body.get("invite_code"));
    return json("Deleted");
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
      const name = String(body.get("tag") ?? "")
        .trim()
        .toLowerCase();
      if (method === "DELETE")
        file.Tags = file.Tags?.filter((t) => t.Name !== name) ?? [];
      else if (!name) return json({ detail: "Enter a tag name." }, 400);
      else if (!file.Tags?.some((t) => t.Name === name))
        file.Tags = [...(file.Tags ?? []), { Name: name }];
    } else return unavailable();
    return json("Updated");
  }
  return unavailable();
};

export function demoUploadTransport(
  progress: (value: number) => void,
): typeof fetch {
  return async (input, init) => {
    if (scenario() === "upload-progress") {
      progress(64);
      const request =
        input instanceof Request ? input : new Request(input, init);
      return pending(request.signal);
    }
    const response = await demoFetch(input, init);
    if (response.ok) progress(100);
    return response;
  };
}

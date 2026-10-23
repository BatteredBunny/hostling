import { client } from "./generated/client.gen";

client.setConfig({
  baseUrl: "",
  credentials: "same-origin",
  cache: "no-store",
  headers: { Accept: "application/json" },
});
client.interceptors.response.use((response) => {
  if (typeof window === "undefined") return response;
  const path = new URL(response.url || "/", window.location.origin).pathname;
  if (
    response.status === 401 &&
    (path.startsWith("/api/account/") || path.startsWith("/api/admin/")) &&
    window.location.pathname !== "/login"
  ) {
    window.location.assign("/login");
  }
  return response;
});

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
  ) {
    super(message);
  }
}

function errorMessage(error: unknown): string {
  if (typeof error === "string" && error.trim()) return error.slice(0, 300);
  if (
    error &&
    typeof error === "object" &&
    "detail" in error &&
    typeof error.detail === "string"
  ) {
    if (
      error.detail === "validation failed" &&
      "errors" in error &&
      Array.isArray(error.errors)
    ) {
      const messages = error.errors.flatMap((detail: unknown) =>
        detail &&
        typeof detail === "object" &&
        "message" in detail &&
        typeof detail.message === "string"
          ? [detail.message]
          : [],
      );
      if (messages.length) return messages.join("; ").slice(0, 300);
    }
    return error.detail.slice(0, 300);
  }
  return "The request failed. Please try again.";
}

// The only shared response/error handling; routes, payloads and types are generated.
export async function apiData<T>(
  request: Promise<{ data?: T; error?: unknown; response?: Response }>,
): Promise<T> {
  const result = await request;
  if (!result.response && result.error instanceof Error) throw result.error;
  if (!result.response?.ok || result.data === undefined) {
    throw new ApiError(
      errorMessage(result.error),
      result.response?.status ?? 0,
    );
  }
  return result.data;
}

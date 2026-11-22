// The server's branding, remembered so the loading screen can show it before
// the app has asked the server again.
export const BRANDING_CACHE_KEY = "hostling-branding";
// The server's own default (see internal/app.go).
export const DEFAULT_BRANDING = "Hostling";

export function cachedBranding(): string {
  try {
    return localStorage.getItem(BRANDING_CACHE_KEY) || DEFAULT_BRANDING;
  } catch {
    return DEFAULT_BRANDING;
  }
}

export function rememberBranding(branding: string) {
  try {
    localStorage.setItem(BRANDING_CACHE_KEY, branding);
  } catch {
    // Storage may be unavailable; the default is shown instead.
  }
}

export const FILES_PER_PAGE = 8;

const dateTimeFormatter = new Intl.DateTimeFormat("en-GB", {
  dateStyle: "medium",
  timeStyle: "medium",
  hourCycle: "h23",
});
const relativeTimeFormatter = new Intl.RelativeTimeFormat("en", {
  numeric: "always",
});
// Month and year durations are approximate for relative display.
const relativeTimeUnits: readonly [Intl.RelativeTimeFormatUnit, number][] = [
  ["year", 365 * 24 * 60 * 60],
  ["month", 30 * 24 * 60 * 60],
  ["week", 7 * 24 * 60 * 60],
  ["day", 24 * 60 * 60],
  ["hour", 60 * 60],
  ["minute", 60],
  ["second", 1],
];

export function formatTimeDate(date: string): string {
  if (!date) return "";
  const timestamp = Date.parse(date);
  return Number.isFinite(timestamp) ? dateTimeFormatter.format(timestamp) : "";
}

export function relativeTime(date: string): string {
  if (!date) return "";
  const timestamp = Date.parse(date);
  if (!Number.isFinite(timestamp)) return "";
  const seconds = (timestamp - Date.now()) / 1000;
  for (const [unit, duration] of relativeTimeUnits) {
    if (Math.abs(seconds) >= duration || unit === "second") {
      const value = Math.trunc(seconds / duration);
      return relativeTimeFormatter.format(
        seconds <= 0 && value === 0 ? -0 : value,
        unit,
      );
    }
  }
  return "";
}

export function humanizeBytes(bytes: number): string {
  if (!bytes || bytes <= 0) return "0 B";

  const k = 1000;
  const sizes = ["B", "kB", "MB", "GB", "TB", "PB"];
  const i = Math.min(
    Math.floor(Math.log(bytes) / Math.log(k)),
    sizes.length - 1,
  );

  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

export function mimeIsImage(mimeType: string): boolean {
  return mimeType?.startsWith("image/") ?? false;
}

export function mimeIsVideo(mimeType: string): boolean {
  return mimeType?.startsWith("video/") ?? false;
}

export function mimeIsAudio(mimeType: string): boolean {
  return mimeType?.startsWith("audio/") ?? false;
}

export function hashStringToHSL(str: string): {
  h: number;
  s: number;
  l: number;
} {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
    hash |= 0;
  }

  const h = Math.abs(hash) % 360;
  const s = 65 + (Math.abs(hash >> 8) % 20);
  const l = 45 + (Math.abs(hash >> 16) % 15);

  return { h, s, l };
}

export function hasExpiry(expiryDate: string): boolean {
  if (!expiryDate) return false;
  const t = new Date(expiryDate).getTime();
  return Number.isFinite(t) && t > 0;
}

export function fileUrl(fileName: string): string {
  return `/${encodeURIComponent(fileName)}`;
}

export async function copyText(
  text: string,
  container: HTMLElement = document.querySelector<HTMLElement>(
    "dialog:modal",
  ) ?? document.body,
) {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    const previousFocus = document.activeElement;
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.style.position = "fixed";
    textarea.style.opacity = "0";
    container.appendChild(textarea);
    try {
      textarea.select();
      if (!document.execCommand("copy")) throw new Error("Copy failed");
    } finally {
      textarea.remove();
      if (previousFocus instanceof HTMLElement && previousFocus.isConnected)
        previousFocus.focus();
    }
  }
}

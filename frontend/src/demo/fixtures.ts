import type { Files, UiUploadToken, InviteCodeApiOutput } from "../generated";
export const timestamp = "2026-01-01T12:00:00Z";
const media = [
  ["sample-1.png", "Mascot 1.png", "image/png", "favorites"],
  ["sample-1.png", "Mascot 2.png", "image/png", "travel"],
  ["sample-1.png", "Mascot poster.png", "image/png", "art"],
  ["sample-1.png", "Mascot sketch.png", "image/png", "art"],
  ["clip.webm", "Short film.webm", "video/webm", "video"],
  ["tone.wav", "Sound sample.wav", "audio/wav", "audio"],
  ["notes.txt", "Project notes.txt", "text/plain", "work"],
  ["sample-1.png", "Private mascot.png", "image/png", ""],
  ["sample-1.png", "Weekend trip.png", "image/png", "travel"],
  ["sample-1.png", "Poster draft.png", "image/png", "art"],
  ["sample-1.png", "Color study.png", "image/png", ""],
  ["notes.txt", "Meeting notes.txt", "text/plain", "work"],
];
export function createFiles(): Files[] {
  const seen = new Set<string>();
  return media.map(([asset, name, mime, tag], index) => ({
    CreatedAt: new Date(Date.parse(timestamp) - index * 3600000).toISOString(),
    ExpiryDate: index === 7 ? "2026-02-01T12:00:00Z" : "",
    // Reused assets get a unique name; media.ts strips the `copy-N-` prefix.
    FileName: seen.has(asset) ? `copy-${index}-${asset}` : (seen.add(asset), asset),
    OriginalFileName: name,
    FileSize: 204800 + index * 8192,
    MimeType: mime,
    Public: index % 2 === 0,
    Tags: tag ? [{ Name: tag }] : [],
    ViewsCount: 12 + index * 7,
  }));
}
export function createTokens(): UiUploadToken[] {
  return [
    {
      id: 1,
      name: "Desktop uploader",
      created_at: timestamp,
      last_used: timestamp,
    },
    { id: 2, name: "Backup script", created_at: timestamp, last_used: null },
  ];
}
export function createInvites(): InviteCodeApiOutput[] {
  return [
    { code: "DEMO-INVITE-ONLY", uses: 3, expiry_date: "2026-02-01T12:00:00Z" },
  ];
}

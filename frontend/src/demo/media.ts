const uploads = new Map<string, string>();
export function demoFileUrl(name: string): string {
  return (
    uploads.get(name) ??
    `/demo-assets/${encodeURIComponent(name.replace(/^copy-\d+-/, ""))}`
  );
}
export function rememberUpload(name: string, file: File) {
  uploads.set(name, URL.createObjectURL(file));
}

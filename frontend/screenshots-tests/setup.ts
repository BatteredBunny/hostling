import { rm } from "node:fs/promises";

export default async function setup() {
  if (!process.env.SCREENSHOT_BASELINE) await rm("screenshots", { recursive: true, force: true });
}

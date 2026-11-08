import { defineConfig, fontProviders } from "astro/config";
import solid from "@astrojs/solid-js";
import { cpSync, readFileSync, statSync } from "node:fs";
import { extname, join, normalize } from "node:path";
import { fileURLToPath } from "node:url";

const demo = process.env.HOSTLING_DEMO === "1";

// Demo builds keep the normal public/ files (favicon, robots.txt, font licence)
// and add the sample media from demo-public/ on top.
function demoAssets() {
  const source = fileURLToPath(new URL("./demo-public", import.meta.url));
  const types = {
    ".png": "image/png",
    ".svg": "image/svg+xml",
    ".webm": "video/webm",
    ".wav": "audio/wav",
    ".txt": "text/plain; charset=utf-8",
  };
  return {
    name: "hostling-demo-assets",
    hooks: {
      "astro:server:setup": ({ server }) => {
        server.middlewares.use((req, res, next) => {
          const path = normalize(
            decodeURIComponent(new URL(req.url ?? "/", "http://x").pathname),
          );
          const file = join(source, path);
          if (!file.startsWith(source + "/")) return next();
          try {
            if (!statSync(file).isFile()) return next();
            res.setHeader(
              "Content-Type",
              types[extname(file)] ?? "application/octet-stream",
            );
            res.end(readFileSync(file));
          } catch {
            next();
          }
        });
      },
      "astro:build:done": ({ dir }) => {
        cpSync(source, fileURLToPath(dir), { recursive: true });
      },
    },
  };
}

export default defineConfig({
  output: "static",
  publicDir: "./public",
  prefetch: { prefetchAll: true, defaultStrategy: "hover" },
  integrations: demo ? [solid(), demoAssets()] : [solid()],
  fonts: [
    {
      name: "Nordwand Mono",
      cssVariable: "--font-nordwand-mono",
      provider: fontProviders.local(),
      fallbacks: ["monospace"],
      options: {
        variants: [
          {
            src: [
              "./src/assets/fonts/nordwand-mono/NordwandMono-Regular.woff2",
            ],
            weight: 400,
            style: "normal",
          },
          {
            src: ["./src/assets/fonts/nordwand-mono/NordwandMono-Medium.woff2"],
            weight: 500,
            style: "normal",
          },
          {
            src: [
              "./src/assets/fonts/nordwand-mono/NordwandMono-SemiBold.woff2",
            ],
            weight: 600,
            style: "normal",
          },
          {
            src: ["./src/assets/fonts/nordwand-mono/NordwandMono-Bold.woff2"],
            weight: 700,
            style: "normal",
          },
          {
            src: ["./src/assets/fonts/nordwand-mono/NordwandMono-Italic.woff2"],
            weight: 400,
            style: "italic",
          },
        ],
      },
    },
  ],
  outDir: demo ? "./dist-demo" : "../public/dist",
  build: { assets: "_astro", assetsPrefix: demo ? undefined : "/assets" },
  vite: {
    define: { "import.meta.env.PUBLIC_DEMO": JSON.stringify(demo) },
    // Development proxy only
    server: {
      proxy: demo
        ? {}
        : Object.fromEntries(
            ["/api", "/logout"].map((path) => [
              path,
              process.env.HOSTLING_BACKEND_URL || "http://localhost:8081",
            ]),
          ),
    },
  },
});

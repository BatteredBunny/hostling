import { defineConfig, fontProviders } from "astro/config";
import solid from "@astrojs/solid-js";

export default defineConfig({
  output: "static",
  prefetch: { prefetchAll: true, defaultStrategy: "hover" },
  integrations: [solid()],
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
  outDir: "../public/dist",
  build: { assets: "_astro", assetsPrefix: "/assets" },
  vite: {
    // Development proxy only
    server: {
      proxy: Object.fromEntries(
        ["/api", "/logout"].map((path) => [
          path,
          process.env.HOSTLING_BACKEND_URL || "http://localhost:8081",
        ]),
      ),
    },
  },
});

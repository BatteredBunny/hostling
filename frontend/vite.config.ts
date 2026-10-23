import { defineConfig } from 'vite';
import solid from 'vite-plugin-solid';

export default defineConfig({
  plugins: [solid()],
  build: {
    outDir: '../public/dist',
    emptyOutDir: true,
    rollupOptions: {
      input: { fileLibrary: './src/index.tsx', tokens: './src/tokens.tsx' },
      output: {
        entryFileNames: '[name].js',
        assetFileNames: '[name].[ext]',
      },
    },
  },
});

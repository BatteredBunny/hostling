import { defineConfig } from 'vite';
import solid from 'vite-plugin-solid';

export default defineConfig({
  plugins: [solid()],
  build: {
    outDir: '../public/dist',
    emptyOutDir: true,
    lib: {
      entry: './src/index.tsx',
      name: 'FileLibrary',
      fileName: 'fileLibrary',
      formats: ['es'],
    },
    rollupOptions: {
      output: {
        entryFileNames: 'fileLibrary.js',
        assetFileNames: 'fileLibrary.[ext]',
      },
    },
  },
});

import { defineConfig } from 'vite';
import solidOxc from '@oxc-solid-js/vite';

export default defineConfig({
  plugins: [solidOxc()],
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

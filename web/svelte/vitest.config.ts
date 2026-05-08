import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { resolve } from 'node:path';

export default defineConfig({
  plugins: [svelte({ hot: !process.env.VITEST })],
  resolve: {
    alias: {
      $lib: resolve(__dirname, 'src/lib'),
      '$app/navigation': resolve(__dirname, 'src/test/sveltekit-mocks/navigation.ts'),
      '$app/stores': resolve(__dirname, 'src/test/sveltekit-mocks/stores.ts'),
    },
    conditions: ['browser'],
  },
  test: {
    environment: 'happy-dom',
    setupFiles: ['./src/test/setup.ts'],
    globals: true,
    include: ['src/**/*.{test,spec}.{js,ts}'],
  },
});

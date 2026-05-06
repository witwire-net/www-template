import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

const svelteRuntimeChunkPattern =
  /[/\\]svelte[/\\]src[/\\](?:internal[/\\]client[/\\]runtime|index-client)\.js$/;

export default defineConfig({
  plugins: [sveltekit()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (svelteRuntimeChunkPattern.test(id)) {
            return 'svelte-runtime';
          }

          return undefined;
        },
      },
    },
  },
  optimizeDeps: {
    exclude: ['@opentelemetry/sdk-trace-base'],
  },
  server: {
    host: '0.0.0.0',
    port: 5174,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  preview: {
    host: '0.0.0.0',
    port: 4174,
  },
});

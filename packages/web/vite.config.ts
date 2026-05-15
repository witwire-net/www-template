import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

const svelteRuntimeChunkPattern =
  /[/\\]svelte[/\\]src[/\\](?:internal[/\\]client[/\\]runtime|index-client)\.js$/;

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
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
    port: 5173,
    host: '0.0.0.0',
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
});

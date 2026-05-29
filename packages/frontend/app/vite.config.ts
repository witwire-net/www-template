import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig, type Plugin } from 'vite';

const svelteRuntimeChunkPattern =
  /[/\\]svelte[/\\]src[/\\](?:internal[/\\]client[/\\]runtime|index-client)\.js$/;
const loopbackDevHosts = new Set(['localhost', '127.0.0.1', '[::1]']);

function canonicalHostRedirectPlugin(canonicalHost: string, canonicalPort: number): Plugin {
  // Step 1: dev server middleware でだけ使う canonical port を文字列化し、Host header 比較の揺れをなくす。
  const canonicalPortString = String(canonicalPort);
  const canonicalOrigin = `http://${canonicalHost}:${canonicalPortString}`;

  return {
    name: `www-template-canonical-host:${canonicalHost}`,
    configureServer(server) {
      server.middlewares.use((request, response, next) => {
        // Step 2: Host header が欠けている request は Vite の通常処理に委譲し、壊れた redirect を作らない。
        const hostHeader = request.headers.host;
        if (hostHeader == null) {
          next();
          return;
        }

        // Step 3: loopback host かつ対象 dev port の場合だけ canonical host へ寄せ、WebAuthn RP ID と origin を一致させる。
        const host = parseHostHeader(hostHeader);
        if (host?.port !== canonicalPortString || !loopbackDevHosts.has(host.hostname)) {
          next();
          return;
        }

        // Step 4: path と query を維持した 308 redirect を返し、誤った localhost origin を history に積ませない。
        const redirectURL = new URL(request.url ?? '/', canonicalOrigin);
        response.statusCode = 308;
        response.setHeader('Location', redirectURL.href);
        response.end();
      });
    },
  };
}

function parseHostHeader(hostHeader: string): { hostname: string; port: string } | null {
  try {
    // Step 1: Host header は IPv6 bracket や port を含むため、URL parser に委譲して安全に分解する。
    const parsed = new URL(`http://${hostHeader}`);
    return { hostname: parsed.hostname, port: parsed.port };
  } catch {
    // Step 2: 不正な Host header は redirect 対象にせず、Vite の通常の host validation に委譲する。
    return null;
  }
}

export default defineConfig({
  plugins: [canonicalHostRedirectPlugin('app.localhost', 5174), tailwindcss(), sveltekit()],

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
    strictPort: true,
    allowedHosts: ['app.localhost'],
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

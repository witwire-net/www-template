<svelte:head>
  <title>www-template</title>
  <meta
    name="description"
    content="www-template の公開 SSR ルートを提供する SvelteKit フロントエンド"
  />
</svelte:head>
<script lang="ts">
  import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
  import '@www-template/ui/styles.css';
  import '../app.css';
  import { useObservability } from '$lib/observability.svelte';
  import type { Snippet } from 'svelte';

  type NavLink = {
    href: string;
    label: string;
  };

  let { children }: { children: Snippet } = $props();

  useObservability('www-template-web');

  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        gcTime: 300_000,
        refetchOnWindowFocus: false,
      },
    },
  });

  const links: NavLink[] = [
    { href: '/', label: 'Home' },
  ];
</script>

<QueryClientProvider client={queryClient}>
  <div class="web-layout">
    <header class="web-layout__topbar">
      <a class="web-layout__brand" href="/">www-template</a>
      <nav class="web-layout__nav">
        {#each links as link (link.href)}
          <a class="web-layout__nav-link" href={link.href}>{link.label}</a>
        {/each}
      </nav>
    </header>

    <main class="web-layout__content">
      {@render children()}
    </main>
  </div>
</QueryClientProvider>

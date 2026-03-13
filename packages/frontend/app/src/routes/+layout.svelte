<svelte:head>
  <title>www-template</title>
  <meta
    name="description"
    content="www-template の公開 SSR ルートと /app/* CSR ルートを同居させる SvelteKit フロントエンド"
  />
</svelte:head>
<script lang="ts">
  import type { Snippet } from 'svelte';

  type NavLink = {
    href: string;
    label: string;
  };

  let { children }: { children: Snippet } = $props();

  const links: NavLink[] = [
    { href: '/', label: 'Home' },
    { href: '/timeline', label: 'Timeline' },
    { href: '/app/profiles', label: 'App Profiles' },
  ];
</script>

<div class="shell">
  <header class="topbar">
    <a class="brand" href="/">www-template</a>
    <nav>
      {#each links as link (link.href)}
        <a href={link.href}>{link.label}</a>
      {/each}
    </nav>
  </header>

  <main class="content">
    {@render children()}
  </main>
</div>

<style>
  :global(body) {
    margin: 0;
    background:
      radial-gradient(circle at top, rgba(245, 158, 11, 0.18), transparent 30%),
      linear-gradient(180deg, #fff7ed 0%, #fff 44%, #f8fafc 100%);
    color: #0f172a;
    font-family: 'Avenir Next', 'Hiragino Sans', 'Yu Gothic', sans-serif;
  }

  :global(button) {
    font: inherit;
  }

  :global(a) {
    color: inherit;
    text-decoration: none;
  }

  .shell {
    min-height: 100vh;
  }

  .topbar {
    position: sticky;
    top: 0;
    z-index: 10;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 1rem clamp(1rem, 4vw, 2rem);
    border-bottom: 1px solid rgba(15, 23, 42, 0.06);
    background: rgba(255, 255, 255, 0.72);
    backdrop-filter: blur(16px);
  }

  .brand {
    font-size: 1.1rem;
    font-weight: 800;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    border: none;
    background: transparent;
    cursor: pointer;
  }

  nav {
    display: flex;
    flex-wrap: wrap;
    gap: 0.85rem;
  }

  nav a {
    padding: 0.5rem 0.8rem;
    border-radius: 999px;
    background: transparent;
    color: #334155;
    font-weight: 600;
  }

  nav a:hover {
    background: rgba(15, 23, 42, 0.06);
  }

  .content {
    width: min(1120px, calc(100vw - 2rem));
    margin: 0 auto;
    padding: clamp(1.25rem, 4vw, 3rem) 0 4rem;
  }
</style>

<svelte:head>
  <title>www-template</title>
  <meta
    name="description"
    content="www-template の公開 SSR ルートを提供し、認証付き `/app/*` は別CSRアプリへ委譲する SvelteKit フロントエンド"
  />
</svelte:head>
<script lang="ts">
  import '@www-template-frontend/ui/styles';
  import type { Snippet } from 'svelte';

  type NavLink = {
    href: string;
    label: string;
  };

  let { children }: { children: Snippet } = $props();

  const links: NavLink[] = [
    { href: '/', label: 'Home' },
    { href: '/app/login', label: 'Login' },
    { href: '/app', label: 'App' },
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

  <footer class="site-footer">
    <a class="footer-link" href="/app/login" aria-label="ログイン">ログイン</a>
  </footer>
</div>

<style>
  :global(body) {
    margin: 0;
    background:
      radial-gradient(circle at top, color-mix(in srgb, var(--color-primary) 18%, transparent), transparent 30%),
      linear-gradient(180deg, var(--color-background) 0%, var(--color-surface) 44%, var(--color-background) 100%);
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
    gap: var(--spacing-md);
    padding: var(--spacing-md) clamp(1rem, 4vw, 2rem);
    border-bottom: 1px solid var(--color-border-subtle);
    background: color-mix(in srgb, var(--color-surface) 72%, transparent);
    backdrop-filter: blur(16px);
  }

  .brand {
    font-family: var(--font-family-display);
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
    padding: var(--spacing-sm) 0.8rem;
    border-radius: var(--radius-full);
    background: transparent;
    color: var(--color-text-secondary);
    font-weight: 600;
  }

  nav a:hover {
    background: var(--color-surface-hover);
  }

  .content {
    width: min(1120px, calc(100vw - 2rem));
    margin: 0 auto;
    padding: clamp(1.25rem, 4vw, 3rem) 0 4rem;
  }

  .site-footer {
    display: flex;
    justify-content: center;
    gap: var(--spacing-md);
    padding: var(--spacing-md);
    border-top: 1px solid var(--color-border-subtle);
  }

  .footer-link {
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-regular);
    color: var(--color-text-muted);
    text-decoration: none;
  }

  .footer-link:hover {
    color: var(--color-text-secondary);
  }

  .footer-link:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
    border-radius: 0.25rem;
  }
</style>

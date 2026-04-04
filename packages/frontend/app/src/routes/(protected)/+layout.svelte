<script lang="ts">
  import type { Snippet } from 'svelte';

  import { useSessionGuard } from '@www-template/domain/hooks/auth/useSessionGuard';

  let { children }: { children: Snippet } = $props();

  const { data } = useSessionGuard({
    readPathname: () => window.location.pathname,
    redirectTo: (intent) => {
      window.location.href = intent;
    },
  });
</script>

{#if data.state.phase === 'authenticated' && data.state.session !== null}
  <section class="app-shell">
    <div class="banner">
      <div>
        <div class="eyebrow">SVELTEKIT SPA SHELL</div>
        <h1>認証済みアプリ</h1>
      </div>
      <div class="links">
        <a href="/">Overview</a>
        <a href="/logout">ログアウト</a>
      </div>
    </div>

    <div class="panel">
      {@render children()}
    </div>
  </section>
{:else}
  <div class="loading-shell">
    <p>セッションを確認しています…</p>
  </div>
{/if}

<style>
  .app-shell {
    display: grid;
    gap: 1.25rem;
  }

  .banner {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 1.2rem 1.3rem;
    border-radius: var(--radius-xl);
    background: var(--palette-neutral-900);
    color: var(--palette-neutral-50);
  }

  .eyebrow {
    font-size: 0.72rem;
    font-weight: 700;
    letter-spacing: 0.18em;
    color: var(--color-text-muted);
  }

  h1 {
    margin: 0.35rem 0 0;
    font-size: clamp(1.5rem, 4vw, 2.4rem);
  }

  .links {
    display: flex;
    flex-wrap: wrap;
    gap: 0.65rem;
  }

  .links a {
    display: inline-flex;
    align-items: center;
    padding: 0.55rem 0.85rem;
    border-radius: var(--radius-full);
    background: var(--color-surface-hover);
    color: inherit;
    font-weight: 600;
  }

  .panel {
    padding: clamp(1.1rem, 3vw, 1.6rem);
    border-radius: var(--radius-xl);
    background: var(--color-surface);
    border: 1px solid var(--color-border-subtle);
  }

  .loading-shell {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 200px;
    color: var(--color-text-muted);
    font-size: 0.875rem;
  }
</style>

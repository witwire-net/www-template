<script lang="ts">
  import type { Snippet } from 'svelte';

  let { children }: { children: Snippet } = $props();
</script>

<section class="app-shell">
  <div class="banner">
    <div>
      <div class="eyebrow">CSR APP SHELL</div>
      <h1>/app/* は browser 実行に固定</h1>
    </div>
    <div class="links">
      <a href="/app">Overview</a>
      <a href="/app/profiles">Profiles</a>
      <a href="/timeline">Public timeline</a>
    </div>
  </div>

  <div class="panel">
    {@render children()}
  </div>
</section>

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
    border-radius: 1.5rem;
    background: linear-gradient(135deg, rgba(17, 24, 39, 0.94), rgba(30, 64, 175, 0.88));
    color: white;
  }

  .eyebrow {
    font-size: 0.72rem;
    font-weight: 700;
    letter-spacing: 0.18em;
    color: rgba(191, 219, 254, 0.92);
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
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.12);
    color: inherit;
    font-weight: 600;
  }

  .panel {
    padding: clamp(1.1rem, 3vw, 1.6rem);
    border-radius: 1.5rem;
    background: linear-gradient(180deg, rgba(255, 255, 255, 0.92), rgba(239, 246, 255, 0.9));
    border: 1px solid rgba(59, 130, 246, 0.12);
    box-shadow: 0 18px 50px rgba(15, 23, 42, 0.08);
  }
</style>

<script lang="ts">
  import { env } from '$env/dynamic/public';

  const highlights = [
    'public route は SSR の SvelteKit として運用',
    '認証 UI は別ドメインの CSR アプリとして分離',
  ];

  const appUrl = env.PUBLIC_APP_URL ?? 'http://localhost:5174';
</script>

<section class="hero-section">
  <div class="hero-section__copy">
    <div class="hero-section__eyebrow">SvelteKit + Cloudflare Workers</div>
    <h1>公開面と認証面を再利用しやすい構成でまとめています。</h1>
    <p>
      <code>packages/web</code> は公開ルートの SSR を担い、<code>packages/frontend/app</code> は
      別ドメインの SvelteKit SPA を担います。
    </p>
    <ul>
      {#each highlights as item (item)}
        <li>{item}</li>
      {/each}
    </ul>
    <a href="{appUrl}/login" class="hero-section__cta">ログインを試す</a>
  </div>
</section>

<script lang="ts">
  import { env } from '$env/dynamic/public';

  import { useI18n, type Locale } from '$lib/i18n';

  interface Props {
    data: {
      locale: Locale;
    };
  }

  let { data }: Props = $props();

  const appUrl = env.PUBLIC_APP_URL ?? 'http://app.localhost:5174';
  const locale = $derived(data.locale);
  const i18n = $derived(useI18n(locale));

  /**
   * ルーティングされた locale に合わせて html lang を更新する。
   * SvelteKit の app.html では動的な lang 属性を持てないため、
   * クライアント側で document.documentElement.lang を設定する。
   */
  $effect(() => {
    if (typeof document !== 'undefined') {
      document.documentElement.lang = locale;
    }
  });

  const highlights = $derived([i18n.t('common.highlight1'), i18n.t('common.highlight2')]);
</script>

<svelte:head>
  <title>{i18n.t('common.heroTitle')}</title>
  <meta name="description" content={i18n.t('common.heroDescription')} />
  <link rel="canonical" href={`/${data.locale}`} />
</svelte:head>

<section class="hero-section">
  <div class="hero-section__copy">
    <div class="hero-section__eyebrow">{i18n.t('common.heroEyebrow')}</div>
    <h1>{i18n.t('common.heroTitle')}</h1>
    <p>{i18n.t('common.heroDescription')}</p>
    <ul>
      {#each highlights as item (item)}
        <li>{item}</li>
      {/each}
    </ul>
    <a href={`${appUrl}/login`} class="hero-section__cta">{i18n.t('common.loginCta')}</a>
  </div>
</section>

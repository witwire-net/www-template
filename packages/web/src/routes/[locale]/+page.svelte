<script lang="ts">
  import { env } from '$env/dynamic/public';
  import { MonoLabel } from '@www-template/ui';

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
</script>

<svelte:head>
  <title>{i18n.t('common.heroTitle')}</title>
  <meta name="description" content={i18n.t('common.heroLead')} />
  <link rel="canonical" href={`/${data.locale}`} />
</svelte:head>

<section class="hero-section">
  <div class="hero-section__grid">
    <div class="hero-section__copy">
      <h1 class="hero-section__title">
        {i18n.t('common.heroTitle')}
      </h1>
      <p class="hero-section__lead">{i18n.t('common.heroLead')}</p>
      <div class="hero-section__actions">
        <a href={`${appUrl}/login`} class="hero-section__cta">
          {i18n.t('common.heroCtaPrimary')}
        </a>
        <a
          href="https://github.com/"
          target="_blank"
          rel="noopener noreferrer"
          class="hero-section__secondary-link"
        >
          {i18n.t('common.heroCtaSecondary')}
        </a>
      </div>
    </div>

    <aside class="hero-section__meta" aria-label={i18n.t('common.heroMetaAriaLabel')}>
      <div class="hero-section__meta-row">
        <MonoLabel tone="muted">{i18n.t('common.metaStackLabel')}</MonoLabel>
        <span class="hero-section__meta-value">{i18n.t('common.metaStackValue')}</span>
      </div>
      <div class="hero-section__meta-row">
        <MonoLabel tone="muted">{i18n.t('common.metaRuntimeLabel')}</MonoLabel>
        <span class="hero-section__meta-value">{i18n.t('common.metaRuntimeValue')}</span>
      </div>
      <div class="hero-section__meta-row">
        <MonoLabel tone="muted">{i18n.t('common.metaArchitectureLabel')}</MonoLabel>
        <span class="hero-section__meta-value">{i18n.t('common.metaArchitectureValue')}</span>
      </div>
    </aside>
  </div>
</section>

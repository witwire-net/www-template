<script lang="ts">
  import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
  import { BrandMark, PageHeader } from '@www-template/ui';
  import '@www-template/ui/styles.css';
  import '../app.css';
  import { useObservability } from '$lib/observability.svelte';
  import { SUPPORTED_LOCALES, useI18n, type Locale } from '$lib/i18n';
  import type { Snippet } from 'svelte';

  type NavLink = {
    href: string;
    label: string;
  };

  let { children, data }: { children: Snippet; data: { locale: Locale } } = $props();

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

  const currentLocale = $derived(data.locale);
  const i18n = $derived(useI18n(currentLocale));

  let links = $derived<NavLink[]>([
    { href: `/${currentLocale}`, label: i18n.t('common.home') },
  ]);
</script>

<QueryClientProvider client={queryClient}>
  <div class="web-layout">
    <PageHeader>
      <BrandMark size="md" />
      <nav class="web-layout__nav" aria-label={i18n.t('common.navAriaLabel')}>
        {#each links as link (link.href)}
          <a
            class="web-layout__nav-link"
            href={link.href}
            data-active={link.href === `/${currentLocale}`}
          >
            {link.label}
          </a>
        {/each}
      </nav>
      {#snippet trailing()}
        <nav class="web-layout__nav" aria-label={i18n.t('common.languageSwitchAriaLabel')}>
          {#each SUPPORTED_LOCALES as locale (locale)}
            {#if locale === currentLocale}
              <span class="web-layout__nav-link" aria-current="true">{locale.toUpperCase()}</span>
            {:else}
              <a class="web-layout__nav-link" href={`/${locale}`}>{locale.toUpperCase()}</a>
            {/if}
          {/each}
        </nav>
      {/snippet}
    </PageHeader>

    <main class="web-layout__content">
      {@render children()}
    </main>
  </div>
</QueryClientProvider>

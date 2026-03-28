<svelte:options runes={true} />

<script lang="ts">
  import Container from '@ui/components/atoms/Grid/Container.svelte';

  import { getTextContent, isSnippet, joinClassNames, type Renderable } from '@ui/components/navigation/shared';

  import styles from './Header.module.scss';

  type HeaderLink = {
    active?: boolean;
    href: string;
    label: string;
  };

  type Props = {
    actions?: Renderable;
    className?: string;
    closeButtonLabel?: string;
    isMinimized?: boolean;
    links?: HeaderLink[];
    logo?: Renderable;
    menuButtonLabel?: string;
    menuIcon?: Renderable;
    mobileBackdropLabel?: string;
    mobileMenuHeading?: Renderable;
    mobileNavigationAriaLabel?: string;
    navigationAriaLabel?: string;
    /** 'site': public site with mobile drawer menu. 'app': dashboard header with left-side menu button triggering onMenuClick. */
    onMenuClick?: () => void;
    variant?: 'app' | 'site';
  };

  let {
    actions = undefined,
    className = undefined,
    closeButtonLabel = 'Close',
    isMinimized = false,
    links = [],
    logo = undefined,
    menuButtonLabel = 'Open mobile menu',
    menuIcon = undefined,
    mobileBackdropLabel = 'Close mobile menu',
    mobileMenuHeading = undefined,
    mobileNavigationAriaLabel = undefined,
    navigationAriaLabel = undefined,
    onMenuClick = undefined,
    variant = 'site',
  }: Props = $props();

  let isMobileMenuOpen = $state(false);

  const isSite = $derived(variant === 'site');
  const isApp = $derived(variant === 'app');

  const rootClassName = $derived(
    joinClassNames(
      styles.header ?? '',
      isMinimized ? (styles.minimized ?? '') : undefined,
      isSite ? (styles.site ?? '') : (styles.app ?? ''),
      className
    )
  );
  const openClassName = $derived(isMobileMenuOpen ? (styles.open ?? '') : undefined);
  const hasActions = $derived(actions !== undefined && actions !== null);
  const resolvedMobileMenuHeading = $derived(mobileMenuHeading ?? logo);
  const resolvedMobileNavigationAriaLabel = $derived(mobileNavigationAriaLabel ?? navigationAriaLabel);

  function handleOpen(): void {
    isMobileMenuOpen = true;
  }

  function handleClose(): void {
    isMobileMenuOpen = false;
  }

  function handleBackdropKeydown(event: KeyboardEvent): void {
    if (event.key === 'Enter' || event.key === ' ' || event.key === 'Escape') {
      event.preventDefault();
      handleClose();
    }
  }

  $effect(() => {
    if (typeof document === 'undefined' || !isMobileMenuOpen) {
      return;
    }
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = previousOverflow;
    };
  });
</script>

<header class={rootClassName}>
  <Container className={styles.inner ?? ''}>
    <!-- Left section: app menu button (left) OR logo -->
    <div class={styles.leftSection ?? ''}>
      {#if isApp && onMenuClick !== undefined}
        <button
          type="button"
          class={styles.menuButton ?? ''}
          onclick={onMenuClick}
          aria-label={menuButtonLabel}
        >
          {#if menuIcon !== undefined && menuIcon !== null}
            {#if isSnippet(menuIcon)}
              {@render menuIcon()}
            {:else}
              {getTextContent(menuIcon)}
            {/if}
          {:else}
            <span class={styles.menuIcon ?? ''} aria-hidden="true">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="3" y1="6" x2="21" y2="6" />
                <line x1="3" y1="12" x2="21" y2="12" />
                <line x1="3" y1="18" x2="21" y2="18" />
              </svg>
            </span>
          {/if}
        </button>
      {/if}

      {#if logo !== undefined && logo !== null}
        <div class={styles.logo ?? ''}>
          {#if isSnippet(logo)}
            {@render logo()}
          {:else}
            {getTextContent(logo)}
          {/if}
        </div>
      {/if}
    </div>

    <!-- Center nav -->
    <nav class={styles.nav ?? ''} aria-label={navigationAriaLabel}>
      {#each links as link (link.label)}
        <a
          href={link.href}
          class={joinClassNames(styles.navItem ?? '', link.active === true ? (styles.active ?? '') : undefined)}
        >
          {link.label}
        </a>
      {/each}
    </nav>

    <!-- Right section: actions + site mobile menu button -->
    <div class={styles.rightSection ?? ''}>
      <div class={styles.actions ?? ''}>
        {#if actions !== undefined && actions !== null}
          {#if isSnippet(actions)}
            {@render actions()}
          {:else}
            {getTextContent(actions)}
          {/if}
        {/if}
      </div>
      {#if isSite}
        <button
          type="button"
          class={styles.menuButton ?? ''}
          onclick={handleOpen}
          aria-label={menuButtonLabel}
          aria-expanded={isMobileMenuOpen}
          aria-controls="site-mobile-drawer"
        >
          {#if menuIcon !== undefined && menuIcon !== null}
            {#if isSnippet(menuIcon)}
              {@render menuIcon()}
            {:else}
              {getTextContent(menuIcon)}
            {/if}
          {:else}
            <span class={styles.menuIcon ?? ''} aria-hidden="true">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="3" y1="6" x2="21" y2="6" />
                <line x1="3" y1="12" x2="21" y2="12" />
                <line x1="3" y1="18" x2="21" y2="18" />
              </svg>
            </span>
          {/if}
        </button>
      {/if}
    </div>
  </Container>
</header>

<!-- Site variant: mobile drawer -->
{#if isSite}
  {#if isMobileMenuOpen}
    <div
      class={joinClassNames(styles.mobileBackdrop ?? '', openClassName)}
      role="button"
      tabindex="0"
      aria-label={mobileBackdropLabel}
      onclick={handleClose}
      onkeydown={handleBackdropKeydown}
    ></div>
  {/if}

  <div id="site-mobile-drawer" class={joinClassNames(styles.mobileDrawer ?? '', openClassName)} aria-hidden={!isMobileMenuOpen}>
    <div class={styles.overlayHeader ?? ''}>
      {#if resolvedMobileMenuHeading !== undefined && resolvedMobileMenuHeading !== null}
        {#if isSnippet(resolvedMobileMenuHeading)}
          {@render resolvedMobileMenuHeading()}
        {:else}
          {getTextContent(resolvedMobileMenuHeading)}
        {/if}
      {/if}
      <button
        class={styles.closeButton ?? ''}
        type="button"
        onclick={handleClose}
        aria-label={closeButtonLabel}
      >
        <span class={styles.closeIcon ?? ''} aria-hidden="true">×</span>
      </button>
    </div>
    <div class={styles.overlayContent ?? ''}>
      <nav class={styles.mobileNav ?? ''} aria-label={resolvedMobileNavigationAriaLabel}>
        {#each links as link (link.label)}
          <a
            href={link.href}
            class={joinClassNames(
              styles.mobileNavItem ?? '',
              link.active === true ? (styles.mobileActive ?? '') : undefined
            )}
            onclick={handleClose}
          >
            {link.label}
          </a>
        {/each}
      </nav>
      {#if hasActions}
        <div class={styles.mobileActions ?? ''}>
          {#if isSnippet(actions)}
            {@render actions()}
          {:else}
            {getTextContent(actions)}
          {/if}
        </div>
      {/if}
    </div>
  </div>
{/if}

<svelte:options runes={true} />

<script lang="ts">
  import Header from './Header.svelte';
  import MenuButton from './MenuButton.svelte';

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
  };

  let {
    logo = undefined,
    links = [],
    actions = undefined,
    menuIcon = undefined,
    className = undefined,
    closeButtonLabel = 'Close',
    isMinimized = false,
    menuButtonLabel = 'Open mobile menu',
    mobileBackdropLabel = 'Close mobile menu',
    mobileMenuHeading = undefined,
    mobileNavigationAriaLabel = undefined,
    navigationAriaLabel = undefined,
  }: Props = $props();

  let isMobileMenuOpen = $state(false);

  const mergedClassName = $derived(joinClassNames(styles.website ?? '', className));
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

<Header {logo} {links} {actions} className={mergedClassName} {isMinimized} navigationAriaLabel={navigationAriaLabel}>
  {#snippet rightSlot()}
    <MenuButton onClick={handleOpen} icon={menuIcon} ariaLabel={menuButtonLabel} />
  {/snippet}
</Header>

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

<div class={joinClassNames(styles.mobileDrawer ?? '', openClassName)}>
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

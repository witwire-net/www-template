<svelte:options runes={true} />

<script lang="ts">
  import { IconX } from '@tabler/icons-svelte';

  import Icon from '@ui/components/atoms/Icon/Icon.svelte';

  import { getTextContent, isSnippet, joinClassNames, type Renderable } from '@ui/components/navigation/shared';

  import styles from './AppSidebar.module.scss';

  type AppSidebarLink = {
    active?: boolean;
    href: string;
    icon?: Renderable;
    label: string;
  };

  type Props = {
    className?: string;
    closeIcon?: Renderable;
    footer?: Renderable;
    isOpen?: boolean;
    links?: AppSidebarLink[];
    logo?: Renderable;
    onClose?: () => void;
  };

  let {
    logo = undefined,
    links = [],
    footer = undefined,
    className = undefined,
    isOpen = false,
    onClose = undefined,
    closeIcon = undefined,
  }: Props = $props();

  const canClose = $derived(onClose !== undefined);
  const sidebarClassName = $derived(
    joinClassNames(styles.sidebar ?? '', isOpen ? (styles.open ?? '') : undefined, className)
  );
  let isMobileViewport = $state(false);

  const overlayClassName = $derived(
    joinClassNames(styles.overlay ?? '', isOpen ? (styles.open ?? '') : undefined)
  );
  const isMobileSidebarHidden = $derived(isMobileViewport && !isOpen);
  const showMobileOverlay = $derived(isMobileViewport && isOpen);

  function handleClose(): void {
    onClose?.();
  }

  function handleBackdropKeydown(event: KeyboardEvent): void {
    if (onClose === undefined) {
      return;
    }

    if (event.key === 'Enter' || event.key === ' ' || event.key === 'Escape') {
      event.preventDefault();
      handleClose();
    }
  }

  $effect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const mediaQuery = window.matchMedia('(max-width: 768px)');

    const updateViewport = (): void => {
      isMobileViewport = mediaQuery.matches;
    };

    updateViewport();
    mediaQuery.addEventListener('change', updateViewport);

    return () => {
      mediaQuery.removeEventListener('change', updateViewport);
    };
  });

  $effect(() => {
    if (typeof document === 'undefined' || !isOpen) {
      return;
    }

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    return () => {
      document.body.style.overflow = previousOverflow;
    };
  });
</script>

{#if showMobileOverlay && canClose}
  <button
    type="button"
    class={overlayClassName}
    aria-label="Close sidebar"
    onclick={handleClose}
    onkeydown={handleBackdropKeydown}
  ></button>
{/if}

<aside
  class={sidebarClassName}
  inert={isMobileSidebarHidden ? true : undefined}
  aria-hidden={isMobileSidebarHidden ? 'true' : undefined}
>
  <div class={styles.logo ?? ''}>
    {#if logo !== undefined && logo !== null}
      {#if isSnippet(logo)}
        {@render logo()}
      {:else}
        {getTextContent(logo)}
      {/if}
    {/if}
    {#if canClose}
      <button
        type="button"
        class={styles.closeParams ?? ''}
        onclick={handleClose}
        aria-label="Close sidebar"
      >
        {#if closeIcon !== undefined && closeIcon !== null}
          {#if isSnippet(closeIcon)}
            {@render closeIcon()}
          {:else}
            {getTextContent(closeIcon)}
          {/if}
        {:else}
          <Icon icon={IconX} className={styles.closeIcon ?? ''} title="Close sidebar" />
        {/if}
      </button>
    {/if}
  </div>

  <nav class={styles.nav ?? ''}>
    {#each links as link (link.label)}
      <a
        href={link.href}
        class={joinClassNames(styles.navItem ?? '', link.active === true ? (styles.active ?? '') : undefined)}
      >
        {#if link.icon !== undefined && link.icon !== null}
          <span class={styles.navIcon ?? ''}>
            {#if isSnippet(link.icon)}
              {@render link.icon()}
            {:else}
              {getTextContent(link.icon)}
            {/if}
          </span>
        {/if}
        {link.label}
      </a>
    {/each}
  </nav>

  {#if footer !== undefined && footer !== null}
    <div class={styles.footer ?? ''}>
      {#if isSnippet(footer)}
        {@render footer()}
      {:else}
        {getTextContent(footer)}
      {/if}
    </div>
  {/if}
</aside>

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
    isMinimized?: boolean;
    leftSlot?: Renderable;
    links?: HeaderLink[];
    navigationAriaLabel?: string;
    logo?: Renderable;
    rightSlot?: Renderable;
  };

  let {
    logo = undefined,
    links = [],
    actions = undefined,
    leftSlot = undefined,
    rightSlot = undefined,
    className = undefined,
    isMinimized = false,
    navigationAriaLabel = undefined,
  }: Props = $props();

  const rootClassName = $derived(
    joinClassNames(styles.header ?? '', isMinimized ? (styles.minimized ?? '') : undefined, className)
  );
</script>

<header class={rootClassName}>
  <Container className={styles.inner ?? ''}>
    <div class={styles.leftSection ?? ''}>
      {#if leftSlot !== undefined && leftSlot !== null}
        {#if isSnippet(leftSlot)}
          {@render leftSlot()}
        {:else}
          {getTextContent(leftSlot)}
        {/if}
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
      {#if rightSlot !== undefined && rightSlot !== null}
        {#if isSnippet(rightSlot)}
          {@render rightSlot()}
        {:else}
          {getTextContent(rightSlot)}
        {/if}
      {/if}
    </div>
  </Container>
</header>

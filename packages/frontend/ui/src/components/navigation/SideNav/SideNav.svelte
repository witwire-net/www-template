<svelte:options runes={true} />

<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements';

  import { getTextContent, isSnippet, joinClassNames, type Renderable } from '@ui/components/navigation/shared';

  import styles from './SideNav.module.scss';

  type SideNavItem = {
    active?: boolean;
    href?: string;
    icon?: Renderable;
    label: string;
    onClick?: () => void;
  };

  type Props = HTMLAttributes<HTMLElement> & {
    className?: string;
    footer?: Renderable;
    header?: Renderable;
    items: SideNavItem[];
  };

  let {
    header = undefined,
    footer = undefined,
    items,
    className = undefined,
    ...restProps
  }: Props = $props();

  const rootClassName = $derived(joinClassNames(styles.nav ?? '', className));

  function getItemClassName(item: SideNavItem): string {
    return joinClassNames(styles.item ?? '', item.active === true ? (styles.active ?? '') : undefined);
  }
</script>

<nav class={rootClassName} {...restProps}>
  {#if header !== undefined && header !== null}
    <div class={styles.header ?? ''}>
      {#if isSnippet(header)}
        {@render header()}
      {:else}
        {getTextContent(header)}
      {/if}
    </div>
  {/if}
  <div class={styles.list ?? ''}>
    {#each items as item (item.label)}
      {#if item.href !== undefined && item.href !== ''}
        <a href={item.href} onclick={item.onClick} class={getItemClassName(item)}>
          {#if item.icon !== undefined && item.icon !== null}
            <span class={styles.icon ?? ''}>
              {#if isSnippet(item.icon)}
                {@render item.icon()}
              {:else}
                {getTextContent(item.icon)}
              {/if}
            </span>
          {/if}
          <span>{item.label}</span>
        </a>
      {:else if item.onClick !== undefined}
        <button
          type="button"
          onclick={item.onClick}
          class={joinClassNames(getItemClassName(item), styles.itemButton ?? '')}
        >
          {#if item.icon !== undefined && item.icon !== null}
            <span class={styles.icon ?? ''}>
              {#if isSnippet(item.icon)}
                {@render item.icon()}
              {:else}
                {getTextContent(item.icon)}
              {/if}
            </span>
          {/if}
          <span>{item.label}</span>
        </button>
      {:else}
        <span class={joinClassNames(getItemClassName(item), styles.itemStatic ?? '')}>
          {#if item.icon !== undefined && item.icon !== null}
            <span class={styles.icon ?? ''}>
              {#if isSnippet(item.icon)}
                {@render item.icon()}
              {:else}
                {getTextContent(item.icon)}
              {/if}
            </span>
          {/if}
          <span>{item.label}</span>
        </span>
      {/if}
    {/each}
  </div>
  {#if footer !== undefined && footer !== null}
    <div class={styles.footer ?? ''}>
      {#if isSnippet(footer)}
        {@render footer()}
      {:else}
        {getTextContent(footer)}
      {/if}
    </div>
  {/if}
</nav>

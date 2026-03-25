<script lang="ts">
  import type { Snippet } from 'svelte';

  import { VirtualList } from '@ui/components/shared';

  import type { VirtualizeOptions } from '@ui/components/shared';

  import styles from './List.module.scss';

  interface ListItem {
    title: string;
    description?: string;
    meta?: string;
    icon?: unknown;
    action?: unknown;
    iconContent?: Snippet;
    actionContent?: Snippet;
  }

  interface ListProps {
    items?: readonly ListItem[];
    className?: string;
    /** When set, enables virtual scrolling for large lists. */
    virtualize?: VirtualizeOptions;
  }

  /** Matches `--spacing-sm` (0.5rem at 16px root). */
  const DEFAULT_GAP = 8;

  let { items = [], className, virtualize }: ListProps = $props();

  const rootClassName = $derived([styles.list ?? '', className ?? ''].filter((value) => value !== '').join(' '));

  const renderFallback = (value: unknown): string => {
    if (typeof value === 'string' || typeof value === 'number') {
      return `${value}`;
    }

    return '';
  };

  const getItemKey = (item: ListItem): string => {
    const safeDescription = item.description ?? '';
    const safeMeta = item.meta ?? '';

    return `${item.title}-${safeDescription}-${safeMeta}`;
  };

  const getItemKeyByIndex = (index: number): string => {
    return getItemKey(getItemByIndex(index));
  };

  const getItemByIndex = (index: number): ListItem => {
    const item = items[index];

    if (item === undefined) {
      throw new RangeError('List item index out of bounds');
    }

    return item;
  };
</script>

{#snippet itemRow(item: ListItem)}
  <div class={styles.item ?? ''}>
    {#if item.iconContent !== undefined || item.icon !== undefined}
      <div class={styles.icon ?? ''}>
        {#if item.iconContent !== undefined}
          {@render item.iconContent()}
        {:else}
          {renderFallback(item.icon)}
        {/if}
      </div>
    {/if}

    <div class={styles.content ?? ''}>
      <div class={styles.title ?? ''}>{item.title}</div>
      {#if item.description !== undefined && item.description !== ''}
        <div class={styles.description ?? ''}>{item.description}</div>
      {/if}
    </div>

    {#if item.meta !== undefined && item.meta !== ''}
      <div class={styles.meta ?? ''}>{item.meta}</div>
    {/if}

    {#if item.actionContent !== undefined || item.action !== undefined}
      <div class={styles.action ?? ''}>
        {#if item.actionContent !== undefined}
          {@render item.actionContent()}
        {:else}
          {renderFallback(item.action)}
        {/if}
      </div>
    {/if}
  </div>
{/snippet}

{#if virtualize !== undefined}
  <VirtualList
    count={items.length}
    estimateSize={virtualize.estimateSize}
    overscan={virtualize.overscan}
    height={virtualize.height}
    gap={virtualize.gap ?? DEFAULT_GAP}
    getItemKey={virtualize.getItemKey ?? getItemKeyByIndex}
    className={rootClassName}
    ariaLabel="List"
    row={(index) => itemRow(getItemByIndex(index))}
  />
{:else}
  <div class={rootClassName}>
    {#each items as item (getItemKey(item))}
      {@render itemRow(item)}
    {/each}
  </div>
{/if}

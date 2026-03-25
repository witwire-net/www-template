<svelte:options runes={true} />

<script lang="ts">
  import RenderableContent from '@ui/components/app/RenderableContent.svelte';
  import { VirtualList } from '@ui/components/shared';

  import type { Renderable } from '@ui/components/app/shared';
  import type { VirtualizeOptions } from '@ui/components/shared';

  import styles from './ActivityFeed.module.scss';

  type ActivityItem = {
    avatar?: Renderable;
    description?: string;
    time?: string;
    title: string;
  };

  type Props = {
    items: ActivityItem[];
    /** When set, enables virtual scrolling for large lists. */
    virtualize?: VirtualizeOptions;
  };

  /** Matches `--spacing-sm` (0.5rem at 16px root). */
  const DEFAULT_GAP = 8;

  let { items, virtualize }: Props = $props();

  function getItemKey(item: ActivityItem): string {
    const details = item.description ?? '';

    return `${item.title}-${details}`;
  }

  function getItemKeyByIndex(index: number): string {
    return getItemKey(getItemByIndex(index));
  }

  function getItemByIndex(index: number): ActivityItem {
    const item = items[index];

    if (item === undefined) {
      throw new RangeError('Activity item index out of bounds');
    }

    return item;
  }
</script>

{#snippet itemRow(item: ActivityItem)}
  <div class={styles.item ?? ''}>
    {#if item.avatar !== undefined && item.avatar !== null}
      <div class={styles.avatar ?? ''}>
        <RenderableContent value={item.avatar} />
      </div>
    {/if}
    <div class={styles.content ?? ''}>
      <div class={styles.title ?? ''}>{item.title}</div>
      {#if item.description !== undefined && item.description !== ''}
        <div class={styles.description ?? ''}>{item.description}</div>
      {/if}
    </div>
    {#if item.time !== undefined && item.time !== ''}
      <div class={styles.time ?? ''}>{item.time}</div>
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
    className={styles.feed ?? ''}
    ariaLabel="Activity feed"
    row={(index) => itemRow(getItemByIndex(index))}
  />
{:else}
  <div class={styles.feed ?? ''}>
    {#each items as item (getItemKey(item))}
      {@render itemRow(item)}
    {/each}
  </div>
{/if}

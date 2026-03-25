<svelte:options runes={true} />

<script lang="ts">
  import { VirtualList } from '@ui/components/shared';
  import { joinClassNames } from '@ui/components/app/shared';

  import type { VirtualizeOptions } from '@ui/components/shared';

  import styles from './NotificationCenter.module.scss';

  type NotificationItem = {
    description?: string;
    read?: boolean;
    time?: string;
    title: string;
  };

  type Props = {
    items: NotificationItem[];
    /** When set, enables virtual scrolling for large lists. */
    virtualize?: VirtualizeOptions;
  };

  /** Matches `--spacing-sm` (0.5rem at 16px root). */
  const DEFAULT_GAP = 8;

  let { items, virtualize }: Props = $props();

  function getNotificationKey(item: NotificationItem): string {
    const description = item.description ?? '';

    return `${item.title}-${item.time ?? ''}-${description}`;
  }

  function getNotificationKeyByIndex(index: number): string {
    return getNotificationKey(getNotificationByIndex(index));
  }

  function getNotificationByIndex(index: number): NotificationItem {
    const item = items[index];

    if (item === undefined) {
      throw new RangeError('Notification item index out of bounds');
    }

    return item;
  }
</script>

{#snippet itemRow(item: NotificationItem)}
  <div class={joinClassNames(styles.item ?? '', item.read === true ? (styles.read ?? '') : undefined)}>
    <div class={styles.title ?? ''}>{item.title}</div>
    {#if item.description !== undefined && item.description !== ''}
      <div class={styles.description ?? ''}>{item.description}</div>
    {/if}
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
    getItemKey={virtualize.getItemKey ?? getNotificationKeyByIndex}
    className={styles.center ?? ''}
    ariaLabel="Notification center"
    row={(index) => itemRow(getNotificationByIndex(index))}
  />
{:else}
  <div class={styles.center ?? ''}>
    {#each items as item (getNotificationKey(item))}
      {@render itemRow(item)}
    {/each}
  </div>
{/if}

<svelte:options runes={true} />

<script lang="ts">
  import { createVirtualizer, measureElement } from '@tanstack/svelte-virtual';

  import { DEFAULT_HEIGHT, DEFAULT_OVERSCAN } from '@ui/components/shared/virtual';

  import type { Snippet } from 'svelte';

  type Props = {
    /** Total number of items in the list. */
    count: number;
    /** Estimated height (px) of a single item. Used for initial layout before measurement. */
    estimateSize: number;
    /** Number of extra items to render outside the viewport. @default 5 */
    overscan?: number;
    /** Fixed height (px) for the scrollable container. @default 400 */
    height?: number;
    /** Gap (px) between items, matching the non-virtualized layout. @default 0 */
    gap?: number;
    /** Returns a stable key for the item at the given index. Used by the virtualizer for measurement caching and by Svelte for DOM reconciliation. */
    getItemKey?: (index: number) => number | string | bigint;
    /** Snippet that receives the virtual item index and renders a single row. */
    row: Snippet<[number]>;
    /** Optional CSS class on the outer scroll container. */
    className?: string;
    /** Optional accessible label. */
    ariaLabel?: string;
  };

  let {
    count,
    estimateSize,
    overscan = DEFAULT_OVERSCAN,
    height = DEFAULT_HEIGHT,
    gap = 0,
    getItemKey,
    row,
    className,
    ariaLabel,
  }: Props = $props();

  let scrollElement: HTMLDivElement | undefined = $state(undefined);

  const virtualizer = createVirtualizer({
    get count() {
      return count;
    },
    getScrollElement: () => scrollElement ?? null,
    get estimateSize() {
      return () => estimateSize;
    },
    get overscan() {
      return overscan;
    },
    get gap() {
      return gap;
    },
    get getItemKey() {
      return getItemKey;
    },
    measureElement,
  });

  const virtualItems = $derived($virtualizer.getVirtualItems());

  const totalSize = $derived($virtualizer.getTotalSize());
</script>

<div
  bind:this={scrollElement}
  class={className ?? ''}
  role="list"
  aria-label={ariaLabel}
  style="overflow-y: auto; height: {height}px; contain: strict;"
>
  <div style="height: {totalSize}px; width: 100%; position: relative;">
    {#each virtualItems as item (item.key)}
      <div
        role="listitem"
        data-index={item.index}
        style="position: absolute; top: 0; left: 0; width: 100%; transform: translateY({item.start}px);"
      >
        {@render row(item.index)}
      </div>
    {/each}
  </div>
</div>

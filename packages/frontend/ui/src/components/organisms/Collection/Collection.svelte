<script lang="ts" generics="TItem">
  import type { Snippet } from 'svelte';

  import styles from './Collection.module.scss';

  interface CollectionProps<TValue> {
    items?: readonly TValue[];
    columns?: 1 | 2 | 3 | 4 | 5 | 6;
    className?: string;
    itemClassName?: string;
    getKey: (item: TValue, index: number) => string | number;
    renderItem: Snippet<[TValue, number]>;
  }

  const {
    items = [],
    columns = 2,
    className,
    itemClassName,
    getKey,
    renderItem,
  }: CollectionProps<TItem> = $props();

  const joinClassNames = (...values: (string | undefined)[]): string => {
    return values.filter((value) => value !== undefined && value !== '').join(' ');
  };

  let rootClassName = $derived(joinClassNames(styles.grid, className));
  let itemClass = $derived(joinClassNames(styles.item, itemClassName));
  let rootStyle = $derived(`--_columns: ${String(columns)};`);
</script>

<div class={rootClassName} style={rootStyle}>
  {#each items as item, index (getKey(item, index))}
    <div class={itemClass}>
      {@render renderItem(item, index)}
    </div>
  {/each}
</div>

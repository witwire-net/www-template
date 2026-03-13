<script lang="ts" generics="TData">
  import type { Snippet } from 'svelte';

  import styles from './Table.module.scss';

  interface TableColumn<TItem> {
    header: string;
    accessor: Snippet<[TItem, number]>;
    className?: string;
  }

  interface TableProps<TRow> {
    columns?: readonly TableColumn<TRow>[];
    data?: readonly TRow[];
    className?: string;
    containerClassName?: string;
    getRowKey?: (row: TRow, index: number) => string | number;
  }

  const {
    columns = [],
    data = [],
    className,
    containerClassName,
    getRowKey,
  }: TableProps<TData> = $props();

  const joinClassNames = (...values: (string | undefined)[]): string => {
    return values.filter((value) => value !== undefined && value !== '').join(' ');
  };

  const resolveRowKey = (row: TData, index: number): string | number => {
    return getRowKey?.(row, index) ?? index;
  };

  let tableClass = $derived(joinClassNames(styles.table, className));
  let outerClass = $derived(joinClassNames(styles.container, containerClassName));
</script>

<div class={outerClass}>
  <table class={tableClass}>
    <thead>
      <tr>
        {#each columns as column (column.header)}
          <th class={joinClassNames(styles.headerCell, column.className)}>
            {column.header}
          </th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#each data as row, rowIndex (resolveRowKey(row, rowIndex))}
        {@const rowKey = resolveRowKey(row, rowIndex)}
        <tr>
          {#each columns as column (`${String(rowKey)}-${column.header}`)}
            <td class={joinClassNames(styles.cell, column.className)}>
              {@render column.accessor(row, rowIndex)}
            </td>
          {/each}
        </tr>
      {/each}
    </tbody>
  </table>
</div>

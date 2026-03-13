<script lang="ts" generics="TRow extends { id?: string | number }">
  import type { Snippet } from 'svelte';

  import Table from '@ui/components/organisms/Table/Table.svelte';

  import styles from './DataGrid.module.scss';

  interface DataGridColumn<TData> {
    header: string;
    accessor: Snippet<[TData, number]>;
    className?: string;
  }

  interface DataGridProps<TData> {
    columns?: readonly DataGridColumn<TData>[];
    data?: readonly TData[];
    compact?: boolean;
    className?: string;
    containerClassName?: string;
    getRowKey?: (row: TData, index: number) => string | number;
  }

  let {
    columns = [],
    data = [],
    compact = false,
    className,
    containerClassName,
    getRowKey,
  }: DataGridProps<TRow> = $props();

  const tableClassName = $derived(
    [className ?? '', compact ? (styles.compact ?? '') : ''].filter((value) => value !== '').join(' ')
  );
  const outerClassName = $derived(
    [styles.container ?? '', containerClassName ?? ''].filter((value) => value !== '').join(' ')
  );
</script>

<Table
  {columns}
  {data}
  {getRowKey}
  className={tableClassName}
  containerClassName={outerClassName}
/>

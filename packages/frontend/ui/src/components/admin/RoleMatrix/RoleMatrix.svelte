<script lang="ts">
  import type { Snippet } from 'svelte';

  import Badge from '@ui/components/atoms/Badge/Badge.svelte';

  import styles from './RoleMatrix.module.scss';

  type BadgeVariant = 'primary' | 'neutral' | 'success' | 'warning' | 'error' | 'info';
  type Renderable = Snippet | string | number | null | undefined;

  interface RoleMatrixColumn {
    id?: string | number;
    label: string;
  }

  interface RoleMatrixCell {
    content?: Renderable;
    label?: string;
    value?: boolean | string | number | null;
    variant?: BadgeVariant;
  }

  interface RoleMatrixRow {
    id?: string | number;
    label: string;
    cells: readonly RoleMatrixCell[];
  }

  interface Props {
    columns?: readonly RoleMatrixColumn[];
    rows?: readonly RoleMatrixRow[];
    rowHeaderLabel?: string;
    trueLabel?: string;
    falseLabel?: string;
    renderCell?: Snippet<[RoleMatrixCell, RoleMatrixRow, RoleMatrixColumn, number, number]>;
  }

  let {
    columns = [],
    rows = [],
    rowHeaderLabel = 'Item',
    trueLabel = 'Yes',
    falseLabel = 'No',
    renderCell = undefined,
  }: Props = $props();

  const isSnippet = (value: Renderable): value is Snippet => {
    return typeof value === 'function';
  };

  const hasRenderable = (value: Renderable): boolean => {
    return value !== undefined && value !== null;
  };

  const getTextContent = (value: Renderable): string => {
    if (typeof value === 'string') {
      return value;
    }

    if (typeof value === 'number') {
      return String(value);
    }

    return '';
  };

  const getRowKey = (row: RoleMatrixRow, index: number): string | number => {
    return row.id ?? row.label ?? index;
  };

  const getColumnKey = (column: RoleMatrixColumn, index: number): string | number => {
    return column.id ?? column.label ?? index;
  };

  const getCellLabel = (cell: RoleMatrixCell): string => {
    if (typeof cell.label === 'string' && cell.label !== '') {
      return cell.label;
    }

    if (typeof cell.value === 'boolean') {
      return cell.value ? trueLabel : falseLabel;
    }

    if (typeof cell.value === 'number') {
      return String(cell.value);
    }

    if (typeof cell.value === 'string') {
      return cell.value;
    }

    return '';
  };

  const getCellVariant = (cell: RoleMatrixCell): BadgeVariant | undefined => {
    if (cell.variant !== undefined) {
      return cell.variant;
    }

    if (typeof cell.value === 'boolean') {
      return cell.value ? 'success' : 'neutral';
    }

    return undefined;
  };
</script>

<div class={styles.wrapper ?? ''}>
  <table class={styles.table ?? ''}>
    <thead>
      <tr>
        <th>{rowHeaderLabel}</th>
        {#each columns as column, columnIndex (String(getColumnKey(column, columnIndex)))}
          <th>{column.label}</th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#each rows as row, rowIndex (String(getRowKey(row, rowIndex)))}
        {@const rowKey = getRowKey(row, rowIndex)}
        <tr>
          <td class={styles.permission ?? ''}>{row.label}</td>
          {#each columns as column, columnIndex (`${String(rowKey)}-${String(getColumnKey(column, columnIndex))}`)}
            {@const cell = row.cells[columnIndex]}
            <td>
              <div class={styles.cell ?? ''}>
                {#if cell !== undefined}
                  {#if renderCell !== undefined}
                    {@render renderCell(cell, row, column, rowIndex, columnIndex)}
                  {:else if hasRenderable(cell.content)}
                    {#if isSnippet(cell.content)}
                      {@render cell.content()}
                    {:else}
                      {getTextContent(cell.content)}
                    {/if}
                  {:else}
                    {@const cellVariant = getCellVariant(cell)}
                    {@const cellLabel = getCellLabel(cell)}
                    {#if cellVariant !== undefined}
                      <Badge variant={cellVariant} size="sm">{cellLabel}</Badge>
                    {:else}
                      {cellLabel}
                    {/if}
                  {/if}
                {/if}
              </div>
            </td>
          {/each}
        </tr>
      {/each}
    </tbody>
  </table>
</div>

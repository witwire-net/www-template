<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './MatrixTable.module.scss';

  type MatrixCellValue = string | number | boolean | null | undefined;

  interface MatrixTableRow {
    label: MatrixCellValue;
    values: readonly MatrixCellValue[];
    key?: string | number;
  }

  interface MatrixTableProps extends HTMLAttributes<HTMLDivElement> {
    headers?: readonly MatrixCellValue[];
    rows?: readonly MatrixTableRow[];
    firstColumnHeader?: MatrixCellValue;
    className?: string;
    tableClassName?: string;
    headCellClassName?: string;
    labelCellClassName?: string;
    valueCellClassName?: string;
  }

  let {
    headers = [],
    rows = [],
    firstColumnHeader = 'Item',
    className,
    tableClassName,
    headCellClassName,
    labelCellClassName,
    valueCellClassName,
    ...restProps
  }: MatrixTableProps = $props();

  const joinClassNames = (...classNames: (string | undefined)[]): string => {
    return classNames.filter((value) => value !== undefined && value !== '').join(' ');
  };

  const renderCellValue = (value: MatrixCellValue): string => {
    if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
      return String(value);
    }

    return '';
  };

  const rootClassName = $derived(joinClassNames(styles.wrapper, className));
  const tableRootClassName = $derived(joinClassNames(styles.table, tableClassName));
</script>

<div class={rootClassName} {...restProps}>
  <table class={tableRootClassName}>
    <thead>
      <tr>
        <th class={joinClassNames(styles.headCell, headCellClassName)}>
          {renderCellValue(firstColumnHeader)}
        </th>
        {#each headers as header, headerIndex (`header-${String(headerIndex)}`)}
          <th class={joinClassNames(styles.headCell, headCellClassName)}>{renderCellValue(header)}</th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#each rows as row, rowIndex (row.key ?? `${renderCellValue(row.label)}-${String(rowIndex)}`)}
        <tr>
          <td class={joinClassNames(styles.labelCell, labelCellClassName)}>
            {renderCellValue(row.label)}
          </td>
          {#each row.values as value, valueIndex (`cell-${String(rowIndex)}-${String(valueIndex)}`)}
            <td class={joinClassNames(styles.valueCell, valueCellClassName)}>{renderCellValue(value)}</td>
          {/each}
        </tr>
      {/each}
    </tbody>
  </table>
</div>

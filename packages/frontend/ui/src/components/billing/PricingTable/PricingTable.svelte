<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './PricingTable.module.scss';

  interface PricingColumn {
    id: string;
    title: string;
    subtitle?: string;
    highlight?: string;
  }

  interface PricingCell {
    value: string;
    supportingText?: string;
    emphasis?: boolean;
  }

  interface PricingRow {
    id: string;
    label: string;
    description?: string;
    values: readonly (PricingCell | string)[];
  }

  interface LegacyPricingPlan {
    name: string;
    price: string;
    interval?: string;
  }

  interface LegacyPricingFeature {
    label: string;
    values: readonly string[];
  }

  interface PricingTableProps extends HTMLAttributes<HTMLDivElement> {
    columns?: readonly PricingColumn[];
    rows?: readonly PricingRow[];
    heading?: string;
    plans?: readonly LegacyPricingPlan[];
    features?: readonly LegacyPricingFeature[];
    className?: string;
  }

  let {
    columns,
    rows,
    heading = 'Comparison point',
    plans,
    features,
    className,
    ...restProps
  }: PricingTableProps = $props();

  const rootClassName = $derived.by((): string => {
    const values = [styles.tableWrapper ?? '', className ?? ''];

    return values.filter((value): value is string => typeof value === 'string' && value !== '').join(' ');
  });

  const hasText = (value: string | undefined): boolean => {
    return value !== undefined && value !== '';
  };

  const toColumnId = (title: string, index: number): string => {
    return `${title}-${index}`;
  };

  const resolvedColumns = $derived(
    columns ??
      plans?.map((plan, index) => ({
        id: toColumnId(plan.name, index),
        title: plan.name,
        highlight: plan.price,
        subtitle: hasText(plan.interval) ? `per ${plan.interval}` : undefined,
      })) ??
      []
  );

  const resolvedRows = $derived.by((): readonly PricingRow[] => {
    if (rows !== undefined) {
      return rows;
    }

    if (features !== undefined) {
      return features.map((feature, index) => ({
        id: `${feature.label}-${index}`,
        label: feature.label,
        description: undefined,
        values: feature.values,
      }));
    }

    return [];
  });

  const normalizeCell = (cell: PricingCell | string): PricingCell => {
    if (typeof cell === 'string') {
      return { value: cell };
    }

    return cell;
  };
</script>

<div class={rootClassName} {...restProps}>
  <table class={styles.table ?? ''}>
    <thead>
      <tr>
        <th class={styles.headingCell ?? ''}>{heading}</th>
        {#each resolvedColumns as column (column.id)}
          <th>
            <div class={styles.planName ?? ''}>{column.title}</div>
            {#if hasText(column.subtitle)}
              <div class={styles.columnSubtitle ?? ''}>{column.subtitle}</div>
            {/if}
            {#if hasText(column.highlight)}
              <div class={styles.planPrice ?? ''}>{column.highlight}</div>
            {/if}
          </th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#each resolvedRows as row (row.id)}
        <tr>
          <td class={styles.featureLabel ?? ''}>
            <div>{row.label}</div>
            {#if hasText(row.description)}
              <div class={styles.rowDescription ?? ''}>{row.description}</div>
            {/if}
          </td>
          {#each row.values as cell, index (`${row.id}-${resolvedColumns[index]?.id ?? String(index)}`)}
            {@const normalizedCell = normalizeCell(cell)}
            <td class={`${styles.featureValue ?? ''} ${normalizedCell.emphasis ? (styles.emphasis ?? '') : ''}`}>
              <div>{normalizedCell.value}</div>
              {#if hasText(normalizedCell.supportingText)}
                <div class={styles.cellSupportingText ?? ''}>{normalizedCell.supportingText}</div>
              {/if}
            </td>
          {/each}
        </tr>
      {/each}
    </tbody>
  </table>
</div>

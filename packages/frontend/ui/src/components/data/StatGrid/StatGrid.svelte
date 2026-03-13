<script lang="ts">
  import MetricGrid from '@ui/components/data/MetricGrid/MetricGrid.svelte';

  type MetricTone = 'primary' | 'success' | 'warning' | 'info' | 'neutral';

  interface StatGridItem {
    label: string;
    value: string;
    trend?: string;
    context?: string;
    tone?: MetricTone;
  }

  interface StatGridProps {
    items?: readonly StatGridItem[];
    columns?: 1 | 2 | 3 | 4;
    className?: string;
    itemClassName?: string;
    labelClassName?: string;
    valueClassName?: string;
    metaClassName?: string;
    trendClassName?: string;
    contextClassName?: string;
    toneClassNames?: Partial<Record<MetricTone, string>>;
  }

  let {
    items = [],
    columns = 4,
    className,
    itemClassName,
    labelClassName,
    valueClassName,
    metaClassName,
    trendClassName,
    contextClassName,
    toneClassNames,
  }: StatGridProps = $props();
</script>

<MetricGrid
  {items}
  {columns}
  {className}
  {itemClassName}
  {labelClassName}
  {valueClassName}
  {metaClassName}
  {trendClassName}
  {contextClassName}
  {toneClassNames}
></MetricGrid>

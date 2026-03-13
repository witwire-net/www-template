<script lang="ts">
  import Collection from '@ui/components/organisms/Collection/Collection.svelte';

  import styles from './MetricGrid.module.scss';

  type MetricTone = 'primary' | 'success' | 'warning' | 'info' | 'neutral';

  interface MetricItem {
    label: string;
    value: string;
    trend?: string;
    context?: string;
    tone?: MetricTone;
  }

  interface MetricGridProps {
    items?: readonly MetricItem[];
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

  const toneClassDefaults: Record<MetricTone, string> = {
    neutral: styles.neutral ?? '',
    primary: styles.primary ?? '',
    success: styles.success ?? '',
    warning: styles.warning ?? '',
    info: styles.info ?? '',
  };

  const joinClassNames = (...classNames: (string | undefined)[]): string => {
    return classNames.filter((value) => value !== undefined && value !== '').join(' ');
  };

  const getMetricToneClassName = (
    tone: MetricTone,
    toneClassNames?: Partial<Record<MetricTone, string>>
  ): string => {
    switch (tone) {
      case 'primary': {
        return toneClassNames?.primary ?? toneClassDefaults.primary;
      }
      case 'success': {
        return toneClassNames?.success ?? toneClassDefaults.success;
      }
      case 'warning': {
        return toneClassNames?.warning ?? toneClassDefaults.warning;
      }
      case 'info': {
        return toneClassNames?.info ?? toneClassDefaults.info;
      }
      case 'neutral':
      default: {
        return toneClassNames?.neutral ?? toneClassDefaults.neutral;
      }
    }
  };

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
  }: MetricGridProps = $props();

  const rootClassName = $derived(joinClassNames(styles.grid, className));
  const cardClassName = $derived(joinClassNames(styles.card, itemClassName));
</script>

<Collection
  items={items}
  {columns}
  className={rootClassName}
  itemClassName={cardClassName}
  getKey={(item, index) => `${item.label}-${item.value}-${String(index)}`}
  renderItem={metricCard}
></Collection>

{#snippet metricCard(item: MetricItem)}
  {@const tone = item.tone ?? 'neutral'}
  {@const toneClassName = getMetricToneClassName(tone, toneClassNames)}

  <div class={joinClassNames(styles.label, labelClassName)}>{item.label}</div>
  <div class={joinClassNames(styles.value, valueClassName)}>{item.value}</div>

  {#if item.trend !== undefined || item.context !== undefined}
    <div class={joinClassNames(styles.meta, metaClassName)}>
      {#if item.trend !== undefined && item.trend !== ''}
        <span class={joinClassNames(styles.trend, toneClassName, trendClassName)}>{item.trend}</span>
      {/if}
      {#if item.context !== undefined && item.context !== ''}
        <span class={joinClassNames(styles.context, contextClassName)}>{item.context}</span>
      {/if}
    </div>
  {/if}
{/snippet}

<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Stat.module.scss';

  interface StatProps extends HTMLAttributes<HTMLDivElement> {
    label: string;
    value: string;
    description?: string;
    trend?: 'up' | 'down' | 'neutral';
    change?: string;
    className?: string;
  }

  let {
    label,
    value,
    description,
    trend = 'neutral',
    change,
    className,
    ...restProps
  }: StatProps = $props();

  const getTrendClassName = (value: NonNullable<StatProps['trend']>): string => {
    switch (value) {
      case 'up': {
        return styles.up ?? '';
      }
      case 'down': {
        return styles.down ?? '';
      }
      case 'neutral':
      default: {
        return styles.neutral ?? '';
      }
    }
  };

  const rootClassName = $derived([styles.stat ?? '', className ?? ''].filter((value_) => value_ !== '').join(' '));
  const hasMeta = $derived(
    (description !== undefined && description !== '') || (change !== undefined && change !== '')
  );
  const hasChange = $derived(change !== undefined && change !== '');
  const hasDescription = $derived(description !== undefined && description !== '');
</script>

<div class={rootClassName} {...restProps}>
  <div class={styles.label ?? ''}>{label}</div>
  <div class={styles.value ?? ''}>{value}</div>
  {#if hasMeta}
    <div class={styles.meta ?? ''}>
      {#if hasChange}
        <span class={[styles.change ?? '', getTrendClassName(trend)].filter((value_) => value_ !== '').join(' ')}>
          {change}
        </span>
      {/if}
      {#if hasDescription}
        <span class={styles.description ?? ''}>{description}</span>
      {/if}
    </div>
  {/if}
</div>

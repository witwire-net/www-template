<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './UsageMeter.module.scss';

  interface UsageMeterProps extends HTMLAttributes<HTMLDivElement> {
    label: string;
    used: number;
    limit: number;
    unit?: string;
    className?: string;
  }

  let { label, used, limit, unit, className, ...restProps }: UsageMeterProps = $props();

  const percent = $derived(Math.min(100, Math.round((used / Math.max(limit, 1)) * 100)));
  const hasUnit = $derived(unit !== undefined && unit !== '');
  const rootClassName = $derived([styles.meter ?? '', className ?? '']
    .filter((value) => value !== '')
    .join(' '));
  const displayValue = $derived(`${String(used)}/${String(limit)}${hasUnit ? ` ${unit}` : ''}`);
  const fillStyle = $derived(`width: ${String(percent)}%;`);
</script>

<div class={rootClassName} {...restProps}>
  <div class={styles.header ?? ''}>
    <span class={styles.label ?? ''}>{label}</span>
    <span class={styles.value ?? ''}>{displayValue}</span>
  </div>
  <div class={styles.track ?? ''}>
    <div class={styles.fill ?? ''} style={fillStyle}></div>
  </div>
</div>

<script lang="ts">
  import styles from './StatusIndicator.module.scss';

  interface StatusIndicatorProps {
    status: 'active' | 'idle' | 'warning' | 'error';
    label?: string;
  }

  let { status, label }: StatusIndicatorProps = $props();

  const getStatusClassName = (value: StatusIndicatorProps['status']): string => {
    if (value === 'active') {
      return styles.active ?? '';
    }

    if (value === 'idle') {
      return styles.idle ?? '';
    }

    if (value === 'warning') {
      return styles.warning ?? '';
    }

    return styles.error ?? '';
  };

  const dotClassName = $derived(
    [styles.dot ?? '', getStatusClassName(status)].filter((value) => value !== '').join(' ')
  );
  const hasLabel = $derived(label !== undefined);
</script>

<div class={styles.indicator}>
  <span class={dotClassName}></span>
  {#if hasLabel}
    <span class={styles.label}>{label}</span>
  {/if}
</div>

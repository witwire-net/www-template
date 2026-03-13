<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Divider.module.scss';

  type Props = HTMLAttributes<HTMLDivElement> & {
    align?: 'left' | 'center' | 'right';
    className?: string;
    label?: string;
    orientation?: 'horizontal' | 'vertical';
  };

  let {
    orientation = 'horizontal',
    label = undefined,
    align = 'center',
    className = undefined,
    ...restProps
  }: Props = $props();

  const hasLabel = $derived(label !== undefined && label !== '');
  const rootClassName = $derived(
    [
      styles.divider ?? '',
      styles[orientation] ?? '',
      hasLabel ? (styles.withLabel ?? '') : '',
      styles[align] ?? '',
      className ?? '',
    ]
      .filter((value) => value !== '')
      .join(' ')
  );
</script>

<div class={rootClassName} role="separator" {...restProps}>
  {#if hasLabel}
    <span class={styles.label ?? ''}>{label}</span>
  {/if}
</div>

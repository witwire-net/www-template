<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Chip.module.scss';

  type Props = HTMLAttributes<HTMLDivElement> & {
    children?: Snippet | string;
    className?: string;
    onRemove?: () => void;
    size?: 'sm' | 'md';
    variant?: 'default' | 'primary' | 'success' | 'warning' | 'error' | 'info';
  };

  let {
    variant = 'default',
    size = 'md',
    onRemove = undefined,
    className = undefined,
    children,
    ...restProps
  }: Props = $props();

  const hasRemove = $derived(onRemove !== undefined);
  const rootClassName = $derived(
    [styles.chip ?? '', styles[variant] ?? '', styles[size] ?? '', className ?? '']
      .filter((value) => value !== '')
      .join(' ')
  );
</script>

<div class={rootClassName} {...restProps}>
  <span class={styles.label ?? ''}>
    {#if typeof children === 'function'}
      {@render children()}
    {:else if typeof children === 'string'}
      {children}
    {/if}
  </span>
  {#if hasRemove}
    <button type="button" class={styles.remove ?? ''} aria-label="Remove" onclick={() => onRemove?.()}>
      x
    </button>
  {/if}
</div>

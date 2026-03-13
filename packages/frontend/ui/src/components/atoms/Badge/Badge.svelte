<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Badge.module.scss';

  type Props = HTMLAttributes<HTMLSpanElement> & {
    children?: Snippet | string;
    className?: string;
    size?: 'sm' | 'md';
    variant?: 'primary' | 'neutral' | 'success' | 'warning' | 'error' | 'info';
  };

  let {
    variant = 'neutral',
    size = 'md',
    className = undefined,
    children,
    ...restProps
  }: Props = $props();

  const rootClassName = $derived(
    [styles.badge ?? '', styles[variant] ?? '', styles[size] ?? '', className ?? '']
      .filter((value) => value !== '')
      .join(' ')
  );
</script>

<span class={rootClassName} {...restProps}>
  {#if typeof children === 'function'}
    {@render children()}
  {:else if typeof children === 'string'}
    {children}
  {/if}
</span>

<script lang="ts">
  import { IconCircle } from '@tabler/icons-svelte';
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Tag.module.scss';

  type Props = HTMLAttributes<HTMLSpanElement> & {
    children?: Snippet | string;
    className?: string;
    icon?: typeof IconCircle;
    size?: 'sm' | 'md';
    variant?: 'default' | 'primary' | 'success' | 'warning' | 'error' | 'info';
  };

  let {
    variant = 'default',
    size = 'md',
    icon = undefined,
    className = undefined,
    children,
    ...restProps
  }: Props = $props();

  const rootClassName = $derived(
    [styles.tag ?? '', styles[variant] ?? '', styles[size] ?? '', className ?? '']
      .filter((value) => value !== '')
      .join(' ')
  );
  const hasIcon = $derived(icon !== undefined);
  const IconComponent = $derived(icon ?? IconCircle);
</script>

<span class={rootClassName} {...restProps}>
  {#if hasIcon}
    <span class={styles.icon ?? ''}>
      <IconComponent />
    </span>
  {/if}
  {#if typeof children === 'function'}
    {@render children()}
  {:else if typeof children === 'string'}
    {children}
  {/if}
</span>

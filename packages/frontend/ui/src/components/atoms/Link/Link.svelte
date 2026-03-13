<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAnchorAttributes } from 'svelte/elements';

  import styles from './Link.module.scss';

  type Props = HTMLAnchorAttributes & {
    children?: Snippet | string;
    className?: string;
    underline?: 'none' | 'hover' | 'always';
    variant?: 'default' | 'primary' | 'muted' | 'ghost';
  };

  let {
    variant = 'default',
    underline = 'hover',
    className = undefined,
    children,
    ...restProps
  }: Props = $props();

  const rootClassName = $derived(
    [styles.link ?? '', styles[variant] ?? '', styles[underline] ?? '', className ?? '']
      .filter((value) => value !== '')
      .join(' ')
  );
</script>

<a class={rootClassName} {...restProps}>
  {#if typeof children === 'function'}
    {@render children()}
  {:else if typeof children === 'string'}
    {children}
  {/if}
</a>

<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Grid.module.scss';

  type Props = HTMLAttributes<HTMLDivElement> & {
    children?: Snippet | string;
    className?: string;
  };

  let { class: classProp = undefined, className = undefined, children, ...restProps }: Props = $props();

  const rootClassName = $derived(
    [styles.row ?? '', classProp ?? '', className ?? ''].filter((value) => value !== '').join(' ')
  );
</script>

<div class={rootClassName} {...restProps}>
  {#if typeof children === 'function'}
    {@render children()}
  {:else if typeof children === 'string'}
    {children}
  {/if}
</div>

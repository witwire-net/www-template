<svelte:options runes={true} />

<script lang="ts">
  import type { Snippet } from 'svelte';

  import styles from './Tooltip.module.scss';

  type Props = {
    content: string;
    placement?: 'top' | 'bottom' | 'left' | 'right';
    children?: Snippet;
  };

  let { content, placement = 'top', children = undefined }: Props = $props();

  const placementClassName = $derived(styles[placement] ?? styles.top ?? '');
</script>

<span class={styles.wrapper ?? ''}>
  {@render children?.()}
  <span class={[styles.tooltip ?? '', placementClassName].filter((value) => value !== '').join(' ')}>
    {content}
  </span>
</span>

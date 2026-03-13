<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Card.module.scss';

  interface CardSectionProps extends HTMLAttributes<HTMLDivElement> {
    className?: string;
    children?: Snippet;
  }

  let { className, children, ...restProps }: CardSectionProps = $props();

  const rootClassName = $derived(
    [styles.header ?? '', className ?? ''].filter((value) => value !== '').join(' ')
  );
</script>

<div class={rootClassName} {...restProps}>
  {@render children?.()}
</div>

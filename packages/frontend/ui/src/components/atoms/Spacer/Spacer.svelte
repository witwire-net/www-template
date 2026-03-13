<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Spacer.module.scss';

  type Props = HTMLAttributes<HTMLDivElement> & {
    axis?: 'vertical' | 'horizontal';
    className?: string;
    size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl';
  };

  let { size = 'md', axis = 'vertical', className = undefined, ...restProps }: Props = $props();

  const sizeClassName = $derived(size === '2xl' ? (styles.twoXl ?? '') : (styles[size] ?? ''));
  const rootClassName = $derived(
    [styles.spacer ?? '', sizeClassName, styles[axis] ?? '', className ?? '']
      .filter((value) => value !== '')
      .join(' ')
  );
</script>

<div class={rootClassName} aria-hidden="true" {...restProps}></div>

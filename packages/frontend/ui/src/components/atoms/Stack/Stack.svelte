<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Stack.module.scss';

  type Props = HTMLAttributes<HTMLDivElement> & {
    align?: 'start' | 'center' | 'end' | 'stretch';
    children?: Snippet | string;
    className?: string;
    direction?: 'row' | 'column';
    gap?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl';
    justify?: 'start' | 'center' | 'end' | 'between' | 'around';
    style?: string;
    wrap?: boolean;
  };

  let {
    direction = 'column',
    align = 'stretch',
    justify = 'start',
    wrap = false,
    gap = 'md',
    className = undefined,
    style = undefined,
    children,
    ...restProps
  }: Props = $props();

  const rootClassName = $derived(
    [
      styles.stack ?? '',
      styles[direction] ?? '',
      styles[`align-${align}`] ?? '',
      styles[`justify-${justify}`] ?? '',
      wrap ? (styles.wrap ?? '') : '',
      styles[`gap-${gap}`] ?? '',
      className ?? '',
    ]
      .filter((value) => value !== '')
      .join(' ')
  );
</script>

<div class={rootClassName} style={style} {...restProps}>
  {#if typeof children === 'function'}
    {@render children()}
  {:else if typeof children === 'string'}
    {children}
  {/if}
</div>

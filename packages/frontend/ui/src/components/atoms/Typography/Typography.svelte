<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Typography.module.scss';

  type TypographyVariant =
    | 'h1'
    | 'h2'
    | 'h3'
    | 'h4'
    | 'h5'
    | 'h6'
    | 'subtitle'
    | 'body'
    | 'body-sm'
    | 'caption'
    | 'overline';

  type Props = HTMLAttributes<HTMLElement> & {
    align?: 'left' | 'center' | 'right';
    as?: string;
    children?: Snippet | string;
    className?: string;
    color?: 'default' | 'secondary' | 'muted' | 'primary';
    truncate?: boolean;
    variant?: TypographyVariant;
    weight?: 'regular' | 'medium' | 'bold' | 'black';
  };

  let {
    as = undefined,
    variant = 'body',
    weight = 'regular',
    color = 'default',
    align = 'left',
    truncate = false,
    class: classProp = undefined,
    className = undefined,
    children,
    ...restProps
  }: Props = $props();

  const defaultTagByVariant: Record<TypographyVariant, string> = {
    h1: 'h1',
    h2: 'h2',
    h3: 'h3',
    h4: 'h4',
    h5: 'h5',
    h6: 'h6',
    subtitle: 'p',
    body: 'p',
    'body-sm': 'p',
    caption: 'span',
    overline: 'span',
  };

  const componentTag = $derived(as ?? defaultTagByVariant[variant]);
  const rootClassName = $derived(
    [
      styles.typography ?? '',
      styles[variant] ?? '',
      styles[weight] ?? '',
      styles[color] ?? '',
      styles[align] ?? '',
      truncate ? (styles.truncate ?? '') : '',
      classProp ?? '',
      className ?? '',
    ]
      .filter((value) => value !== '')
      .join(' ')
  );
</script>

<svelte:element this={componentTag} class={rootClassName} {...restProps}>
  {#if typeof children === 'function'}
    {@render children()}
  {:else if typeof children === 'string'}
    {children}
  {/if}
</svelte:element>

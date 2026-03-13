<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Card.module.scss';

  interface CardProps extends HTMLAttributes<HTMLDivElement> {
    width?: string | number;
    height?: string | number;
    padding?: 'none' | 'sm' | 'md' | 'lg' | 'xl';
    variant?: 'default' | 'unstyled';
    className?: string;
    children?: Snippet;
  }

  let {
    width,
    height,
    padding = 'none',
    variant = 'default',
    className,
    style,
    children,
    ...restProps
  }: CardProps = $props();

  const toCssDimension = (value: string | number | undefined): string | undefined => {
    if (typeof value === 'number') {
      return `${value}px`;
    }

    return value;
  };

  const rootClassName = $derived([
    styles.card ?? '',
    variant === 'unstyled' ? (styles.unstyled ?? '') : '',
    className ?? '',
  ]
    .filter((value) => value !== '')
    .join(' '));

  const mergedStyle = $derived([
    toCssDimension(width) !== undefined ? `width: ${toCssDimension(width)};` : '',
    toCssDimension(height) !== undefined ? `height: ${toCssDimension(height)};` : '',
    padding !== 'none' ? `padding: var(--spacing-${padding});` : '',
    style ?? '',
  ]
    .filter((value) => value !== '')
    .join(' '));
</script>

<div class={rootClassName} style={mergedStyle === '' ? undefined : mergedStyle} {...restProps}>
  {@render children?.()}
</div>

<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Skeleton.module.scss';

  type Props = HTMLAttributes<HTMLDivElement> & {
    className?: string;
    height?: string | number;
    variant?: 'text' | 'circular' | 'rectangular';
    width?: string | number;
  };

  let {
    variant = 'text',
    width = undefined,
    height = undefined,
    className = undefined,
    ...restProps
  }: Props = $props();

  function toCssDimension(value: string | number | undefined): string | undefined {
    if (value === undefined) {
      return undefined;
    }

    return typeof value === 'number' ? `${value}px` : value;
  }

  const variantClassName = $derived(
    variant === 'circular' ? (styles.circle ?? '') : variant === 'text' ? (styles.text ?? '') : (styles.rect ?? '')
  );
  const rootClassName = $derived(
    [styles.skeleton ?? '', variantClassName, className ?? ''].filter((value) => value !== '').join(' ')
  );
  const styleValue = $derived(
    [
      toCssDimension(width) === undefined ? '' : `width: ${toCssDimension(width)};`,
      toCssDimension(height) === undefined ? '' : `height: ${toCssDimension(height)};`,
    ]
      .filter((value) => value !== '')
      .join(' ')
  );
</script>

<div class={rootClassName} style={styleValue === '' ? undefined : styleValue} aria-hidden="true" {...restProps}></div>

<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLButtonAttributes } from 'svelte/elements';

  import styles from './Button.module.scss';

  type Props = HTMLButtonAttributes & {
    children?: Snippet | string;
    className?: string;
    fullWidth?: boolean;
    isLoading?: boolean;
    size?: 'sm' | 'md' | 'lg';
    variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger';
  };

  let {
    variant = 'primary',
    size = 'md',
    fullWidth = false,
    isLoading = false,
    class: classProp = undefined,
    className = undefined,
    disabled = undefined,
    children,
    ...restProps
  }: Props = $props();

  const rootClassName = $derived(
    [
      styles.button ?? '',
      styles[variant] ?? '',
      styles[size] ?? '',
      fullWidth ? (styles.fullWidth ?? '') : '',
      classProp ?? '',
      className ?? '',
    ]
      .filter((value) => value !== '')
      .join(' ')
  );
  const isDisabled = $derived(disabled === true || isLoading);
</script>

<button class={rootClassName} disabled={isDisabled} {...restProps}>
  {#if isLoading}
    <span class={styles.spinner ?? ''}></span>
  {/if}
  {#if typeof children === 'function'}
    {@render children()}
  {:else if typeof children === 'string'}
    {children}
  {/if}
</button>

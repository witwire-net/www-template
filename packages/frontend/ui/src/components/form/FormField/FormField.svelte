<svelte:options runes={true} />

<script lang="ts">
  import type { Snippet } from 'svelte';

  import styles from './FormField.module.scss';

  type Props = Record<string, unknown> & {
    label?: string;
    helperText?: string;
    error?: string;
    required?: boolean;
    htmlFor?: string;
    className?: string;
    children?: Snippet;
  };

  const joinClassName = (...values: (string | undefined)[]) =>
    values.filter((value) => value !== undefined && value !== '').join(' ');

  const {
    label = undefined,
    helperText = undefined,
    error = undefined,
    required = false,
    htmlFor = undefined,
    className = undefined,
    children,
    ...restProps
  }: Props = $props();

  const hasLabel = $derived(label !== undefined && label !== '');
  const hasError = $derived(error !== undefined && error !== '');
  const hasHelperText = $derived(helperText !== undefined && helperText !== '');
  const rootClassName = $derived(joinClassName(styles.field, className));
</script>

<div class={rootClassName} {...restProps}>
  {#if hasLabel}
    <label class={styles.label ?? ''} for={htmlFor}>
      {label}
      {#if required}
        <span class={styles.required ?? ''}>*</span>
      {/if}
    </label>
  {/if}
  <div class={styles.control ?? ''}>
    {#if children !== undefined}
      {@render children()}
    {/if}
  </div>
  {#if hasError}
    <span class={styles.errorMessage ?? ''}>{error}</span>
  {:else if hasHelperText}
    <span class={styles.helperText ?? ''}>{helperText}</span>
  {/if}
</div>

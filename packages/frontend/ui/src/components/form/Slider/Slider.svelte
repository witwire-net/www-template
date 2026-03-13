<svelte:options runes={true} />

<script lang="ts">
  import styles from './Slider.module.scss';

  type Props = Record<string, unknown> & {
    label?: string;
    helperText?: string;
    error?: string;
    className?: string;
  };

  const joinClassName = (...values: (string | undefined)[]) =>
    values.filter((value) => value !== undefined && value !== '').join(' ');

  const {
    label = undefined,
    helperText = undefined,
    error = undefined,
    className = undefined,
    ...restProps
  }: Props = $props();

  const hasLabel = $derived(label !== undefined && label !== '');
  const hasError = $derived(error !== undefined && error !== '');
  const hasHelperText = $derived(!hasError && helperText !== undefined && helperText !== '');
  const sliderClassName = $derived(
    joinClassName(styles.slider, hasError ? styles.error : undefined, className)
  );
</script>

<div class={styles.wrapper ?? ''}>
  {#if hasLabel}
    <span class={styles.label ?? ''}>{label}</span>
  {/if}
  <input type="range" class={sliderClassName} {...restProps} />
  {#if hasError}
    <span class={styles.errorMessage ?? ''}>{error}</span>
  {:else if hasHelperText}
    <span class={styles.helperText ?? ''}>{helperText}</span>
  {/if}
</div>

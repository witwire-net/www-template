<script module lang="ts">
  let inputIdCounter = 0;
</script>

<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';

  import styles from './Input.module.scss';

  type Props = Omit<HTMLInputAttributes, 'size'> & {
    className?: string;
    error?: string;
    fullWidth?: boolean;
    helperText?: string;
    label?: string;
    size?: 'sm' | 'md' | 'lg';
  };

  let {
    id = undefined,
    label = undefined,
    size = 'md',
    error = undefined,
    helperText = undefined,
    fullWidth = true,
    className = undefined,
    disabled = false,
    ...restProps
  }: Props = $props();

  const fallbackId = `input-${++inputIdCounter}`;
  const inputId = $derived(id ?? fallbackId);
  const hasLabel = $derived(label !== undefined && label !== '');
  const hasError = $derived(error !== undefined && error !== '');
  const hasHelperText = $derived(helperText !== undefined && helperText !== '');
  const rootClassName = $derived(
    [styles.input ?? '', styles[size] ?? '', hasError ? (styles.error ?? '') : '', className ?? '']
      .filter((value) => value !== '')
      .join(' ')
  );
  const describedBy = $derived(hasError ? `${inputId}-error` : hasHelperText ? `${inputId}-helper` : undefined);
  const wrapperStyle = $derived(fullWidth ? 'width: 100%;' : 'width: auto;');
</script>

<div class={styles.inputWrapper ?? ''} style={wrapperStyle}>
  {#if hasLabel}
    <label for={inputId} class={styles.label ?? ''}>{label}</label>
  {/if}
  <input
    id={inputId}
    class={rootClassName}
    disabled={disabled}
    aria-invalid={hasError}
    aria-describedby={describedBy}
    {...restProps}
  />
  {#if hasError}
    <span id={`${inputId}-error`} class={styles.errorMessage ?? ''}>{error}</span>
  {:else if hasHelperText}
    <span id={`${inputId}-helper`} class={styles.helperText ?? ''}>{helperText}</span>
  {/if}
</div>

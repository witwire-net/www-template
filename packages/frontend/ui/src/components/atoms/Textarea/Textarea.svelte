<script module lang="ts">
  let textareaIdCounter = 0;
</script>

<script lang="ts">
  import type { HTMLTextareaAttributes } from 'svelte/elements';

  import styles from '@ui/components/atoms/Input/Input.module.scss';

  type Props = HTMLTextareaAttributes & {
    className?: string;
    error?: string;
    fullWidth?: boolean;
    helperText?: string;
    label?: string;
  };

  let {
    id = undefined,
    label = undefined,
    error = undefined,
    helperText = undefined,
    fullWidth = true,
    className = undefined,
    disabled = false,
    ...restProps
  }: Props = $props();

  const fallbackId = `textarea-${++textareaIdCounter}`;
  const inputId = $derived(id ?? fallbackId);
  const hasLabel = $derived(label !== undefined && label !== '');
  const hasError = $derived(error !== undefined && error !== '');
  const hasHelperText = $derived(helperText !== undefined && helperText !== '');
  const rootClassName = $derived(
    [styles.input ?? '', styles.textarea ?? '', hasError ? (styles.error ?? '') : '', className ?? '']
      .filter((value) => value !== '')
      .join(' ')
  );
  const wrapperStyle = $derived(fullWidth ? 'width: 100%;' : 'width: auto;');
</script>

<div class={styles.inputWrapper ?? ''} style={wrapperStyle}>
  {#if hasLabel}
    <label for={inputId} class={styles.label ?? ''}>{label}</label>
  {/if}
  <textarea id={inputId} class={rootClassName} disabled={disabled} aria-invalid={hasError} {...restProps}></textarea>
  {#if hasError}
    <span class={styles.errorMessage ?? ''}>{error}</span>
  {:else if hasHelperText}
    <span class={styles.helperText ?? ''}>{helperText}</span>
  {/if}
</div>

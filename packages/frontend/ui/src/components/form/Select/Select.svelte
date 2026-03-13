<svelte:options runes={true} />

<script module lang="ts">
  let selectIdCounter = 0;
</script>

<script lang="ts">
  import styles from './Select.module.scss';

  interface SelectOption {
    label: string;
    value: string;
    disabled?: boolean;
  }

  type Props = Record<string, unknown> & {
    id?: string;
    label?: string;
    options?: SelectOption[];
    placeholder?: string;
    size?: 'sm' | 'md' | 'lg';
    error?: string;
    helperText?: string;
    fullWidth?: boolean;
    className?: string;
    disabled?: boolean;
  };

  const joinClassName = (...values: (string | undefined)[]) =>
    values.filter((value) => value !== undefined && value !== '').join(' ');

  const {
    id = undefined,
    label = undefined,
    options = [],
    placeholder = undefined,
    size = 'md',
    error = undefined,
    helperText = undefined,
    fullWidth = true,
    className = undefined,
    disabled = false,
    ...restProps
  }: Props = $props();

  const fallbackId = `select-${++selectIdCounter}`;

  const selectId = $derived(id ?? fallbackId);
  const hasPlaceholder = $derived(placeholder !== undefined && placeholder !== '');
  const hasError = $derived(error !== undefined && error !== '');
  const hasHelperText = $derived(!hasError && helperText !== undefined && helperText !== '');
  const rootClassName = $derived(
    joinClassName(styles.select, styles[size], hasError ? styles.error : undefined, className)
  );
  const wrapperStyle = $derived(fullWidth ? 'width: 100%;' : 'width: auto;');
  const describedBy = $derived(
    hasError ? `${selectId}-error` : hasHelperText ? `${selectId}-helper` : undefined
  );
</script>

<div class={styles.wrapper ?? ''} style={wrapperStyle}>
  {#if label !== undefined && label !== ''}
    <label for={selectId} class={styles.label ?? ''}>{label}</label>
  {/if}
  <div class={styles.selectWrapper ?? ''}>
    <select
      id={selectId}
      class={rootClassName}
      {disabled}
      aria-invalid={hasError}
      aria-describedby={describedBy}
      {...restProps}
    >
      {#if hasPlaceholder}
        <option value="">{placeholder}</option>
      {/if}
      {#each options as option (option.value)}
        <option value={option.value} disabled={option.disabled}>{option.label}</option>
      {/each}
    </select>
    <span class={styles.arrow ?? ''} aria-hidden="true"></span>
  </div>
  {#if hasError}
    <span id={`${selectId}-error`} class={styles.errorMessage ?? ''}>{error}</span>
  {:else if hasHelperText}
    <span id={`${selectId}-helper`} class={styles.helperText ?? ''}>{helperText}</span>
  {/if}
</div>

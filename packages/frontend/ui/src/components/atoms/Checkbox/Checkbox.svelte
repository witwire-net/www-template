<script module lang="ts">
  let checkboxIdCounter = 0;
</script>

<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';

  import styles from './Checkbox.module.scss';

  type Props = Omit<HTMLInputAttributes, 'type'> & {
    className?: string;
    defaultChecked?: boolean;
    label?: string;
  };

  let {
    id = undefined,
    label = undefined,
    className = undefined,
    disabled = false,
    checked = undefined,
    defaultChecked = undefined,
    ...restProps
  }: Props = $props();

  const fallbackId = `checkbox-${++checkboxIdCounter}`;
  const inputId = $derived(id ?? fallbackId);
  const hasLabel = $derived(label !== undefined && label !== '');
  const rootClassName = $derived(
    [styles.wrapper ?? '', disabled ? (styles.disabled ?? '') : '', className ?? '']
      .filter((value) => value !== '')
      .join(' ')
  );
  const resolvedChecked = $derived(checked ?? defaultChecked);
</script>

<label for={inputId} class={rootClassName}>
  <input
    id={inputId}
    type="checkbox"
    class={styles.input ?? ''}
    disabled={disabled}
    checked={resolvedChecked}
    {...restProps}
  />
  <span class={styles.checkmark ?? ''}></span>
  {#if hasLabel}
    <span class={styles.label ?? ''}>{label}</span>
  {/if}
</label>

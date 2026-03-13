<svelte:options runes={true} />

<script module lang="ts">
  let radioIdCounter = 0;
</script>

<script lang="ts">
  import styles from './Radio.module.scss';

  type Props = Record<string, unknown> & {
    id?: string;
    label?: string;
    className?: string;
    disabled?: boolean;
    checked?: boolean;
    defaultChecked?: boolean;
  };

  const joinClassName = (...values: (string | undefined)[]) =>
    values.filter((value) => value !== undefined && value !== '').join(' ');

  const {
    id = undefined,
    label = undefined,
    className = undefined,
    disabled = false,
    checked = undefined,
    defaultChecked = undefined,
    ...restProps
  }: Props = $props();

  const fallbackId = `radio-${++radioIdCounter}`;

  const inputId = $derived(id ?? fallbackId);
  const hasLabel = $derived(label !== undefined && label !== '');
  const rootClassName = $derived(joinClassName(styles.wrapper, disabled ? styles.disabled : undefined, className));
  const resolvedChecked = $derived(checked ?? defaultChecked);
</script>

<label for={inputId} class={rootClassName}>
  <input
    id={inputId}
    type="radio"
    class={styles.input ?? ''}
    {disabled}
    checked={resolvedChecked}
    {...restProps}
  />
  <span class={styles.dot ?? ''}></span>
  {#if hasLabel}
    <span class={styles.label ?? ''}>{label}</span>
  {/if}
</label>

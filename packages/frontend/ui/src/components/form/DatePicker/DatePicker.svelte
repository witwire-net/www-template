<svelte:options runes={true} />

<script lang="ts">
  import Input from '@ui/components/atoms/Input/Input.svelte';

  type Props = Record<string, unknown> & {
    id?: string;
    label?: string;
    size?: 'sm' | 'md' | 'lg';
    error?: string;
    helperText?: string;
    fullWidth?: boolean;
    className?: string;
    disabled?: boolean;
  };

  const {
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
</script>

<Input
  type="date"
  {id}
  {label}
  {size}
  {error}
  {helperText}
  {fullWidth}
  {className}
  {disabled}
  {...restProps}
/>

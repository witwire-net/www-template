<script lang="ts">
  import Button from './Button.svelte';

  let {
    variant = 'primary',
    size = 'md',
    fullWidth = false,
    isLoading = false,
    disabled = false,
    text = 'Button',
  }: {
    disabled?: boolean;
    fullWidth?: boolean;
    isLoading?: boolean;
    size?: 'sm' | 'md' | 'lg';
    text?: string;
    variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger';
  } = $props();
</script>

<Button {variant} {size} {fullWidth} {isLoading} {disabled}>{text}</Button>

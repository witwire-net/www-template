<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import Dialog from '@ui/components/feedback/Dialog/Dialog.svelte';

  type Props = {
    open: boolean;
    title?: string;
    description?: string;
    confirmLabel?: string;
    cancelLabel?: string;
    onConfirm: () => void;
    onCancel: () => void;
  };

  let {
    open,
    title = 'Are you sure',
    description = undefined,
    confirmLabel = 'Confirm',
    cancelLabel = 'Cancel',
    onConfirm,
    onCancel,
  }: Props = $props();
</script>

<Dialog {open} onClose={onCancel} {title} {description}>
  {#snippet actions()}
    <Button variant="ghost" onclick={onCancel}>{cancelLabel}</Button>
    <Button variant="primary" onclick={onConfirm}>{confirmLabel}</Button>
  {/snippet}
</Dialog>

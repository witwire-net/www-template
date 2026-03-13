<svelte:options runes={true} />

<script lang="ts">
  import type { Snippet } from 'svelte';

  import Modal from '@ui/components/feedback/Modal/Modal.svelte';

  import styles from './Dialog.module.scss';

  type Props = {
    open: boolean;
    onClose?: () => void;
    size?: 'sm' | 'md' | 'lg' | 'xl';
    closeOnBackdrop?: boolean;
    title?: string;
    description?: string;
    children?: Snippet;
    actions?: Snippet;
  };

  let {
    open,
    onClose = undefined,
    size = 'md',
    closeOnBackdrop = true,
    title = undefined,
    description = undefined,
    children = undefined,
    actions = undefined,
  }: Props = $props();

  const hasTitle = $derived(typeof title === 'string' && title !== '');
  const hasDescription = $derived(typeof description === 'string' && description !== '');
  const hasBody = $derived(children !== undefined);
  const hasActions = $derived(actions !== undefined);
</script>

<Modal {open} {onClose} {size} {closeOnBackdrop}>
  <div class={styles.dialog ?? ''}>
    {#if hasTitle || hasDescription}
      <div class={styles.header ?? ''}>
        {#if hasTitle}
          <h3 class={styles.title ?? ''}>{title}</h3>
        {/if}
        {#if hasDescription}
          <p class={styles.description ?? ''}>{description}</p>
        {/if}
      </div>
    {/if}
    {#if hasBody}
      <div class={styles.body ?? ''}>
        {@render children?.()}
      </div>
    {/if}
    {#if hasActions}
      <div class={styles.actions ?? ''}>
        {@render actions?.()}
      </div>
    {/if}
  </div>
</Modal>

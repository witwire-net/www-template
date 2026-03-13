<svelte:options runes={true} />

<script lang="ts">
  import type { Snippet } from 'svelte';

  import styles from './Modal.module.scss';

  type Props = {
    open: boolean;
    onClose?: () => void;
    size?: 'sm' | 'md' | 'lg' | 'xl';
    closeOnBackdrop?: boolean;
    children?: Snippet;
  };

  let {
    open,
    onClose = undefined,
    size = 'md',
    closeOnBackdrop = true,
    children = undefined,
  }: Props = $props();

  function handleBackdropClick(): void {
    if (closeOnBackdrop) {
      onClose?.();
    }
  }

  function handleBackdropKeydown(event: KeyboardEvent): void {
    if (closeOnBackdrop && (event.key === 'Enter' || event.key === ' ')) {
      event.preventDefault();
      onClose?.();
    }
  }

  const modalClassName = $derived(
    [styles.modal ?? '', styles[size] ?? ''].filter((value) => value !== '').join(' ')
  );

  $effect(() => {
    if (typeof document === 'undefined' || !open) {
      return;
    }

    const handleKeydown = (event: KeyboardEvent): void => {
      if (event.key === 'Escape') {
        onClose?.();
      }
    };

    const previousOverflow = document.body.style.overflow;
    document.addEventListener('keydown', handleKeydown);
    document.body.style.overflow = 'hidden';

    return () => {
      document.removeEventListener('keydown', handleKeydown);
      document.body.style.overflow = previousOverflow;
    };
  });
</script>

{#if open}
  <div class={styles.overlay ?? ''} role="dialog" aria-modal="true">
    <div
      class={styles.backdrop ?? ''}
      role="button"
      tabindex="0"
      onclick={handleBackdropClick}
      onkeydown={handleBackdropKeydown}
    ></div>
    <div class={modalClassName}>
      {@render children?.()}
    </div>
  </div>
{/if}

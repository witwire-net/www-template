<svelte:options runes={true} />

<script lang="ts">
  import type { Snippet } from 'svelte';

  import styles from './Drawer.module.scss';

  type Props = {
    open: boolean;
    onClose?: () => void;
    position?: 'left' | 'right';
    width?: number | string;
    children?: Snippet;
  };

  let {
    open,
    onClose = undefined,
    position = 'right',
    width = 320,
    children = undefined,
  }: Props = $props();

  function handleClose(): void {
    onClose?.();
  }

  function handleBackdropKeydown(event: KeyboardEvent): void {
    if (onClose === undefined) {
      return;
    }

    if (event.key === 'Enter' || event.key === ' ' || event.key === 'Escape') {
      event.preventDefault();
      handleClose();
    }
  }

  const normalizedWidth = $derived(typeof width === 'number' ? `${String(width)}px` : width);
  const isCloseable = $derived(onClose !== undefined);
  const overlayClassName = $derived(
    [styles.overlay ?? '', open ? (styles.open ?? '') : ''].filter((value) => value !== '').join(' ')
  );
  const backdropClassName = $derived(
    [styles.backdrop ?? '', open ? (styles.open ?? '') : '']
      .filter((value) => value !== '')
      .join(' ')
  );
  const drawerClassName = $derived(
    [
      styles.drawer ?? '',
      position === 'left' ? (styles.left ?? '') : (styles.right ?? ''),
      open ? (styles.open ?? '') : '',
    ]
      .filter((value) => value !== '')
      .join(' ')
  );
  const isHidden = $derived(!open);

  $effect(() => {
    if (typeof document === 'undefined' || !open) {
      return;
    }

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    return () => {
      document.body.style.overflow = previousOverflow;
    };
  });
</script>

<div class={overlayClassName} inert={isHidden ? true : undefined} aria-hidden={isHidden ? 'true' : undefined}>
  <button
    class={backdropClassName}
    type="button"
    aria-label="Close drawer"
    disabled={!open || !isCloseable}
    onclick={open && isCloseable ? handleClose : undefined}
    onkeydown={open ? handleBackdropKeydown : undefined}
  ></button>
  <aside class={drawerClassName} style={`width: ${normalizedWidth};`}>
    {@render children?.()}
  </aside>
</div>

<svelte:options runes={true} />

<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Snackbar.module.scss';

  type Props = HTMLAttributes<HTMLDivElement> & {
    message: string;
    action?: Snippet;
    duration?: number;
    onClose?: () => void;
    className?: string;
  };

  let {
    message,
    action = undefined,
    duration = undefined,
    onClose = undefined,
    className = undefined,
    ...restProps
  }: Props = $props();

  const rootClassName = $derived(
    [styles.snackbar ?? '', className ?? ''].filter((value) => value !== '').join(' ')
  );
  const hasAction = $derived(action !== undefined);
  const hasClose = $derived(onClose !== undefined);

  $effect(() => {
    if (typeof window === 'undefined' || duration === undefined || duration === 0 || onClose === undefined) {
      return;
    }

    const timer = window.setTimeout(() => {
      onClose?.();
    }, duration);

    return () => {
      window.clearTimeout(timer);
    };
  });
</script>

<div class={rootClassName} {...restProps}>
  <span class={styles.message ?? ''}>{message}</span>
  {#if hasAction}
    <span class={styles.action ?? ''}>
      {@render action?.()}
    </span>
  {/if}
  {#if hasClose}
    <button type="button" class={styles.close ?? ''} onclick={onClose} aria-label="Close">
      ×
    </button>
  {/if}
</div>

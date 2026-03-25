<svelte:options runes={true} />

<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import {
    getTextContent,
    isSnippet,
    joinClassNames,
    type Renderable,
  } from '@ui/components/shared';

  import styles from './Toast.module.scss';

  type Props = HTMLAttributes<HTMLDivElement> & {
    variant?: 'info' | 'success' | 'warning' | 'error' | 'neutral';
    title?: string;
    description?: string;
    icon?: Renderable;
    actions?: Renderable;
    duration?: number;
    onClose?: () => void;
    className?: string;
    children?: Snippet;
  };

  let {
    variant = 'neutral',
    title = undefined,
    description = undefined,
    icon = undefined,
    actions = undefined,
    duration = undefined,
    onClose = undefined,
    className = undefined,
    children = undefined,
    ...restProps
  }: Props = $props();

  const rootClassName = $derived(joinClassNames(styles.toast ?? '', styles[variant] ?? '', className));
  const hasIcon = $derived(icon !== undefined);
  const hasTitle = $derived(typeof title === 'string' && title !== '');
  const hasDescription = $derived(typeof description === 'string' && description !== '');
  const hasChildren = $derived(children !== undefined);
  const hasActions = $derived(actions !== undefined);
  const hasClose = $derived(onClose !== undefined);

  $effect(() => {
    if (
      typeof window === 'undefined' ||
      typeof duration !== 'number' ||
      !Number.isFinite(duration) ||
      duration <= 0 ||
      onClose === undefined
    ) {
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
  {#if hasIcon}
    <div class={styles.icon ?? ''}>
      {#if isSnippet(icon)}
        {@render icon()}
      {:else}
        {getTextContent(icon)}
      {/if}
    </div>
  {/if}
  <div class={styles.content ?? ''}>
    {#if hasTitle}
      <div class={styles.title ?? ''}>{title}</div>
    {/if}
    {#if hasDescription}
      <div class={styles.description ?? ''}>{description}</div>
    {/if}
    {#if hasChildren}
      {@render children?.()}
    {/if}
  </div>
  {#if hasActions}
    <div class={styles.actions ?? ''}>
      {#if isSnippet(actions)}
        {@render actions()}
      {:else}
        {getTextContent(actions)}
      {/if}
    </div>
  {/if}
  {#if hasClose}
    <button type="button" class={styles.close ?? ''} onclick={onClose} aria-label="Close">
      ×
    </button>
  {/if}
</div>

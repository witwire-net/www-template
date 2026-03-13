<svelte:options runes={true} />

<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './EmptyState.module.scss';

  type Props = HTMLAttributes<HTMLDivElement> & {
    title: string;
    description?: string;
    icon?: Snippet;
    action?: Snippet;
    className?: string;
  };

  let {
    title,
    description = undefined,
    icon = undefined,
    action = undefined,
    className = undefined,
    ...restProps
  }: Props = $props();

  const rootClassName = $derived(
    [styles.empty ?? '', className ?? ''].filter((value) => value !== '').join(' ')
  );
  const hasIcon = $derived(icon !== undefined);
  const hasDescription = $derived(typeof description === 'string' && description !== '');
  const hasAction = $derived(action !== undefined);
</script>

<div class={rootClassName} {...restProps}>
  {#if hasIcon}
    <div class={styles.icon ?? ''}>
      {@render icon?.()}
    </div>
  {/if}
  <div class={styles.title ?? ''}>{title}</div>
  {#if hasDescription}
    <div class={styles.description ?? ''}>{description}</div>
  {/if}
  {#if hasAction}
    <div class={styles.action ?? ''}>
      {@render action?.()}
    </div>
  {/if}
</div>

<svelte:options runes={true} />

<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './Alert.module.scss';

  type Renderable = Snippet | string | number | null | undefined;

  type Props = HTMLAttributes<HTMLDivElement> & {
    variant?: 'info' | 'success' | 'warning' | 'error' | 'neutral';
    title?: string;
    description?: string;
    icon?: Renderable;
    actions?: Renderable;
    children?: Snippet;
    className?: string;
  };

  let {
    variant = 'neutral',
    title = undefined,
    description = undefined,
    icon = undefined,
    actions = undefined,
    children = undefined,
    className = undefined,
    ...restProps
  }: Props = $props();

  function isSnippet(value: Renderable): value is Snippet {
    return typeof value === 'function';
  }

  function getTextContent(value: Renderable): string {
    if (typeof value === 'string') {
      return value;
    }

    if (typeof value === 'number') {
      return String(value);
    }

    return '';
  }

  const rootClassName = $derived(
    [styles.alert ?? '', styles[variant] ?? '', className ?? '']
      .filter((value) => value !== '')
      .join(' ')
  );

  const hasIcon = $derived(icon !== undefined);
  const hasTitle = $derived(typeof title === 'string' && title !== '');
  const hasDescription = $derived(typeof description === 'string' && description !== '');
  const hasChildren = $derived(children !== undefined);
  const hasActions = $derived(actions !== undefined);
</script>

<div class={rootClassName} role="alert" {...restProps}>
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
</div>

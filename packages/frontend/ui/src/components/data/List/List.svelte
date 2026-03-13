<script lang="ts">
  import type { Snippet } from 'svelte';

  import styles from './List.module.scss';

  interface ListItem {
    title: string;
    description?: string;
    meta?: string;
    icon?: unknown;
    action?: unknown;
    iconContent?: Snippet;
    actionContent?: Snippet;
  }

  interface ListProps {
    items?: readonly ListItem[];
    className?: string;
  }

  let { items = [], className }: ListProps = $props();

  const rootClassName = $derived([styles.list ?? '', className ?? ''].filter((value) => value !== '').join(' '));

  const renderFallback = (value: unknown): string => {
    if (typeof value === 'string' || typeof value === 'number') {
      return `${value}`;
    }

    return '';
  };

  const getItemKey = (item: ListItem): string => {
    const safeDescription = item.description ?? '';
    const safeMeta = item.meta ?? '';

    return `${item.title}-${safeDescription}-${safeMeta}`;
  };
</script>

<div class={rootClassName}>
  {#each items as item (getItemKey(item))}
    <div class={styles.item ?? ''}>
      {#if item.iconContent !== undefined || item.icon !== undefined}
        <div class={styles.icon ?? ''}>
          {#if item.iconContent !== undefined}
            {@render item.iconContent()}
          {:else}
            {renderFallback(item.icon)}
          {/if}
        </div>
      {/if}

      <div class={styles.content ?? ''}>
        <div class={styles.title ?? ''}>{item.title}</div>
        {#if item.description !== undefined && item.description !== ''}
          <div class={styles.description ?? ''}>{item.description}</div>
        {/if}
      </div>

      {#if item.meta !== undefined && item.meta !== ''}
        <div class={styles.meta ?? ''}>{item.meta}</div>
      {/if}

      {#if item.actionContent !== undefined || item.action !== undefined}
        <div class={styles.action ?? ''}>
          {#if item.actionContent !== undefined}
            {@render item.actionContent()}
          {:else}
            {renderFallback(item.action)}
          {/if}
        </div>
      {/if}
    </div>
  {/each}
</div>

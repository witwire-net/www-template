<svelte:options runes={true} />

<script lang="ts">
  import RenderableContent from '@ui/components/app/RenderableContent.svelte';

  import type { Renderable } from '@ui/components/app/shared';

  import styles from './ActivityFeed.module.scss';

  type ActivityItem = {
    avatar?: Renderable;
    description?: string;
    time?: string;
    title: string;
  };

  type Props = {
    items: ActivityItem[];
  };

  let { items }: Props = $props();

  function getItemKey(item: ActivityItem): string {
    const details = item.description ?? '';

    return `${item.title}-${details}`;
  }
</script>

<div class={styles.feed ?? ''}>
  {#each items as item (getItemKey(item))}
    <div class={styles.item ?? ''}>
      {#if item.avatar !== undefined && item.avatar !== null}
        <div class={styles.avatar ?? ''}>
          <RenderableContent value={item.avatar} />
        </div>
      {/if}
      <div class={styles.content ?? ''}>
        <div class={styles.title ?? ''}>{item.title}</div>
        {#if item.description !== undefined && item.description !== ''}
          <div class={styles.description ?? ''}>{item.description}</div>
        {/if}
      </div>
      {#if item.time !== undefined && item.time !== ''}
        <div class={styles.time ?? ''}>{item.time}</div>
      {/if}
    </div>
  {/each}
</div>

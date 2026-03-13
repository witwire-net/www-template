<svelte:options runes={true} />

<script lang="ts">
  import { joinClassNames } from '@ui/components/app/shared';

  import styles from './NotificationCenter.module.scss';

  type NotificationItem = {
    description?: string;
    read?: boolean;
    time?: string;
    title: string;
  };

  type Props = {
    items: NotificationItem[];
  };

  let { items }: Props = $props();

  function getNotificationKey(item: NotificationItem): string {
    const description = item.description ?? '';

    return `${item.title}-${item.time ?? ''}-${description}`;
  }
</script>

<div class={styles.center ?? ''}>
  {#each items as item (getNotificationKey(item))}
    <div class={joinClassNames(styles.item ?? '', item.read === true ? (styles.read ?? '') : undefined)}>
      <div class={styles.title ?? ''}>{item.title}</div>
      {#if item.description !== undefined && item.description !== ''}
        <div class={styles.description ?? ''}>{item.description}</div>
      {/if}
      {#if item.time !== undefined && item.time !== ''}
        <div class={styles.time ?? ''}>{item.time}</div>
      {/if}
    </div>
  {/each}
</div>

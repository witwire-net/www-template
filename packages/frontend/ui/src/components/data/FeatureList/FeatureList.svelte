<script lang="ts">
  import Collection from '@ui/components/organisms/Collection/Collection.svelte';

  import styles from './FeatureList.module.scss';

  interface FeatureListItem {
    title: string;
    description?: string;
  }

  interface FeatureListProps {
    items?: readonly FeatureListItem[];
    columns?: 1 | 2 | 3 | 4;
    className?: string;
  }

  let { items = [], columns = 2, className }: FeatureListProps = $props();
</script>

{#snippet featureItem(item: FeatureListItem)}
  <div class={styles.title ?? ''}>{item.title}</div>
  {#if item.description !== undefined && item.description !== ''}
    <div class={styles.description ?? ''}>{item.description}</div>
  {/if}
{/snippet}

<Collection
  {items}
  {columns}
  {className}
  itemClassName={styles.item ?? ''}
  getKey={(item, index) => `${item.title}-${String(index)}`}
  renderItem={featureItem}
></Collection>

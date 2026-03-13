<script lang="ts">
  import type { Snippet } from 'svelte';

  import Collection from '@ui/components/organisms/Collection/Collection.svelte';
  import InfoCard from '@ui/components/molecules/InfoCard/InfoCard.svelte';

  interface ArticleCardItem {
    title: string;
    excerpt: string;
    date: string;
    tag?: string;
    action?: unknown;
    actionContent?: Snippet;
  }

  interface ArticleCardListProps {
    items?: readonly ArticleCardItem[];
    className?: string;
  }

  let { items = [], className }: ArticleCardListProps = $props();

  const buildMeta = (item: ArticleCardItem): string => {
    return item.tag !== undefined && item.tag !== '' ? `${item.date} | ${item.tag}` : item.date;
  };
</script>

{#snippet articleCard(item: ArticleCardItem)}
  <InfoCard
    title={item.title}
    description={item.excerpt}
    meta={buildMeta(item)}
    action={item.action}
    actionContent={item.actionContent}
  />
{/snippet}

<Collection
  {items}
  columns={3}
  {className}
  getKey={(item, index) => `${item.title}-${String(index)}`}
  renderItem={articleCard}
></Collection>

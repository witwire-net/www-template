<script lang="ts">
  import type { Snippet } from 'svelte';

  import Collection from '@ui/components/organisms/Collection/Collection.svelte';
  import InfoCard from '@ui/components/molecules/InfoCard/InfoCard.svelte';

  interface ShowcaseCardItem {
    title: string;
    description: string;
    industry?: string;
    action?: unknown;
    actionContent?: Snippet;
  }

  interface ShowcaseCardListProps {
    items?: readonly ShowcaseCardItem[];
    className?: string;
  }

  let { items = [], className }: ShowcaseCardListProps = $props();
</script>

<Collection
  {items}
  columns={2}
  {className}
  getKey={(item, index) => `${item.title}-${String(index)}`}
  renderItem={showcaseCard}
></Collection>

{#snippet showcaseCard(item: ShowcaseCardItem)}
  <InfoCard
    title={item.title}
    description={item.description}
    meta={item.industry}
    action={item.action}
    actionContent={item.actionContent}
  ></InfoCard>
{/snippet}

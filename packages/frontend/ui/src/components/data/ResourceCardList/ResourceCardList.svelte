<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { IconCircle } from '@tabler/icons-svelte';

  import Collection from '@ui/components/organisms/Collection/Collection.svelte';
  import InfoCard from '@ui/components/molecules/InfoCard/InfoCard.svelte';

  interface ResourceCardItem {
    title: string;
    description: string;
    meta?: unknown;
    icon?: typeof IconCircle;
    action?: unknown;
    metaContent?: Snippet;
    iconContent?: Snippet;
    actionContent?: Snippet;
  }

  interface ResourceCardListProps {
    items?: readonly ResourceCardItem[];
    className?: string;
  }

  let { items = [], className }: ResourceCardListProps = $props();
</script>

<Collection
  {items}
  columns={2}
  {className}
  getKey={(item, index) => `${item.title}-${String(index)}`}
  renderItem={resourceCard}
></Collection>

{#snippet resourceCard(item: ResourceCardItem)}
  <InfoCard
    title={item.title}
    description={item.description}
    meta={item.meta}
    icon={item.icon}
    action={item.action}
    metaContent={item.metaContent}
    iconContent={item.iconContent}
    actionContent={item.actionContent}
  ></InfoCard>
{/snippet}

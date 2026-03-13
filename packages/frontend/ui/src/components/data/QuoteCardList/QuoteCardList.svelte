<script lang="ts">
  import type { Snippet } from 'svelte';

  import Avatar from '@ui/components/atoms/Avatar/Avatar.svelte';
  import Card from '@ui/components/molecules/Card/Card.svelte';
  import CardBody from '@ui/components/molecules/Card/CardBody.svelte';
  import Collection from '@ui/components/organisms/Collection/Collection.svelte';

  import styles from './QuoteCardList.module.scss';

  interface QuoteCardItem {
    quote: string;
    name: string;
    role?: string;
    avatarSrc?: string;
    avatarAlt?: string;
    avatarContent?: Snippet;
  }

  interface QuoteCardListProps {
    items?: readonly QuoteCardItem[];
    columns?: 1 | 2 | 3;
    className?: string;
  }

  let { items = [], columns = 3, className }: QuoteCardListProps = $props();
</script>

<Collection
  {items}
  {columns}
  {className}
  getKey={(item, index) => `${item.quote}-${String(index)}`}
  renderItem={quoteCard}
></Collection>

{#snippet quoteCard(item: QuoteCardItem)}
  <Card className={styles.card ?? ''}>
    <CardBody className={styles.body ?? ''}>
      <p class={styles.quote ?? ''}>{item.quote}</p>
      <div class={styles.profile ?? ''}>
        {#if item.avatarContent !== undefined}
          {@render item.avatarContent()}
        {:else}
          <Avatar name={item.name} src={item.avatarSrc} alt={item.avatarAlt ?? item.name} size="sm" />
        {/if}

        <div class={styles.identity ?? ''}>
          <div class={styles.name ?? ''}>{item.name}</div>
          {#if item.role !== undefined && item.role !== ''}
            <div class={styles.role ?? ''}>{item.role}</div>
          {/if}
        </div>
      </div>
    </CardBody>
  </Card>
{/snippet}

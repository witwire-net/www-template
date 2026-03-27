<script lang="ts">
  import Collection from '@ui/components/organisms/Collection/Collection.svelte';

  import styles from './CollectionShowcase.module.scss';

  type CollectionItem = {
    id: string;
    title: string;
    details: string;
    tone: string;
  };

  const items: readonly CollectionItem[] = [
    {
      id: 'analytics',
      title: 'Analytics',
      details: 'Weekly funnel snapshots and conversion deltas.',
      tone: 'var(--color-warning)',
    },
    {
      id: 'ops',
      title: 'Operations',
      details: 'Runbooks, alerts, and deployment health in one place.',
      tone: 'var(--color-success)',
    },
    {
      id: 'billing',
      title: 'Billing',
      details: 'Invoice recovery, plan changes, and payment follow-up.',
      tone: 'var(--color-info)',
    },
    {
      id: 'eng',
      title: 'Engineering',
      details: 'Release notes, backlog triage, and QA sign-off.',
      tone: 'var(--color-primary)',
    },
  ];
</script>

{#snippet card(item: CollectionItem, index: number)}
  <article class={styles.panel} style={`--_tone: ${item.tone};`}>
    <div class={styles.header}>
      <strong class={styles.title}>{item.title}</strong>
      <span class={styles.badge}>
        {index + 1}
      </span>
    </div>
    <p class={styles.details}>{item.details}</p>
  </article>
{/snippet}

<div class={styles.frame}>
  <Collection items={items} columns={2} getKey={(item) => item.id} renderItem={card} />
</div>

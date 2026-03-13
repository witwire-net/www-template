<script lang="ts">
  import Collection from '@ui/components/organisms/Collection/Collection.svelte';

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
      tone: '#f97316',
    },
    {
      id: 'ops',
      title: 'Operations',
      details: 'Runbooks, alerts, and deployment health in one place.',
      tone: '#0f766e',
    },
    {
      id: 'billing',
      title: 'Billing',
      details: 'Invoice recovery, plan changes, and payment follow-up.',
      tone: '#2563eb',
    },
    {
      id: 'eng',
      title: 'Engineering',
      details: 'Release notes, backlog triage, and QA sign-off.',
      tone: '#7c3aed',
    },
  ];
</script>

{#snippet card(item: CollectionItem, index: number)}
  <article
    style={`padding: 20px; border-radius: 16px; border: 1px solid color-mix(in srgb, ${item.tone} 20%, #d4d4d8); background: linear-gradient(180deg, color-mix(in srgb, ${item.tone} 10%, white), white); display: grid; gap: 10px; min-height: 164px;`}
  >
    <div style="display: flex; align-items: center; justify-content: space-between; gap: 12px;">
      <strong style="font-size: 1rem;">{item.title}</strong>
      <span
        style={`display: inline-flex; min-width: 32px; height: 32px; align-items: center; justify-content: center; border-radius: 999px; background: ${item.tone}; color: white; font-size: 0.75rem; font-weight: 700;`}
      >
        {index + 1}
      </span>
    </div>
    <p style="margin: 0; color: #52525b; line-height: 1.6;">{item.details}</p>
  </article>
{/snippet}

<div style="width: min(100%, 960px);">
  <Collection items={items} columns={2} getKey={(item) => item.id} renderItem={card} />
</div>

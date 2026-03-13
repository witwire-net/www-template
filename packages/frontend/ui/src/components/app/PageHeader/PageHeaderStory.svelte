<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';
  import Breadcrumb from '@ui/components/navigation/Breadcrumb/Breadcrumb.svelte';

  import PageHeader from './PageHeader.svelte';

  let lastAction = $state('None');

  const items = [
    { label: 'Home', href: '/home' },
    { label: 'Reports', href: '/reports' },
    { label: 'Analytics' },
  ];
</script>

<PageHeader title="Analytics" description="Track usage and engagement across your workspace." meta="Updated 2 hours ago">
  {#snippet breadcrumb()}
    <Breadcrumb {items} />
  {/snippet}

  {#snippet actions()}
    <Button
      variant="ghost"
      size="sm"
      onclick={() => {
        lastAction = 'Export';
      }}
    >
      Export
    </Button>
    <Button
      size="sm"
      onclick={() => {
        lastAction = 'Create report';
      }}
    >
      Create report
    </Button>
  {/snippet}
</PageHeader>

<p style="padding: 1rem 1.5rem; color: #666;">Last action: {lastAction}</p>

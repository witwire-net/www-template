<svelte:options runes={true} />

<script lang="ts">
  import AppSidebar from '@ui/components/navigation/AppSidebar/AppSidebar.svelte';

  import AppShell from './AppShell.svelte';

  const items = [
    {
      label: 'Overview',
      href: '/workspace/overview',
      active: true,
    },
    { label: 'Reports', href: '/workspace/reports' },
    { label: 'Settings', href: '/workspace/settings' },
  ];
</script>

<AppShell>
  {#snippet sidebar()}
    <AppSidebar variant="inline" header="Workspace" {items} />
  {/snippet}

  {#snippet header()}
    <div
      style="padding: 1rem 1.5rem; border-bottom: 1px solid var(--color-border); background: var(--color-surface);"
    >
      App Header
    </div>
  {/snippet}

  <div style="padding: 1.5rem;">App content area</div>
</AppShell>

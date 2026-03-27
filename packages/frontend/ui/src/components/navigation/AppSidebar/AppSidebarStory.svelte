<svelte:options runes={true} />

<script lang="ts">
  import { IconBolt } from '@tabler/icons-svelte';

  import Button from '@ui/components/atoms/Button/Button.svelte';
  import Icon from '@ui/components/atoms/Icon/Icon.svelte';

  import AppSidebar from './AppSidebar.svelte';

  let { customIcon = false }: { customIcon?: boolean } = $props();

  const items = [
    { label: 'Overview', href: '/dashboard/overview', active: true },
    { label: 'Analytics', href: '/dashboard/analytics' },
    { label: 'Customers', href: '/dashboard/customers' },
    { label: 'Settings', href: '/dashboard/settings' },
  ];

  let open = $state(true);
</script>

<div style="padding: 1rem;">
  <Button
    size="sm"
    onclick={() => {
      open = true;
    }}
  >
    Open App Sidebar
  </Button>
</div>

{#if customIcon}
  <AppSidebar header="www-template UI" {items} footer="© 2024 www-template UI" isOpen={open} onClose={() => (open = false)}>
    {#snippet closeIcon()}
      <Icon icon={IconBolt} size={24} title="Close sidebar" />
    {/snippet}
  </AppSidebar>
{:else}
  <AppSidebar header="www-template UI" {items} footer="© 2024 www-template UI" isOpen={open} onClose={() => (open = false)} />
{/if}

<svelte:options runes={true} />

<script lang="ts">
  import { IconBolt } from '@tabler/icons-svelte';

  import Button from '@ui/components/atoms/Button/Button.svelte';
  import Icon from '@ui/components/atoms/Icon/Icon.svelte';

  import SiteHeader from './SiteHeader.svelte';

  let { customIcon = false, customLabels = false }: { customIcon?: boolean; customLabels?: boolean } = $props();

  const links = [
    { label: 'Overview', href: '#overview', active: true },
    { label: 'Analytics', href: '#analytics' },
    { label: 'Settings', href: '#settings' },
  ];

  let lastAction = $state('None');
</script>

{#if customIcon}
  <SiteHeader logo="www-template UI" {links}>
    {#snippet menuIcon()}
      <Icon icon={IconBolt} size={24} title="Open menu" />
    {/snippet}
    {#snippet actions()}
      <Button
        variant="ghost"
        size="sm"
        onclick={() => {
          lastAction = 'Login';
        }}
      >
        Login
      </Button>
      <Button
        size="sm"
        onclick={() => {
          lastAction = 'Get Started';
        }}
      >
        Get Started
      </Button>
    {/snippet}
  </SiteHeader>
{:else}
  <SiteHeader
    logo="www-template UI"
    {links}
    navigationAriaLabel={customLabels ? 'Primary site navigation' : undefined}
    mobileNavigationAriaLabel={customLabels ? 'Mobile site navigation' : undefined}
    menuButtonLabel={customLabels ? 'Open navigation panel' : undefined}
    mobileBackdropLabel={customLabels ? 'Dismiss navigation panel' : undefined}
    closeButtonLabel={customLabels ? 'Close navigation panel' : undefined}
    mobileMenuHeading={customLabels ? 'Site menu' : undefined}
  >
    {#snippet actions()}
      <Button
        variant="ghost"
        size="sm"
        onclick={() => {
          lastAction = 'Login';
        }}
      >
        Login
      </Button>
      <Button
        size="sm"
        onclick={() => {
          lastAction = 'Get Started';
        }}
      >
        Get Started
      </Button>
    {/snippet}
  </SiteHeader>
{/if}

<div style="padding: 1rem 1.5rem; color: var(--color-text-muted);">Last action: {lastAction}</div>
<div id="overview" style="padding: 0 1.5rem 2rem;">Overview section</div>
<div id="analytics" style="padding: 0 1.5rem 2rem;">Analytics section</div>
<div id="settings" style="padding: 0 1.5rem 2rem;">Settings section</div>

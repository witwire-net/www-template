<svelte:options runes={true} />

<script lang="ts">
  import { IconBolt } from '@tabler/icons-svelte';

  import Button from '@ui/components/atoms/Button/Button.svelte';
  import Icon from '@ui/components/atoms/Icon/Icon.svelte';

  import Header from './Header.svelte';

  let { customIcon = false }: { customIcon?: boolean } = $props();

  const links = [
    { label: 'Overview', href: '/overview', active: true },
    { label: 'Analytics', href: '/analytics' },
    { label: 'Settings', href: '/settings' },
  ];

  let menuClickCount = $state(0);
  let lastAction = $state('None');
</script>

{#if customIcon}
  <Header
    variant="app"
    logo="www-template UI"
    {links}
    onMenuClick={() => {
      menuClickCount += 1;
    }}
  >
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
  </Header>
{:else}
  <Header
    variant="app"
    logo="www-template UI"
    {links}
    onMenuClick={() => {
      menuClickCount += 1;
    }}
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
  </Header>
{/if}

<div style="padding: 1rem 1.5rem; color: var(--color-text-muted);">
  Menu clicked: {menuClickCount} times
  <br />
  Last action: {lastAction}
</div>

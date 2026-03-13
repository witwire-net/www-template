<svelte:options runes={true} />

<script lang="ts">
  import Header from './Header.svelte';
  import MenuButton from './MenuButton.svelte';

  import type { Renderable } from '@ui/components/navigation/shared';

  type HeaderLink = {
    active?: boolean;
    href: string;
    label: string;
  };

  type Props = {
    actions?: Renderable;
    className?: string;
    isMinimized?: boolean;
    links?: HeaderLink[];
    logo?: Renderable;
    menuIcon?: Renderable;
    onMenuClick?: () => void;
  };

  let {
    logo = undefined,
    links = [],
    actions = undefined,
    className = undefined,
    isMinimized = false,
    onMenuClick = undefined,
    menuIcon = undefined,
  }: Props = $props();
</script>

<Header {logo} {links} {actions} {className} {isMinimized}>
  {#snippet leftSlot()}
    {#if onMenuClick !== undefined}
      <MenuButton onClick={onMenuClick} icon={menuIcon} />
    {/if}
  {/snippet}
</Header>

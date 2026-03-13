<script lang="ts">
  import { IconShieldCheckFilled } from '@tabler/icons-svelte';
  import type { Snippet } from 'svelte';

  import TrustBadge from '@ui/components/molecules/TrustBadge/TrustBadge.svelte';

  interface Props {
    label?: string;
    description?: string;
    icon?: typeof IconShieldCheckFilled;
    iconContent?: Snippet;
  }

  let {
    label = 'SOC2',
    description = 'Certified compliance',
    icon = IconShieldCheckFilled,
  }: Props = $props();
</script>

<TrustBadge {label} {description} {icon} />

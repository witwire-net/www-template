<script lang="ts">
  import { IconBolt, IconDeviceMobile, IconPalette } from '@tabler/icons-svelte';

  import Icon from './Icon.svelte';
</script>

<div style="display: flex; align-items: center; gap: 12px;">
  <Icon icon={IconBolt} size={24} title="Fast" />
  <Icon icon={IconPalette} size={24} title="Design" />
  <Icon icon={IconDeviceMobile} size={24} title="Responsive" />
</div>

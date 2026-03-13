<script lang="ts">
  import Avatar from './Avatar.svelte';

  let { name = 'Alex Morgan' }: { name?: string } = $props();
</script>

<div style="display: flex; gap: 16px; align-items: flex-end;">
  <Avatar {name} size="xs" status="online" />
  <Avatar {name} size="sm" status="offline" />
  <Avatar {name} size="md" status="busy" />
  <Avatar {name} size="lg" status="away" />
  <Avatar {name} size="xl" status="online" />
</div>

<script lang="ts">
  import Stack from './Stack.svelte';

  let { direction = 'column' }: { direction?: 'row' | 'column' } = $props();
</script>

<Stack {direction} gap="sm">
  <div style="padding: 12px 16px; border-radius: 12px; border: 1px solid var(--color-border, #e6e6e6);">Item 1</div>
  <div style="padding: 12px 16px; border-radius: 12px; border: 1px solid var(--color-border, #e6e6e6);">Item 2</div>
  <div style="padding: 12px 16px; border-radius: 12px; border: 1px solid var(--color-border, #e6e6e6);">Item 3</div>
</Stack>

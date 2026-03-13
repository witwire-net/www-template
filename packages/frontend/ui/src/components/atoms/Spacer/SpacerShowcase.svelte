<script lang="ts">
  import Spacer from './Spacer.svelte';

  let { axis = 'vertical' }: { axis?: 'vertical' | 'horizontal' } = $props();
</script>

{#if axis === 'vertical'}
  <div>
    <div style="padding: 8px 12px; border: 1px solid var(--color-border, #e6e6e6); border-radius: 12px;">Top</div>
    <Spacer size="lg" axis="vertical" />
    <div style="padding: 8px 12px; border: 1px solid var(--color-border, #e6e6e6); border-radius: 12px;">Bottom</div>
  </div>
{:else}
  <div style="display: flex; align-items: center;">
    <div style="padding: 8px 12px; border: 1px solid var(--color-border, #e6e6e6); border-radius: 12px;">Left</div>
    <Spacer size="lg" axis="horizontal" />
    <div style="padding: 8px 12px; border: 1px solid var(--color-border, #e6e6e6); border-radius: 12px;">Right</div>
  </div>
{/if}

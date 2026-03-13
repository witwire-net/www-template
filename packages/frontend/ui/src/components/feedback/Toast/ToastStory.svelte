<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import Toast from './Toast.svelte';

  let visible = $state(false);
</script>

<div style="padding: 1.5rem;">
  <Button
    onclick={() => {
      visible = true;
    }}
  >
    Show Toast
  </Button>
  {#if visible}
    <div style="margin-top: 1rem;">
      <Toast
        title="Deployment started"
        description="We will notify you when it is live."
        onClose={() => {
          visible = false;
        }}
      />
    </div>
  {/if}
</div>

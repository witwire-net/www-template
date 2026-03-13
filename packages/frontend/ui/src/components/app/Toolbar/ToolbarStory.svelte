<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import Toolbar from './Toolbar.svelte';

  let lastAction = $state('None');
</script>

<Toolbar>
  {#snippet left()}
    <div>Filters</div>
  {/snippet}

  {#snippet right()}
    <Button
      variant="ghost"
      size="sm"
      onclick={() => {
        lastAction = 'Reset';
      }}
    >
      Reset
    </Button>
    <Button
      size="sm"
      onclick={() => {
        lastAction = 'Apply';
      }}
    >
      Apply
    </Button>
  {/snippet}
</Toolbar>

<p style="padding: 1rem 1.5rem; color: #666;">Last action: {lastAction}</p>

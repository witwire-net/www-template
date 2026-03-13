<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import FilterBar from './FilterBar.svelte';

  let lastAction = $state('None');
</script>

<FilterBar>
  {#snippet filters()}
    <div>Filter controls</div>
  {/snippet}

  {#snippet actions()}
    <Button
      size="sm"
      variant="ghost"
      onclick={() => {
        lastAction = 'Clear';
      }}
    >
      Clear
    </Button>
    <Button
      size="sm"
      onclick={() => {
        lastAction = 'Save view';
      }}
    >
      Save view
    </Button>
  {/snippet}
</FilterBar>

<p style="padding: 1rem 1.5rem; color: #666;">Last action: {lastAction}</p>

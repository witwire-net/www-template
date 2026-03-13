<svelte:options runes={true} />

<script lang="ts">
  import SearchBar from './SearchBar.svelte';

  let lastSearch = $state('');
</script>

<div style="width: min(32rem, 100%); display: grid; gap: 0.75rem;">
  <SearchBar
    placeholder="Search projects"
    onSearch={(value) => {
      lastSearch = value;
    }}
  />

  <p style="margin: 0; color: #666;">Latest query: {lastSearch === '' ? 'None' : lastSearch}</p>
</div>

<svelte:options runes={true} />

<script lang="ts">
  import { getTextContent, isSnippet, type Renderable } from './shared';

  let { value }: { value?: Renderable } = $props();
</script>

{#if value !== undefined && value !== null}
  {#if isSnippet(value)}
    {@render value()}
  {:else}
    {getTextContent(value)}
  {/if}
{/if}

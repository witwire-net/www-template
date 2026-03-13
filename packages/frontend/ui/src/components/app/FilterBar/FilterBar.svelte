<svelte:options runes={true} />

<script lang="ts">
  import type { Renderable } from '@ui/components/app/shared';

  import RenderableContent from '@ui/components/app/RenderableContent.svelte';

  import styles from './FilterBar.module.scss';

  type Props = {
    actions?: Renderable;
    filters?: Renderable;
  };

  let { filters = undefined, actions = undefined }: Props = $props();
</script>

<div class={styles.filterBar ?? ''}>
  <div class={styles.filters ?? ''}>
    <RenderableContent value={filters} />
  </div>
  <div class={styles.actions ?? ''}>
    <RenderableContent value={actions} />
  </div>
</div>

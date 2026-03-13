<svelte:options runes={true} />

<script lang="ts">
  import type { Renderable } from '@ui/components/app/shared';

  import RenderableContent from '@ui/components/app/RenderableContent.svelte';

  import styles from './Toolbar.module.scss';

  type Props = {
    left?: Renderable;
    right?: Renderable;
  };

  let { left = undefined, right = undefined }: Props = $props();
</script>

<div class={styles.toolbar ?? ''}>
  <div class={styles.left ?? ''}>
    <RenderableContent value={left} />
  </div>
  <div class={styles.right ?? ''}>
    <RenderableContent value={right} />
  </div>
</div>

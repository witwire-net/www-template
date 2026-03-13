<svelte:options runes={true} />

<script lang="ts">
  import type { Renderable } from '@ui/components/app/shared';

  import RenderableContent from '@ui/components/app/RenderableContent.svelte';

  import styles from './AppShell.module.scss';

  type Props = {
    children?: Renderable;
    header?: Renderable;
    sidebar?: Renderable;
  };

  let { header = undefined, sidebar = undefined, children = undefined }: Props = $props();
</script>

<div class={styles.shell ?? ''}>
  {#if sidebar !== undefined && sidebar !== null}
    <aside class={styles.sidebar ?? ''}>
      <RenderableContent value={sidebar} />
    </aside>
  {/if}
  <div class={styles.main ?? ''}>
    {#if header !== undefined && header !== null}
      <div class={styles.header ?? ''}>
        <RenderableContent value={header} />
      </div>
    {/if}
    <div class={styles.content ?? ''}>
      <RenderableContent value={children} />
    </div>
  </div>
</div>

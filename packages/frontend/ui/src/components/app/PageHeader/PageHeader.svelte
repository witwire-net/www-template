<svelte:options runes={true} />

<script lang="ts">
  import type { Renderable } from '@ui/components/app/shared';

  import RenderableContent from '@ui/components/app/RenderableContent.svelte';

  import styles from './PageHeader.module.scss';

  type Props = {
    actions?: Renderable;
    breadcrumb?: Renderable;
    description?: string;
    meta?: Renderable;
    title: string;
  };

  let {
    title,
    description = undefined,
    breadcrumb = undefined,
    actions = undefined,
    meta = undefined,
  }: Props = $props();
</script>

<div class={styles.header ?? ''}>
  {#if breadcrumb !== undefined && breadcrumb !== null}
    <div class={styles.breadcrumb ?? ''}>
      <RenderableContent value={breadcrumb} />
    </div>
  {/if}
  <div class={styles.top ?? ''}>
    <div class={styles.titles ?? ''}>
      <h1 class={styles.title ?? ''}>{title}</h1>
      {#if description !== undefined && description !== ''}
        <p class={styles.description ?? ''}>{description}</p>
      {/if}
    </div>
    {#if actions !== undefined && actions !== null}
      <div class={styles.actions ?? ''}>
        <RenderableContent value={actions} />
      </div>
    {/if}
  </div>
  {#if meta !== undefined && meta !== null}
    <div class={styles.meta ?? ''}>
      <RenderableContent value={meta} />
    </div>
  {/if}
</div>

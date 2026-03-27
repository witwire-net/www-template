<svelte:options runes={true} />

<script lang="ts">
  import { getTextContent, isSnippet, joinClassNames, type Renderable } from '@ui/components/navigation/shared';

  import styles from './DashboardLayout.module.scss';

  type Props = {
    children?: Renderable;
    className?: string;
    header?: Renderable;
    /** `<main>` の id。スキップナビのリンク先と一致させる（デフォルト: 'main-content'） */
    mainId?: string;
    sidebar?: Renderable;
  };

  let {
    children = undefined,
    className = undefined,
    header = undefined,
    mainId = 'main-content',
    sidebar = undefined,
  }: Props = $props();

  const layoutClassName = $derived(joinClassNames(styles.layout ?? '', className));
</script>

<!-- スキップナビゲーション（キーボード・スクリーンリーダー対応） -->
<a href={`#${mainId}`} class={styles.skipNav ?? ''}>Skip to main content</a>

<div class={layoutClassName}>
  {#if sidebar !== undefined && sidebar !== null}
    {#if isSnippet(sidebar)}
      {@render sidebar()}
    {:else}
      {getTextContent(sidebar)}
    {/if}
  {/if}

  <div class={styles.mainWrapper ?? ''}>
    {#if header !== undefined && header !== null}
      <div class={styles.headerWrapper ?? ''}>
        {#if isSnippet(header)}
          {@render header()}
        {:else}
          {getTextContent(header)}
        {/if}
      </div>
    {/if}

    <main id={mainId} tabindex="-1" class={styles.content ?? ''}>
      {#if children !== undefined && children !== null}
        {#if isSnippet(children)}
          {@render children()}
        {:else}
          {getTextContent(children)}
        {/if}
      {/if}
    </main>
  </div>
</div>

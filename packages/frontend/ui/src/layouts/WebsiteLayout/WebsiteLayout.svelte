<svelte:options runes={true} />

<script lang="ts">
  import { getTextContent, isSnippet, joinClassNames, type Renderable } from '@ui/components/navigation/shared';

  import styles from './WebsiteLayout.module.scss';

  type Props = {
    children?: Renderable;
    className?: string;
    footer?: Renderable;
    header?: Renderable;
    /** `<main>` の id。スキップナビのリンク先と一致させる（デフォルト: 'main-content'） */
    mainId?: string;
    /**
     * true のとき header を fixed 配置にし、コンテンツ上に重ねる。
     * ヒーローセクションがヘッダーを透過させたい場合に使用する。
     */
    overlayHeader?: boolean;
  };

  let {
    children = undefined,
    className = undefined,
    footer = undefined,
    header = undefined,
    mainId = 'main-content',
    overlayHeader = false,
  }: Props = $props();

  const layoutClassName = $derived(joinClassNames(styles.layout ?? '', className));
  const headerClassName = $derived(
    joinClassNames(styles.headerWrapper ?? '', overlayHeader ? (styles.overlay ?? '') : undefined)
  );
  const contentClassName = $derived(
    joinClassNames(styles.content ?? '', overlayHeader ? (styles.contentWithOverlay ?? '') : undefined)
  );
</script>

<!-- スキップナビゲーション（キーボード・スクリーンリーダー対応） -->
<a href={`#${mainId}`} class={styles.skipNav ?? ''}>Skip to main content</a>

<div class={layoutClassName}>
  <div class={headerClassName}>
    {#if header !== undefined && header !== null}
      {#if isSnippet(header)}
        {@render header()}
      {:else}
        {getTextContent(header)}
      {/if}
    {/if}
  </div>

  <main id={mainId} tabindex="-1" class={contentClassName}>
    {#if children !== undefined && children !== null}
      {#if isSnippet(children)}
        {@render children()}
      {:else}
        {getTextContent(children)}
      {/if}
    {/if}
  </main>

  {#if footer !== undefined && footer !== null}
    {#if isSnippet(footer)}
      {@render footer()}
    {:else}
      {getTextContent(footer)}
    {/if}
  {/if}
</div>

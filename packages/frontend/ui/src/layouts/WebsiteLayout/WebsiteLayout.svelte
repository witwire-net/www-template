<svelte:options runes={true} />

<script lang="ts">
  import { getTextContent, isSnippet, joinClassNames, type Renderable } from '@ui/components/navigation/shared';

  import styles from './WebsiteLayout.module.scss';

  type Props = {
    children?: Renderable;
    className?: string;
    footer?: Renderable;
    header?: Renderable;
    overlayHeader?: boolean;
  };

  let {
    children = undefined,
    className = undefined,
    footer = undefined,
    header = undefined,
    overlayHeader = false,
  }: Props = $props();

  const layoutClassName = $derived(joinClassNames(styles.layout ?? '', className));
  const headerClassName = $derived(
    joinClassNames(styles.headerWrapper ?? '', overlayHeader ? (styles.overlay ?? '') : undefined)
  );
</script>

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

  <main class={styles.content ?? ''}>
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

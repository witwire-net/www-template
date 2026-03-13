<svelte:options runes={true} />

<script lang="ts">
  import { getTextContent, isSnippet, joinClassNames, type Renderable } from '@ui/components/navigation/shared';

  import styles from './DashboardLayout.module.scss';

  type Props = {
    children?: Renderable;
    className?: string;
    header?: Renderable;
    sidebar?: Renderable;
  };

  let {
    children = undefined,
    className = undefined,
    header = undefined,
    sidebar = undefined,
  }: Props = $props();

  const layoutClassName = $derived(joinClassNames(styles.layout ?? '', className));
</script>

<div class={layoutClassName}>
  {#if sidebar !== undefined && sidebar !== null}
    {#if isSnippet(sidebar)}
      {@render sidebar()}
    {:else}
      {getTextContent(sidebar)}
    {/if}
  {/if}

  <div class={styles.mainWrapper ?? ''}>
    <div class={styles.headerWrapper ?? ''}>
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
  </div>
</div>

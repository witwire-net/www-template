<svelte:options runes={true} />

<script lang="ts">
  import Icon from '@ui/components/atoms/Icon/Icon.svelte';

  import { getTextContent, isSnippet, type Renderable } from '@ui/components/navigation/shared';

  import styles from './Header.module.scss';

  type Props = {
    ariaLabel?: string;
    icon?: Renderable;
    onClick?: () => void;
  };

  let { onClick = undefined, icon = undefined, ariaLabel = 'Menu' }: Props = $props();
</script>

<button type="button" class={styles.menuButton ?? ''} onclick={onClick} aria-label={ariaLabel}>
  {#if icon !== undefined && icon !== null}
    {#if isSnippet(icon)}
      {@render icon()}
    {:else}
      {getTextContent(icon)}
    {/if}
  {:else}
    <Icon className={styles.menuIcon ?? ''} />
  {/if}
</button>

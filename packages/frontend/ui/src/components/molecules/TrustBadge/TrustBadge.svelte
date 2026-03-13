<script lang="ts">
  import type { IconCircle } from '@tabler/icons-svelte';
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import Icon from '@ui/components/atoms/Icon/Icon.svelte';

  import styles from './TrustBadge.module.scss';

  interface TrustBadgeProps extends HTMLAttributes<HTMLDivElement> {
    label?: string;
    description?: string;
    icon?: typeof IconCircle;
    iconContent?: Snippet;
  }

  let { label = '', description, icon, iconContent, ...restProps }: TrustBadgeProps = $props();

  const hasDescription = $derived(description !== undefined && description !== '');
  const hasIcon = $derived(iconContent !== undefined || icon !== undefined);
</script>

<div class={styles.badge ?? ''} {...restProps}>
  {#if hasIcon}
    <div class={styles.icon ?? ''}>
      {#if iconContent !== undefined}
        {@render iconContent()}
      {:else if icon !== undefined}
        <Icon {icon} size={16} />
      {/if}
    </div>
  {/if}

  <div class={styles.content ?? ''}>
    <div class={styles.label ?? ''}>{label}</div>
    {#if hasDescription}
      <div class={styles.description ?? ''}>{description}</div>
    {/if}
  </div>
</div>

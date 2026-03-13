<script lang="ts">
  import type { Snippet } from 'svelte';

  import Badge from '@ui/components/atoms/Badge/Badge.svelte';
  import Card from '@ui/components/molecules/Card/Card.svelte';

  import styles from './IntegrationCard.module.scss';

  interface IntegrationCardProps {
    name: string;
    description?: string;
    status?: 'connected' | 'available' | 'beta';
    icon?: Snippet | string | number;
    action?: Snippet | string | number;
  }

  let {
    name,
    description,
    status = 'available',
    icon,
    action,
  }: IntegrationCardProps = $props();

  const isSnippet = (value: string | number | Snippet | undefined): value is Snippet =>
    typeof value === 'function';

  const renderValue = (value: string | number | Snippet | undefined): string => {
    if (typeof value === 'string' || typeof value === 'number') {
      return `${value}`;
    }

    return '';
  };

  const hasDescription = $derived(description !== undefined && description !== '');
  const hasIcon = $derived(icon !== undefined);
  const hasAction = $derived(action !== undefined);
  const variant = $derived.by(() => {
    if (status === 'connected') {
      return 'success';
    }

    if (status === 'beta') {
      return 'warning';
    }

    return 'neutral';
  });
</script>

<Card className={styles.card}>
  <div class={styles.header}>
    <div class={styles.icon}>
      {#if hasIcon}
        {#if isSnippet(icon)}
          {@render icon()}
        {:else}
          {renderValue(icon)}
        {/if}
      {/if}
    </div>
    <Badge {variant}>{status}</Badge>
  </div>
  <div class={styles.name}>{name}</div>
  {#if hasDescription}
    <div class={styles.description}>{description}</div>
  {/if}
  {#if hasAction}
    <div class={styles.action}>
      {#if isSnippet(action)}
        {@render action()}
      {:else}
        {renderValue(action)}
      {/if}
    </div>
  {/if}
</Card>

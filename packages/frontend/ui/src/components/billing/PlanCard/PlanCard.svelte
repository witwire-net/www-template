<script lang="ts">
  import Card from '@ui/components/molecules/Card/Card.svelte';

  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './PlanCard.module.scss';

  type PlanAction = Snippet | string;

  interface PlanCardProps extends HTMLAttributes<HTMLDivElement> {
    name: string;
    price: string;
    interval?: string;
    description?: string;
    features?: readonly string[];
    highlight?: boolean;
    action?: PlanAction;
    className?: string;
  }

  let {
    name,
    price,
    interval,
    description,
    features = [],
    highlight = false,
    action,
    className,
    ...restProps
  }: PlanCardProps = $props();

  const rootClassName = $derived([
    styles.card ?? '',
    highlight ? (styles.highlight ?? '') : '',
    className ?? '',
  ]
    .filter((value) => value !== '')
    .join(' '));
  const hasInterval = $derived(interval !== undefined && interval !== '');
  const hasDescription = $derived(description !== undefined && description !== '');
  const hasFeatures = $derived(features.length > 0);
</script>

<Card className={rootClassName} {...restProps}>
  <div class={styles.header ?? ''}>
    <div class={styles.name ?? ''}>{name}</div>
    <div class={styles.price ?? ''}>
      {price}
      {#if hasInterval}
        <span class={styles.interval ?? ''}>/{interval}</span>
      {/if}
    </div>
    {#if hasDescription}
      <div class={styles.description ?? ''}>{description}</div>
    {/if}
  </div>

  {#if hasFeatures}
    <ul class={styles.features ?? ''}>
      {#each features as feature (feature)}
        <li>{feature}</li>
      {/each}
    </ul>
  {/if}

  {#if typeof action === 'function' || typeof action === 'string'}
    <div class={styles.action ?? ''}>
      {#if typeof action === 'function'}
        {@render action()}
      {:else}
        {action}
      {/if}
    </div>
  {/if}
</Card>

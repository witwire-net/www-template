<script lang="ts">
  import Card from '@ui/components/molecules/Card/Card.svelte';

  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import styles from './CardPaymentMethod.module.scss';

  type PaymentAction = Snippet | string;

  interface CardPaymentMethodProps extends HTMLAttributes<HTMLDivElement> {
    brand: string;
    last4: string;
    expiry: string;
    holder?: string;
    action?: PaymentAction;
    className?: string;
  }

  let {
    brand,
    last4,
    expiry,
    holder,
    action,
    className,
    ...restProps
  }: CardPaymentMethodProps = $props();

  const rootClassName = $derived([styles.card ?? '', className ?? '']
    .filter((value) => value !== '')
    .join(' '));
  const hasHolder = $derived(holder !== undefined && holder !== '');
</script>

<Card className={rootClassName} {...restProps}>
  <div class={styles.brand ?? ''}>{brand}</div>
  <div class={styles.number ?? ''}>•••• {last4}</div>
  <div class={styles.meta ?? ''}>
    <span>{expiry}</span>
    {#if hasHolder}
      <span class={styles.holder ?? ''}>{holder}</span>
    {/if}
  </div>
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

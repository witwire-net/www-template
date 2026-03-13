<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  import Badge from '@ui/components/atoms/Badge/Badge.svelte';

  import styles from './InvoiceList.module.scss';

  type InvoiceAction = Snippet | string | number | null | undefined;
  type InvoiceStatusTone = 'primary' | 'neutral' | 'success' | 'warning' | 'error' | 'info';

  interface InvoiceField {
    label?: string;
    value: string;
  }

  interface InvoiceStatus {
    label: string;
    tone?: InvoiceStatusTone;
  }

  interface InvoiceItem {
    id: string | number;
    number: string;
    title?: string;
    date?: InvoiceField;
    amount: string;
    status: string | InvoiceStatus;
    metadata?: readonly InvoiceField[];
    action?: InvoiceAction;
  }

  interface InvoiceListProps extends HTMLAttributes<HTMLDivElement> {
    invoices?: readonly InvoiceItem[];
    className?: string;
  }

  let { invoices = [], className, ...restProps }: InvoiceListProps = $props();

  const rootClassName = $derived([styles.list ?? '', className ?? '']
    .filter((value) => value !== '')
    .join(' '));

  const hasText = (value: string | undefined): boolean => {
    return value !== undefined && value !== '';
  };

  const hasRenderable = (value: InvoiceAction): boolean => {
    return value !== undefined && value !== null && value !== '';
  };

  const isSnippet = (value: InvoiceAction): value is Snippet => {
    return typeof value === 'function';
  };

  const renderValue = (value: InvoiceAction): string => {
    if (typeof value === 'string' || typeof value === 'number') {
      return String(value);
    }

    return '';
  };

  const resolveStatus = (status: InvoiceItem['status']): InvoiceStatus => {
    if (typeof status === 'string') {
      return { label: status, tone: 'neutral' };
    }

    return {
      label: status.label,
      tone: status.tone ?? 'neutral',
    };
  };
</script>

<div class={rootClassName} {...restProps}>
  {#each invoices as invoice (invoice.id)}
    {@const status = resolveStatus(invoice.status)}
    <div class={styles.row ?? ''}>
      <div class={styles.summary ?? ''}>
        <div class={styles.number ?? ''}>{invoice.number}</div>
        {#if hasText(invoice.title)}
          <div class={styles.title ?? ''}>{invoice.title}</div>
        {/if}
        {#if invoice.metadata !== undefined && invoice.metadata.length > 0}
          <div class={styles.metadata ?? ''}>
            {#each invoice.metadata as item, index (`${invoice.id}-meta-${item.label ?? item.value}-${String(index)}`)}
              <span class={styles.metaItem ?? ''}>
                {#if hasText(item.label)}
                  <span class={styles.metaLabel ?? ''}>{item.label}:</span>
                {/if}
                <span>{item.value}</span>
              </span>
            {/each}
          </div>
        {/if}
      </div>
      <div class={styles.date ?? ''}>
        {#if invoice.date !== undefined}
          {#if hasText(invoice.date.label)}
            <span class={styles.dateLabel ?? ''}>{invoice.date.label}</span>
          {/if}
          <span>{invoice.date.value}</span>
        {/if}
      </div>
      <div class={styles.amount ?? ''}>{invoice.amount}</div>
      <div class={styles.status ?? ''}>
        <Badge variant={status.tone} size="sm">{status.label}</Badge>
      </div>
      {#if hasRenderable(invoice.action)}
        <div class={styles.action ?? ''}>
          {#if isSnippet(invoice.action)}
            {@render invoice.action()}
          {:else}
            {renderValue(invoice.action)}
          {/if}
        </div>
      {/if}
    </div>
  {/each}
</div>

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import InvoiceList from './InvoiceList.svelte';

  const invoices = [
    {
      id: 'invoice-1',
      number: 'INV-1001',
      title: 'Starter workspace renewal',
      date: { label: 'Issued', value: '2025-01-01' },
      amount: '$120.00',
      status: { label: 'Paid', tone: 'success' },
      metadata: [{ label: 'Period', value: 'Jan 2025' }],
      action: downloadInvoice1001,
    },
    {
      id: 'invoice-2',
      number: 'INV-1002',
      title: 'Growth add-on seats',
      date: { label: 'Due', value: '2025-02-01' },
      amount: '$120.00',
      status: { label: 'Pending', tone: 'warning' },
      metadata: [{ label: 'Seats', value: '4 extra' }],
      action: downloadInvoice1002,
    },
  ] as const;
</script>

{#snippet downloadInvoice1001()}
  <Button
    variant="ghost"
    size="sm"
    type="button"
    onclick={() => {
      window.location.hash = 'invoice-1001-downloaded';
    }}
  >
    Download
  </Button>
{/snippet}

{#snippet downloadInvoice1002()}
  <Button
    variant="ghost"
    size="sm"
    type="button"
    onclick={() => {
      window.location.hash = 'invoice-1002-downloaded';
    }}
  >
    Download
  </Button>
{/snippet}

<div style="width: min(100%, 960px);">
  <InvoiceList {invoices} />
</div>

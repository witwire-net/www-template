<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import CardPaymentMethod from './CardPaymentMethod.svelte';

  interface Props {
    brand?: string;
    last4?: string;
    expiry?: string;
    holder?: string;
  }

  let {
    brand = 'Visa',
    last4 = '4242',
    expiry = '06/27',
    holder = 'Jordan Lee',
  }: Props = $props();
</script>

{#snippet updateAction()}
  <Button
    variant="ghost"
    size="sm"
    type="button"
    onclick={() => {
      window.location.hash = 'payment-method-updated';
    }}
  >
    Update
  </Button>
{/snippet}

<div style="width: min(100%, 360px);">
  <CardPaymentMethod {brand} {last4} {expiry} {holder} action={updateAction} />
</div>

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import PlanCard from './PlanCard.svelte';

  interface Props {
    name?: string;
    price?: string;
    interval?: string;
    description?: string;
    features?: string[];
  }

  let {
    name = 'Pro',
    price = '$29',
    interval = 'month',
    description = 'For growing teams that need more control.',
    features = ['Unlimited projects', 'Advanced analytics', 'Priority support'],
  }: Props = $props();
</script>

{#snippet choosePlanAction()}
  <Button
    size="sm"
    type="button"
    onclick={() => {
      window.location.hash = 'plan-pro-selected';
    }}
  >
    Choose plan
  </Button>
{/snippet}

<div style="width: min(100%, 320px);">
  <PlanCard {name} {price} {interval} {description} {features} action={choosePlanAction} />
</div>

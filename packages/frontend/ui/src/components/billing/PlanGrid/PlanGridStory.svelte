<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import PlanGrid from './PlanGrid.svelte';

  const plans = [
    {
      name: 'Starter',
      price: '$0',
      interval: 'month',
      description: 'Best for trying the workspace with a small team.',
      features: ['3 projects', 'Email support', 'Basic reports'],
      action: chooseStarter,
    },
    {
      name: 'Pro',
      price: '$29',
      interval: 'month',
      description: 'Adds deeper analytics and faster support for scaling teams.',
      features: ['Unlimited projects', 'Advanced analytics', 'Priority support'],
      highlight: true,
      action: choosePro,
    },
    {
      name: 'Enterprise',
      price: 'Custom',
      description: 'For organizations that need security reviews and custom rollout plans.',
      features: ['SSO', 'Dedicated onboarding', 'Custom invoicing'],
      action: contactSales,
    },
  ] as const;
</script>

{#snippet chooseStarter()}
  <Button
    variant="outline"
    size="sm"
    type="button"
    onclick={() => {
      window.location.hash = 'starter-plan-selected';
    }}
  >
    Start free
  </Button>
{/snippet}

{#snippet choosePro()}
  <Button
    size="sm"
    type="button"
    onclick={() => {
      window.location.hash = 'pro-plan-selected';
    }}
  >
    Choose Pro
  </Button>
{/snippet}

{#snippet contactSales()}
  <Button
    variant="ghost"
    size="sm"
    type="button"
    onclick={() => {
      window.location.hash = 'enterprise-contact-requested';
    }}
  >
    Contact sales
  </Button>
{/snippet}

<div style="width: min(100%, 1120px);">
  <PlanGrid {plans} />
</div>

<script lang="ts">
  import Collection from '@ui/components/organisms/Collection/Collection.svelte';

  import type { Snippet } from 'svelte';

  import PlanCard from '@ui/components/billing/PlanCard/PlanCard.svelte';

  type PlanAction = Snippet | string;

  interface PlanGridItem {
    name: string;
    price: string;
    interval?: string;
    description?: string;
    features?: readonly string[];
    highlight?: boolean;
    action?: PlanAction;
  }

  interface PlanGridProps {
    plans?: readonly PlanGridItem[];
    className?: string;
  }

  let { plans = [], className }: PlanGridProps = $props();

  const getPlanKey = (plan: PlanGridItem, index: number): string | number => {
    return plan.name !== '' ? plan.name : index;
  };
</script>

{#snippet renderPlan(plan: PlanGridItem)}
  <PlanCard
    name={plan.name}
    price={plan.price}
    interval={plan.interval}
    description={plan.description}
    features={plan.features}
    highlight={plan.highlight}
    action={plan.action}
  />
{/snippet}

<Collection items={plans} columns={3} className={className} getKey={getPlanKey} renderItem={renderPlan} />
